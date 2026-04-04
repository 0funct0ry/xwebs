package cmd

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/config"
	"github.com/0funct0ry/xwebs/internal/repl"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type connectClientContext struct {
	conn            *ws.Connection
	dialChan        chan string
	tmplEngine      *template.Engine
	repl            *repl.REPL
	originalURL     string
	originalHeaders map[string]string // Key: Template
	originalAuth      string            // Auth template
	originalToken     string            // Token template
	automationPending bool              // Flag to avoid premature --once exit
	receivedOnce      chan struct{}     // Pulse when --once condition is met
}

func (c *connectClientContext) GetConnection() *ws.Connection {
	return c.conn
}

func (c *connectClientContext) SetConnection(conn *ws.Connection) {
	c.conn = conn
}

func (c *connectClientContext) Dial(ctx context.Context, url string) error {
	select {
	case c.dialChan <- url:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *connectClientContext) CloseConnection() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *connectClientContext) CloseConnectionWithCode(code int, reason string) error {
	if c.conn != nil {
		return c.conn.CloseWithCode(code, reason)
	}
	return nil
}

func (c *connectClientContext) GetTemplateEngine() *template.Engine {
	return c.tmplEngine
}

var (
	pingInterval      time.Duration
	pongWait          time.Duration
	reconnect         bool
	reconnectBackoff  time.Duration
	reconnectMax      time.Duration
	reconnectAttempts int
	maxMessageSize    int64
	maxFrameSize      int
	compress          bool
	customHeaders     []string
	authToken         string
	authBasic         string
	interactive       bool
	formatStr         string
	filterStr         string
	timestamps        bool
	scriptFile        string
	logFile           string
	recordFile        string
	once              bool
	sendMsgs          []string
	expectMsgs        []string
	untilMsg          string
	inputFile         string
	outputFile        string
	jsonl             bool
	watchPattern      string
	timeout           time.Duration
	exitOn            []string
)

var connectCmd = &cobra.Command{
	Use:   "connect [alias|url]",
	Short: "Connect to a WebSocket endpoint using an alias or URL",
	Long: `Connect to a WebSocket endpoint. You can provide a full URL (starting with ws:// or wss://) 
or a short alias/bookmark defined in your configuration file.

Example:
  xwebs connect prod
  xwebs connect wss://echo.websocket.org --subprotocol v1.xwebs
  xwebs connect secure-server --cert client.crt --key client.key --ca ca.crt`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var r *repl.REPL
		var isInteractive bool
		target := args[0]
		tmplEngine := template.New(false) // Not sandboxed for CLI usage

		// Evaluate target URL as template if it contains {{
		if strings.Contains(target, "{{") {
			evalTarget, err := tmplEngine.Execute("url", target, template.NewContext())
			if err != nil {
				return fmt.Errorf("evaluating URL template: %w", err)
			}
			target = evalTarget
		}

		details, err := config.ResolveConnDetails(target)
		if err != nil {
			// If not an alias, check if it's a valid URL
			if !strings.Contains(target, "://") {
				return fmt.Errorf("invalid URL or alias: %s", target)
			}
			// If it has ://, it might be a direct URL that ResolveConnDetails failed on (e.g. invalid scheme)
			return err
		}

		// Flags override configuration
		if cmd.Flags().Changed("insecure") {
			details.Insecure = insecure
		}
		if cmd.Flags().Changed("cert") {
			details.Cert = certFile
		}
		if cmd.Flags().Changed("key") {
			details.Key = keyFile
		}
		if cmd.Flags().Changed("ca") {
			details.CA = caFile
		}
		if cmd.Flags().Changed("proxy") {
			details.Proxy = proxy
		}
		if cmd.Flags().Changed("ping-interval") {
			details.PingInterval = pingInterval
		}
		if cmd.Flags().Changed("pong-wait") {
			details.PongWait = pongWait
		}
		if cmd.Flags().Changed("reconnect") {
			details.Reconnect = reconnect
		}
		if cmd.Flags().Changed("reconnect-backoff") {
			details.ReconnectBackoff = reconnectBackoff
		}
		if cmd.Flags().Changed("reconnect-max") {
			details.ReconnectMax = reconnectMax
		}
		if cmd.Flags().Changed("reconnect-attempts") {
			details.ReconnectAttempts = reconnectAttempts
		}
		if cmd.Flags().Changed("max-message-size") {
			details.MaxMessageSize = maxMessageSize
		}
		if cmd.Flags().Changed("max-frame-size") {
			details.MaxFrameSize = maxFrameSize
		}
		if cmd.Flags().Changed("compress") {
			details.Compress = compress
		}

		// Automation flags detection
		isAutomation := once || len(sendMsgs) > 0 || len(expectMsgs) > 0 || untilMsg != "" || inputFile != "" || jsonl || watchPattern != "" || timeout > 0 || scriptFile != ""

		if jsonl {
			formatStr = "jsonl"
			quiet = true
		}
		if watchPattern != "" {
			filterStr = watchPattern
			quiet = true
		}
		
		stat, _ := os.Stdin.Stat()
		isTerminal := (stat.Mode() & os.ModeCharDevice) != 0

		statOut, _ := os.Stdout.Stat()
		isStdoutTerminal := (statOut.Mode() & os.ModeCharDevice) != 0

		// Only start the interactive REPL loop if we are actually in a terminal and not Automating,
		// or if explicitly requested via --interactive.
		actuallyInteractive := isTerminal && !isAutomation
		if cmd.Flags().Changed("interactive") {
			actuallyInteractive = interactive
		}

		cc := &connectClientContext{
			dialChan:          make(chan string, 1),
			tmplEngine:        tmplEngine,
			originalURL:       target,
			originalHeaders:   make(map[string]string),
			receivedOnce:      make(chan struct{}),
			automationPending: isAutomation,
		}

		// Store original header templates
		for _, h := range customHeaders {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if strings.Contains(val, "{{") {
					cc.originalHeaders[key] = val
				}
			}
		}
		if strings.Contains(authToken, "{{") {
			cc.originalToken = authToken
		}
		if strings.Contains(authBasic, "{{") {
			cc.originalAuth = authBasic
		}

		header := make(http.Header)
		if len(details.Headers) > 0 {
			for k, v := range details.Headers {
				header.Add(k, v)
			}
		}

		// Add custom headers from flags, evaluating each value as a template
		for _, h := range customHeaders {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid header format %q; expected Key: Value", h)
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			if strings.Contains(val, "{{") {
				evalVal, err := tmplEngine.Execute(key, val, template.NewContext())
				if err != nil {
					return fmt.Errorf("evaluating header %q template: %w", key, err)
				}
				val = evalVal
			}
			header.Add(key, val)
		}

		// Add auth token if provided
		if authToken != "" {
			val := authToken
			if strings.Contains(val, "{{") {
				evalVal, err := tmplEngine.Execute("token", val, template.NewContext())
				if err != nil {
					return fmt.Errorf("evaluating token template: %w", err)
				}
				val = evalVal
			}
			header.Set("Authorization", "Bearer "+val)
		}

		// Add basic auth if provided
		if authBasic != "" {
			val := authBasic
			if strings.Contains(val, "{{") {
				evalVal, err := tmplEngine.Execute("auth", val, template.NewContext())
				if err != nil {
					return fmt.Errorf("evaluating auth template: %w", err)
				}
				val = evalVal
			}
			parts := strings.SplitN(val, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid auth format; expected user:pass")
			}
			auth := base64.StdEncoding.EncodeToString([]byte(val))
			header.Set("Authorization", "Basic "+auth)
		}

		opts := []ws.DialOption{
			ws.WithHeaders(header),
			ws.WithSubprotocols(subprotocols...),
			ws.WithInsecureSkipVerify(details.Insecure),
			ws.WithPingInterval(details.PingInterval),
			ws.WithPongWait(details.PongWait),
			ws.WithVerbose(verbose),
			ws.WithReconnect(details.Reconnect),
			ws.WithReconnectBackoff(details.ReconnectBackoff),
			ws.WithReconnectMax(details.ReconnectMax),
			ws.WithReconnectAttempts(details.ReconnectAttempts),
			ws.WithMaxMessageSize(details.MaxMessageSize),
			ws.WithMaxFrameSize(details.MaxFrameSize),
			ws.WithCompression(details.Compress),
			ws.WithOnDisconnect(func(code int, reason string) {
				if verbose {
					// Use warn to ensure it goes through the REPL if interactive
					msg := fmt.Sprintf("  [ws] disconnected: code=%d, reason=%q\n", code, reason)
					if isInteractive {
						warn(r, isInteractive, "%s", msg)
					} else {
						warn(r, isInteractive, "\n%s", msg)
					}
				}
			}),
		}

		if details.Proxy != "" {
			opts = append(opts, ws.WithProxy(details.Proxy))
		}
		if details.CA != "" {
			opts = append(opts, ws.WithCACert(details.CA))
		}
		if details.Cert != "" || details.Key != "" {
			opts = append(opts, ws.WithClientCert(details.Cert, details.Key))
		}
		isInteractive = isTerminal
		if cmd.Flags().Changed("interactive") {
			isInteractive = interactive
		} else if isAutomation && !isTerminal {
			// If automation is used and it's not a TTY, we use REPL context but not interactive loop
			isInteractive = true
		} else if !isTerminal {
			isInteractive = false
		}

		// If script is provided, we must use the REPL context even if not a TTY
		if scriptFile != "" {
			isInteractive = true
		}

		if isInteractive {
			var err error
			cfg := &repl.Config{}
			var appCfg config.AppConfig
			if err := viper.Unmarshal(&appCfg); err == nil {
				cfg.HistoryFile = appCfg.REPL.HistoryFile
				cfg.HistoryLimit = appCfg.REPL.HistoryLimit
			}

			if outputFile != "" {
				f, err := os.Create(outputFile)
				if err != nil {
					return fmt.Errorf("creating output file %s: %w", outputFile, err)
				}
				cfg.Stdout = f
			}
			cfg.Terminal = actuallyInteractive

			r, err = repl.New(repl.ClientMode, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to initialize REPL: %v\n", err)
				isInteractive = false
			} else {
				cc.repl = r
				r.IsInteractive = actuallyInteractive
				r.TemplateEngine = tmplEngine
				r.RegisterCommonCommands()
				r.RegisterClientCommands(cc)

				// Populate bookmarks and aliases for completion
				var cfg config.AppConfig
				if err := viper.Unmarshal(&cfg); err == nil {
					var targets []string
					for alias := range cfg.Aliases {
						targets = append(targets, alias)
					}
					for bookmark := range cfg.Bookmarks {
						targets = append(targets, bookmark)
					}
					r.SetCompletionData("bookmarks", targets)
				}

				// Populate template functions for completion
				r.SetCompletionData("template_funcs", tmplEngine.FuncNames())

				// Initialize Display state from flags/config
				r.Display.Format = repl.DisplayFormat(formatStr)
				r.Display.Quiet = quiet
				r.Display.Verbose = verbose
				r.Display.Timestamps = timestamps
				r.Display.Color = color
				
				// Enable clean output (no indicators) if the interactive REPL loop is not active
				if !actuallyInteractive {
					r.Display.NoIndicators = true
					// Default to quiet if redirected, unless explicitly set
					if !cmd.Flags().Changed("quiet") && !cmd.Flags().Changed("verbose") {
						r.Display.Quiet = true
						quiet = true
					}
				}

				if filterStr != "" {
					if err := r.Display.SetFilter(filterStr); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: invalid initial filter: %v\n", err)
					}
				}

				
				// Handle --log and --record flags
				if logFile != "" {
					if err := r.Logger.Start(logFile); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to start logging to %s: %v\n", logFile, err)
					} else {
						r.Notify("✓ Logging to %s\n", logFile)
					}
				}
				if recordFile != "" {
					if err := r.Recorder.Start(recordFile, target); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to start recording to %s: %v\n", recordFile, err)
					} else {
						r.Notify("✓ Recording to %s\n", recordFile)
					}
				}

				defer r.Close()
			}
		}

		if isTerminal && !isInteractive {
			infoln(r, isInteractive, "\nEnter message to send (Ctrl+C to disconnect):")
		}

		// Start the input reader exactly once
		inputChan := make(chan string)
		inputErrChan := make(chan error, 1)
		
		// Determine the context to use for the main session loop
		var sessionCtx context.Context
		var sessionCancel context.CancelFunc

		if isInteractive {
			sessionCtx = context.Background()
		} else {
			sessionCtx = cmd.Context()
		}

		if timeout > 0 {
			sessionCtx, sessionCancel = context.WithTimeout(sessionCtx, timeout)
			defer sessionCancel()
		}


		if colorsStr := cmd.Flag("color").Value.String(); colorsStr == "" {
			// Auto-detect color if not specified
			if outputFile != "" {
				color = "off"
			}
		}

		if actuallyInteractive {
			r.SetOnInput(func(ctx context.Context, text string) error {
				r.SetLastSendTime(time.Now())
				inputChan <- text
				return nil
			})
			go func() {
				// Use the persistent sessionCtx for the REPL loop
				if err := r.Run(sessionCtx); err != nil {
					inputErrChan <- err
				}
				close(inputChan)
			}()
		} else {
			go func() {
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					inputChan <- scanner.Text()
				}
				if err := scanner.Err(); err != nil && err != os.ErrClosed {
					inputErrChan <- err
				}
				close(inputChan)
			}()
		}

		reconnectCount := 0
		for {
			var conn *ws.Connection
			var err error

			// Re-evaluate URL if it's a template
			if cc.originalURL != "" && strings.Contains(cc.originalURL, "{{") {
				tmplCtx := template.NewContext()
				if cc.repl != nil {
					tmplCtx.Session = cc.repl.GetVars()
				}
				if evalURL, err := cc.tmplEngine.Execute("url", cc.originalURL, tmplCtx); err == nil {
					details.URL = evalURL
				}
			}

			// Re-evaluate headers if they are templates
			for key, tmpl := range cc.originalHeaders {
				tmplCtx := template.NewContext()
				if cc.repl != nil {
					tmplCtx.Session = cc.repl.GetVars()
				}
				if evalVal, err := cc.tmplEngine.Execute(key, tmpl, tmplCtx); err == nil {
					header.Set(key, evalVal)
				}
			}

			// Re-evaluate auth templates
			if cc.originalToken != "" {
				tmplCtx := template.NewContext()
				if cc.repl != nil {
					tmplCtx.Session = cc.repl.GetVars()
				}
				if evalVal, err := cc.tmplEngine.Execute("token", cc.originalToken, tmplCtx); err == nil {
					header.Set("Authorization", "Bearer "+evalVal)
				}
			}
			if cc.originalAuth != "" {
				tmplCtx := template.NewContext()
				if cc.repl != nil {
					tmplCtx.Session = cc.repl.GetVars()
				}
				if evalVal, err := cc.tmplEngine.Execute("auth", cc.originalAuth, tmplCtx); err == nil {
					parts := strings.SplitN(evalVal, ":", 2)
					if len(parts) == 2 {
						auth := base64.StdEncoding.EncodeToString([]byte(evalVal))
						header.Set("Authorization", "Basic "+auth)
					}
				}
			}

			if details.URL != "" {
				info(r, isInteractive, "Connecting to: %s\n", details.URL)
				if details.Proxy != "" && reconnectCount == 0 {
					info(r, isInteractive, "Proxy: %s\n", details.Proxy)
				}

				conn, err = ws.Dial(sessionCtx, details.URL, opts...)
				if err != nil {
					warn(r, isInteractive, "Connection failed: %v\n", err)
					if !details.Reconnect || (details.ReconnectAttempts > 0 && reconnectCount >= details.ReconnectAttempts) {
						if !isInteractive {
							return fmt.Errorf("connection failed: %w", err)
						}
						// In interactive mode, we don't exit, we wait for manual retry
						infoln(r, isInteractive, "Use :connect <url> or :reconnect to try again.")
						conn = nil // ensure it's nil
					} else {
						backoff := ws.ExponentialBackoff(details.ReconnectBackoff, details.ReconnectMax, reconnectCount)
						info(r, isInteractive, "Retrying in %v... (attempt %d)\n", backoff, reconnectCount+1)
						select {
						case <-time.After(backoff):
							reconnectCount++
							continue
						case newURL := <-cc.dialChan:
							if newURL != "" {
								details.URL = newURL
							}
							reconnectCount = 0
							continue
						case <-sessionCtx.Done():
							return nil
						}
					}
				}
			}

			if conn != nil {
				cc.SetConnection(conn)
				reconnectCount = 0 // Reset on successful connection

				// Log connection event
				if isInteractive && r != nil && r.Logger.IsActive() {
					_ = r.Logger.LogEvent("connected", map[string]interface{}{
						"url":         details.URL,
						"subprotocol": conn.NegotiatedSubprotocol,
					})
				}

				info(r, isInteractive, "Successfully connected to %s\n", details.URL)
				if conn.NegotiatedSubprotocol != "" {
					info(r, isInteractive, "Subprotocol: %s\n", conn.NegotiatedSubprotocol)
				}
				if conn.IsCompressionEnabled() {
					infoln(r, isInteractive, "Compression: permessage-deflate")
				}

				// Message reader goroutine to display incoming messages
				readDone := make(chan struct{})
				go func() {
					defer close(readDone)
					
					var fs *repl.FormattingState
					if !isInteractive {
						fs = repl.NewFormattingState()
						fs.Format = repl.DisplayFormat(formatStr)
						fs.Quiet = quiet
						fs.Verbose = verbose
						fs.Timestamps = timestamps
						fs.Color = color
						fs.NoIndicators = !actuallyInteractive
						if filterStr != "" {
							_ = fs.SetFilter(filterStr)
						}
					}

					ch := conn.Subscribe()
					defer conn.Unsubscribe(ch)
					for {
						msg, ok := <-ch
						if !ok {
							break
						}
						
						// Use REPL's centralized printing logic
						if isInteractive && r != nil {
							r.PrintMessage(msg, conn)
							
							// If interactive, try to extract JSON keys for completion
							if msg.Type == ws.TextMessage {
								var data interface{}
								if err := json.Unmarshal(msg.Data, &data); err == nil {
									keys := ExtractJSONKeys(data, "")
									for _, k := range keys {
										r.AddCompletionItem("json", k)
									}
								}
							}
						} else if fs != nil && r != nil {
							// Non-interactive mode (pipeline) with formatting
							if formatStr == "jsonl" {
								// Special case for machine-readable output
								output, _ := json.Marshal(msg)
								r.Printf("%s\n", string(output))
							} else {
								formatted, ok := fs.FormatMessage(msg, nil, cc.tmplEngine)
								if ok {
									r.Printf("%s\n", formatted)
								}
							}
						} else {
							// Fallback if neither interactive nor fs initialized
							if !quiet || (msg.Type != ws.PingMessage && msg.Type != ws.PongMessage) {
								if isStdoutTerminal {
									fmt.Fprintf(os.Stderr, "< ")
									fmt.Printf("%s\n", string(msg.Data))
								} else {
									// Raw output for pipelines
									fmt.Printf("%s\n", string(msg.Data))
								}
							}
						}

						// Handle --once flag: exit after first received message
						// We skip greetings if we are in the middle of an automation pulse (automationPending)
						if once && !cc.automationPending && msg.Metadata.Direction == "received" {
							select {
							case <-cc.receivedOnce:
								// Already triggered
							default:
								close(cc.receivedOnce)
							}
							_ = conn.Close()
							return
						}

						// Handle --until flag: exit if message matches pattern
						if untilMsg != "" && msg.Metadata.Direction == "received" {
							// Reuse FormattingState filtering logic for until if available,
							// or do a simple check. To be robust, we'll check if it matches.
							// For simplicity, we can just check if fs.FormatMessage would have shown it 
							// if it had the filter set. But fs might not be used if isInteractive.
							// Let's just use a temporary FormattingState for matching.
							matchFS := repl.NewFormattingState()
							if err := matchFS.SetFilter(untilMsg); err == nil {
								if _, matched := matchFS.FormatMessage(msg, nil, nil); matched {
									_ = conn.Close()
									return
								}
							}
						}
					}
				}()

				// Sender goroutine taking input from the singleton input loop
				sendDone := make(chan struct{})
				go func() {
					defer close(sendDone)
					for {
						select {
						case text, ok := <-inputChan:
							if !ok {
								return
							}
							msg := &ws.Message{
								Type: ws.TextMessage, 
								Data: []byte(text),
								Metadata: ws.MessageMetadata{
									Direction: "sent",
									Timestamp: time.Now(),
								},
							}
							if err := conn.Write(msg); err != nil {
								if isInteractive {
									r.Errorf("\nError sending message: %v\n", err)
								} else {
									fmt.Fprintf(os.Stderr, "\nError sending message: %v\n", err)
								}
								return
							}
							if isInteractive && r != nil {
								r.PrintMessage(msg, conn)
							}
						case <-conn.Done():
							return
						}
					}
				}()

				// Execute automation pipeline if provided
				automationCtx := sessionCtx
				cc.automationPending = true
				
				// 1. Send messages from --send
				for _, m := range sendMsgs {
					if err := r.ExecuteCommand(automationCtx, ":send "+m); err != nil {
						warn(r, isInteractive, "Send failed: %v\n", err)
						if !actuallyInteractive { 
							cc.automationPending = false
							return err 
						}
					}
				}

				// 2. Send content from --input file
				if inputFile != "" {
					content, err := os.ReadFile(inputFile)
					if err != nil {
						warn(r, isInteractive, "Failed to read input file %s: %v\n", inputFile, err)
						if !actuallyInteractive { 
							cc.automationPending = false
							return err 
						}
					} else {
						if err := r.ExecuteCommand(automationCtx, ":send "+string(content)); err != nil {
							warn(r, isInteractive, "Send from file failed: %v\n", err)
							if !actuallyInteractive { 
								cc.automationPending = false
								return err 
							}
						}
					}
				}

				// 3. Wait for expectations from --expect
				for _, e := range expectMsgs {
					if err := r.ExecuteCommand(automationCtx, ":expect "+e); err != nil {
						warn(r, isInteractive, "Expectation failed: %v\n", err)
						if !actuallyInteractive { 
							cc.automationPending = false
							return err 
						}
					}
				}

				// Automation sequence finished
				cc.automationPending = false

				// 4. Execute script if provided
				if scriptFile != "" {
					err := r.ExecuteCommand(automationCtx, ":source "+scriptFile)
					if err != nil {
						if err == repl.ErrExit {
							_ = conn.Close()
							return nil
						}
						warn(r, isInteractive, "Script failed: %v\n", err)
						if !actuallyInteractive {
							_ = conn.Close()
							return fmt.Errorf("script failed: %w", err)
						}
					} else if !actuallyInteractive {
						_ = conn.Close()
						return nil
					}
				}

				// If in automation mode and not interactive, we might want to exit now 
				// if no more pending actions and not --watch.
				if isAutomation && !actuallyInteractive && watchPattern == "" {
					// 5. If --once is active, wait for the response to be received and printed
					if once {
						select {
						case <-cc.receivedOnce:
							// Proceed to close
						case <-time.After(2 * time.Second):
							// Fallback timeout to avoid hanging if no response arrives
						case <-sessionCtx.Done():
						}
					} else if untilMsg == "" {
						// Give a small grace period for any late responses if --once was not used
						time.Sleep(500 * time.Millisecond)
					}
					
					_ = conn.Close()
					<-readDone
				}

				// Wait for connection to close, context to be cancelled, or manual dial
				var closedUnexpectedly bool
				var forcedDial bool
				select {
				case <-sendDone:
					// REPL exited via :exit
					if !isTerminal {
						time.Sleep(1 * time.Second)
					}
					_ = conn.Close()
					<-readDone
					return nil
				case <-r.Done():
					// Script or another command called :exit
					_ = conn.Close()
					<-readDone
					return nil
				case newURL := <-cc.dialChan:
					if newURL != "" {
						details.URL = newURL
					}
					forcedDial = true
					_ = conn.Close()
				case <-conn.Done():
					<-readDone
					code, reason := conn.CloseStatus()
					
					// Log disconnection event
					if isInteractive && r != nil && r.Logger.IsActive() {
						_ = r.Logger.LogEvent("disconnected", map[string]interface{}{
							"url":    details.URL,
							"code":   code,
							"reason": reason,
						})
					}

					if err := conn.Err(); err != nil {
						if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
							info(r, isInteractive, "\nConnection lost: %v (code=%d, reason=%q)\n", err, code, reason)
							closedUnexpectedly = true
						} else {
							info(r, isInteractive, "\nConnection closed: %d %s\n", code, reason)
							closedUnexpectedly = false
						}
					} else {
						info(r, isInteractive, "\nConnection closed gracefully: %d %s\n", code, reason)
						closedUnexpectedly = false
					}
				case <-sessionCtx.Done():
					infoln(r, isInteractive, "\nDisconnecting...")
					_ = conn.Close()
					return nil
				}

				cc.SetConnection(nil)
				if forcedDial {
					reconnectCount = 0
					continue
				}

				if details.Reconnect && closedUnexpectedly {
					backoff := ws.ExponentialBackoff(details.ReconnectBackoff, details.ReconnectMax, 0)
					info(r, isInteractive, "Attempting to reconnect in %v...\n", backoff)
					select {
					case <-time.After(backoff):
						continue
					case newURL := <-cc.dialChan:
						if newURL != "" {
							details.URL = newURL
						}
						reconnectCount = 0
						continue
					case <-sessionCtx.Done():
						return nil
					}
				}
			}

			// Idle state or manually disconnected: wait for new dial or exit
			if actuallyInteractive {
				infoln(r, isInteractive, "\nEnter :connect <url> or :reconnect to start a new session (or :exit to quit)")
				select {
				case newURL := <-cc.dialChan:
					if newURL != "" {
						details.URL = newURL
					}
					reconnectCount = 0
					continue
				case <-sessionCtx.Done():
					return nil
				case _, ok := <-inputChan:
					if !ok {
						return nil
					}
					// This case handles bare input when disconnected - we just ignore it
					// but we need to read it from inputChan so it doesn't block the REPL
					infoln(r, isInteractive, "No active connection. Use :connect <url> or :reconnect.")
					continue
				}
			} else {
				// Non-interactive exits if connection is closed and no reconnect happened
				break
			}
		}
		// Final grace period to ensure all output is flushed before process exit
		if !actuallyInteractive {
			time.Sleep(200 * time.Millisecond)
		}
		return nil
	},
}

func init() {
	connectCmd.Flags().StringSliceVarP(&subprotocols, "subprotocol", "s", []string{}, "suggested subprotocols for negotiation")
	connectCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "skip TLS certificate verification")
	connectCmd.Flags().StringVar(&certFile, "cert", "", "path to client certificate file (mTLS)")
	connectCmd.Flags().StringVar(&keyFile, "key", "", "path to client key file (mTLS)")
	connectCmd.Flags().StringVar(&caFile, "ca", "", "path to custom CA certificate file")
	connectCmd.Flags().DurationVar(&pingInterval, "ping-interval", 30*time.Second, "interval for automatic ping messages (0 to disable)")
	connectCmd.Flags().DurationVar(&pongWait, "pong-wait", 60*time.Second, "wait time for a pong response")
	connectCmd.Flags().BoolVar(&reconnect, "reconnect", false, "enable automatic reconnection")
	connectCmd.Flags().DurationVar(&reconnectBackoff, "reconnect-backoff", 1*time.Second, "initial backoff duration for reconnection")
	connectCmd.Flags().DurationVar(&reconnectMax, "reconnect-max", 30*time.Second, "maximum backoff duration for reconnection")
	connectCmd.Flags().IntVar(&reconnectAttempts, "reconnect-attempts", 0, "maximum number of reconnection attempts (0 for unlimited)")
	connectCmd.Flags().Int64Var(&maxMessageSize, "max-message-size", 0, "maximum message size in bytes (0 for unlimited)")
	connectCmd.Flags().IntVarP(&maxFrameSize, "max-frame-size", "f", 0, "maximum frame size for outgoing messages (0 for no fragmentation)")
	connectCmd.Flags().BoolVar(&compress, "compress", false, "enable per-message-deflate compression")
	connectCmd.Flags().StringSliceVarP(&customHeaders, "header", "H", []string{}, "custom headers to include in the handshake (Key: Value)")
	connectCmd.Flags().StringVar(&authToken, "token", "", "bearer token for authentication")
	connectCmd.Flags().StringVar(&authBasic, "auth", "", "basic auth credentials (user:pass)")
	connectCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "enable interactive REPL mode (default: auto-detect TTY)")
	connectCmd.Flags().StringVar(&formatStr, "format", "raw", "initial message display format: json, raw, hex, template")
	connectCmd.Flags().StringVar(&filterStr, "filter", "", "initial display filter (JQ or /regex/)")
	connectCmd.Flags().BoolVar(&timestamps, "timestamps", false, "display message timestamps")
	connectCmd.Flags().StringVar(&scriptFile, "script", "", "execute a .xwebs script file after connecting")
	connectCmd.Flags().StringVar(&logFile, "log", "", "log traffic to file (JSONL)")
	connectCmd.Flags().StringVar(&recordFile, "record", "", "record session to file for replay")
	connectCmd.Flags().BoolVar(&once, "once", false, "exit after the first message is received")
	connectCmd.Flags().StringArrayVar(&sendMsgs, "send", []string{}, "send a message upon connection (can be used multiple times)")
	connectCmd.Flags().StringArrayVar(&expectMsgs, "expect", []string{}, "wait for a message matching a pattern (can be used multiple times)")
	connectCmd.Flags().StringVar(&untilMsg, "until", "", "exit when a message matches this pattern")
	connectCmd.Flags().StringVar(&inputFile, "input", "", "send content from a file upon connection")
	connectCmd.Flags().StringVar(&outputFile, "output", "", "redirect formatted output to a file")
	connectCmd.Flags().BoolVar(&jsonl, "jsonl", false, "shortcut for --format jsonl")
	connectCmd.Flags().StringVar(&watchPattern, "watch", "", "monitor and print updates matching a pattern")
	connectCmd.Flags().DurationVar(&timeout, "timeout", 0, "global timeout for the connection/pipeline")
	connectCmd.Flags().StringArrayVar(&exitOn, "exit-on", []string{}, "exit conditions: match, disconnect, timeout, error")
	rootCmd.AddCommand(connectCmd)
}

var (
	subprotocols []string
	insecure     bool
	certFile     string
	keyFile      string
	caFile       string
)

func info(r *repl.REPL, isInteractive bool, format string, args ...interface{}) {
	if isInteractive && r != nil {
		r.Notify(format, args...)
	} else if !quiet {
		// Non-interactive status always goes to stderr to keep stdout pure for data
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

func infoln(r *repl.REPL, isInteractive bool, text string) {
	if isInteractive && r != nil {
		r.Notify("%s\n", text)
	} else if !quiet {
		fmt.Fprintln(os.Stderr, text)
	}
}

func warn(r *repl.REPL, isInteractive bool, format string, args ...interface{}) {
	if isInteractive && r != nil {
		r.Errorf(format, args...)
	} else {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

// ExtractJSONKeys recursively extracts all map keys from a JSON-decoded interface.
func ExtractJSONKeys(v interface{}, prefix string) []string {
	var keys []string
	switch val := v.(type) {
	case map[string]interface{}:
		for k, v2 := range val {
			fullKey := k
			if prefix != "" {
				fullKey = prefix + "." + k
			}
			keys = append(keys, fullKey)
			keys = append(keys, ExtractJSONKeys(v2, fullKey)...)
		}
	case []interface{}:
		for _, v2 := range val {
			keys = append(keys, ExtractJSONKeys(v2, prefix)...)
		}
	}
	return keys
}
