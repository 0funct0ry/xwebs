package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/config"
	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/observability"
	"github.com/0funct0ry/xwebs/internal/repl"
	"github.com/0funct0ry/xwebs/internal/server"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	servePort  int
	servePaths []string
	serveTLS   bool
	serveCert  string
	serveKey   string
	serveAPI   bool
	serveMetrics bool
	allowedOrigins []string
	allowIPs       []string
	denyIPs        []string
	rateLimit      string
	serveUI         bool
	serveInteractive bool
	serveNoInteract  bool
)

type serveContext struct {
	srv *server.Server
}

func (c *serveContext) GetClientCount() int                            { return c.srv.GetClientCount() }
func (c *serveContext) GetUptime() time.Duration                       { return c.srv.GetUptime() }
func (c *serveContext) GetClients() []template.ClientInfo              { return c.srv.GetClients() }
func (c *serveContext) GetClient(id string) (template.ClientInfo, bool) { return c.srv.GetClient(id) }
func (c *serveContext) Broadcast(msg *ws.Message) error                { return c.srv.Broadcast(msg) }
func (c *serveContext) Send(id string, msg *ws.Message) error          { return c.srv.Send(id, msg) }
func (c *serveContext) Kick(id string, code int, reason string) error  { return c.srv.Kick(id, code, reason) }
func (c *serveContext) GetStatus() string                              { return c.srv.GetStatus() }
func (c *serveContext) GetTemplateEngine() *template.Engine            { return c.srv.GetTemplateEngine() }
func (c *serveContext) GetHandlers() []handler.Handler                 { return c.srv.GetHandlers() }
func (c *serveContext) GetVariables() map[string]interface{}           { return c.srv.GetVariables() }
func (c *serveContext) GetHandlersFile() string                        { return handlersFile }
func (c *serveContext) AddHandler(h handler.Handler) error            { return c.srv.AddHandler(h) }
func (c *serveContext) UpdateHandler(h handler.Handler) error         { return c.srv.UpdateHandler(h) }
func (c *serveContext) DeleteHandler(name string) error               { return c.srv.DeleteHandler(name) }
func (c *serveContext) RenameHandler(oldName, newName string) error   { return c.srv.RenameHandler(oldName, newName) }
func (c *serveContext) ApplyHandlers(handlers []handler.Handler, variables map[string]interface{}) error {
	c.srv.ReloadHandlers(handlers, variables)
	return nil
}

func (c *serveContext) EnableHandler(name string) error {
	return c.srv.EnableHandler(name)
}

func (c *serveContext) DisableHandler(name string) error {
	return c.srv.DisableHandler(name)
}

func (c *serveContext) GetHandlerStats(name string) (uint64, time.Duration, uint64, bool) {
	return c.srv.GetHandlerStats(name)
}

func (c *serveContext) IsHandlerDisabled(name string) bool {
	return c.srv.IsHandlerDisabled(name)
}

func (c *serveContext) GetTopics() []template.TopicInfo {
	return c.srv.GetTopics()
}

func (c *serveContext) GetTopic(name string) (template.TopicInfo, bool) {
	return c.srv.GetTopic(name)
}

func (c *serveContext) PublishToTopic(topic string, msg *ws.Message) (int, error) {
	return c.srv.PublishToTopic(topic, msg)
}

func (c *serveContext) SubscribeClientToTopic(clientID, topic string) error {
	return c.srv.SubscribeClientToTopic(clientID, topic)
}

func (c *serveContext) UnsubscribeClientFromTopic(clientID, topic string) (int, error) {
	return c.srv.UnsubscribeClientFromTopic(clientID, topic)
}

func (c *serveContext) UnsubscribeClientFromAllTopics(clientID string) ([]string, error) {
	return c.srv.UnsubscribeClientFromAllTopics(clientID)
}

func (c *serveContext) ListKV() map[string]interface{} { return c.srv.ListKV() }
func (c *serveContext) GetKV(key string) (interface{}, bool) { return c.srv.GetKV(key) }
func (c *serveContext) SetKV(key string, val interface{}) { c.srv.SetKV(key, val) }
func (c *serveContext) DeleteKV(key string) { c.srv.DeleteKV(key) }

func (c *serveContext) GetGlobalStats() observability.GlobalStats { return c.srv.GetGlobalStats() }
func (c *serveContext) GetRegistryStats() (uint64, uint64) { return c.srv.GetRegistryStats() }
func (c *serveContext) GetSlowLog(limit int) []handler.SlowLogEntry { return c.srv.GetSlowLog(limit) }

func (c *serveContext) Drain()           { c.srv.Drain() }
func (c *serveContext) Pause()           { c.srv.Pause() }
func (c *serveContext) Resume()          { c.srv.Resume() }
func (c *serveContext) IsPaused() bool   { return c.srv.IsPaused() }

func (c *serveContext) ReloadHandlers() error {
	var handlers []handler.Handler
	var variables map[string]interface{}

	if handlersFile != "" {
		cfg, err := handler.LoadConfig(handlersFile)
		if err != nil {
			return fmt.Errorf("loading handlers: %w", err)
		}
		handlers = cfg.Handlers
		variables = cfg.Variables
	}

	// Re-apply inline handlers from CLI flags (preserved from startup)
	for i, hStr := range onHandlers {
		h, err := handler.ParseInlineHandler(hStr, respondTemplate, i+1)
		if err != nil {
			return fmt.Errorf("invalid inline handler %d: %w", i+1, err)
		}
		handlers = append(handlers, h)
	}

	for i, hJSON := range onMatchHandlers {
		var h handler.Handler
		if err := json.Unmarshal([]byte(hJSON), &h); err != nil {
			return fmt.Errorf("invalid inline JSON handler %d: %w", i+1, err)
		}
		if h.Name == "" {
			h.Name = fmt.Sprintf("inline-json-%d", i+1)
		}
		if respondTemplate != "" && h.Respond == "" {
			h.Respond = respondTemplate
		}
		handlers = append(handlers, h)
	}

	c.srv.ReloadHandlers(handlers, variables)
	return nil
}

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"s", "srv"},
	Short:   "Start a WebSocket server",
	Long: `Start a WebSocket server with handler support.
You can specify the listening port and one or more WebSocket paths.
Handlers can be loaded from a configuration file using the --handlers flag.

Example:
  xwebs serve --port 8080 --path /ws
  xwebs serve --handlers echo.yaml --port 9000 --path /api --path /chat`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tmplEngine := template.New(noShellFunc)

		var handlers []handler.Handler
		var variables map[string]interface{}

		if handlersFile != "" {
			cfg, err := handler.LoadConfig(handlersFile)
			if err != nil {
				return fmt.Errorf("loading handlers: %w", err)
			}
			handlers = cfg.Handlers
			variables = cfg.Variables

			// Load sandbox settings from handlers config if not explicitly set via flags
			if !cmd.Flags().Changed("sandbox") && cfg.Sandbox {
				sandboxEnabled = cfg.Sandbox
			}
			if !cmd.Flags().Changed("allowlist") && len(cfg.Allowlist) > 0 {
				allowlist = cfg.Allowlist
			}

			if !quiet {
				fmt.Fprintf(os.Stderr, "✓ Loaded %d handlers from %s\n", len(handlers), handlersFile)
			}
		}

		// Add inline handlers from CLI flags
		for i, hStr := range onHandlers {
			h, err := handler.ParseInlineHandler(hStr, respondTemplate, i+1)
			if err != nil {
				return fmt.Errorf("invalid inline handler %d: %w", i+1, err)
			}
			handlers = append(handlers, h)
		}

		for i, hJSON := range onMatchHandlers {
			var h handler.Handler
			if err := json.Unmarshal([]byte(hJSON), &h); err != nil {
				return fmt.Errorf("invalid inline JSON handler %d: %w", i+1, err)
			}
			if h.Name == "" {
				h.Name = fmt.Sprintf("inline-json-%d", i+1)
			}
			if respondTemplate != "" && h.Respond == "" {
				h.Respond = respondTemplate
			}
			handlers = append(handlers, h)
		}

		if len(handlers) > 0 && !quiet && (len(onHandlers) > 0 || len(onMatchHandlers) > 0) {
			fmt.Fprintf(os.Stderr, "✓ Added %d inline handlers\n", len(onHandlers)+len(onMatchHandlers))
		}

		// Normalize paths: ensure they start with /
		for i, p := range servePaths {
			if !strings.HasPrefix(p, "/") {
				servePaths[i] = "/" + p
			}
		}

		if serveTLS {
			if serveCert == "" || serveKey == "" {
				return fmt.Errorf("--cert and --key must be provided when --tls is enabled")
			}
		}

		srv := server.New(
			server.WithPort(servePort),
			server.WithPaths(servePaths),
			server.WithHandlers(handlers),
			server.WithVariables(variables),
			server.WithTemplateEngine(tmplEngine),
			server.WithVerbose(verbose),
			server.WithSandbox(sandboxEnabled),
			server.WithAllowlist(allowlist),
			server.WithTLS(serveTLS),
			server.WithCertFile(serveCert),
			server.WithKeyFile(serveKey),
			server.WithAPI(serveAPI),
			server.WithMetrics(serveMetrics),
			server.WithAllowedOrigins(allowedOrigins),
			server.WithAllowIPs(allowIPs),
			server.WithDenyIPs(denyIPs),
			server.WithRateLimit(rateLimit),
			server.WithUI(serveUI),
		)

		if !quiet {
			protocol := "ws"
			if serveTLS {
				protocol = "wss"
			}
			fmt.Fprintf(os.Stderr, "✓ xwebs server starting on :%d (%s)\n", servePort, protocol)
			if len(servePaths) == 1 {
				fmt.Fprintf(os.Stderr, "✓ Listening on path: %s\n", servePaths[0])
			} else {
				fmt.Fprintf(os.Stderr, "✓ Listening on paths: %s\n", strings.Join(servePaths, ", "))
			}
		}

		if handlersFile != "" {
			fmt.Fprintln(os.Stderr, "✓ Handlers loaded from:", handlersFile)
		}
		fmt.Fprintln(os.Stderr, "--------------------------------------------------")

		// TTY detection for interactive mode
		isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
		actuallyInteractive := isTerminal
		if cmd.Flags().Changed("no-interact") {
			actuallyInteractive = !serveNoInteract
		}
		if cmd.Flags().Changed("interactive") {
			actuallyInteractive = serveInteractive
		}

		if actuallyInteractive {
			replCfg := &repl.Config{}
			var appCfg config.AppConfig
			if err := viper.Unmarshal(&appCfg); err == nil {
				replCfg.HistoryFile = appCfg.REPL.HistoryFile
				replCfg.HistoryLimit = appCfg.REPL.HistoryLimit
				replCfg.PromptTemplate = appCfg.REPL.Prompt
				replCfg.Shortcuts = appCfg.REPL.Shortcuts
			}
			replCfg.Terminal = true

			r, err := repl.New(repl.ServerMode, replCfg)
			if err != nil {
				return fmt.Errorf("initializing REPL: %w", err)
			}
			defer r.Close()

			r.IsInteractive = true
			r.TemplateEngine = tmplEngine
			r.RegisterCommonCommands()
			r.RegisterServerCommands(&serveContext{srv: srv})

			// Set server logger to REPL to avoid messy concurrent output
			srv.UpdateOptions(server.WithLogger(r))

			// Run server in background
			go func() {
				if err := srv.Start(cmd.Context()); err != nil {
					r.Errorf("Server error: %v\n", err)
				}
			}()

			// Run REPL in foreground
			return r.Run(cmd.Context())
		}

		return srv.Start(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	ConfigFlags(serveCmd)
	OutputFlags(serveCmd)
	HandlerFlags(serveCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	serveCmd.Flags().StringArrayVar(&servePaths, "path", []string{"/"}, "WebSocket path(s) to listen on")
	serveCmd.Flags().BoolVar(&serveTLS, "tls", false, "enable TLS (wss://)")
	serveCmd.Flags().StringVar(&serveCert, "cert", "", "path to certificate file")
	serveCmd.Flags().StringVar(&serveKey, "key", "", "path to private key file")
	serveCmd.Flags().BoolVar(&serveAPI, "api", true, "enable REST API")
	serveCmd.Flags().BoolVar(&serveMetrics, "metrics", false, "enable Prometheus metrics")
	serveCmd.Flags().StringArrayVar(&allowedOrigins, "allowed-origins", nil, "allowed origins for WebSocket connections")
	serveCmd.Flags().StringArrayVar(&allowIPs, "allow-ip", nil, "allowed IP addresses or CIDR ranges")
	serveCmd.Flags().StringArrayVar(&denyIPs, "deny-ip", nil, "denied IP addresses or CIDR ranges")
	serveCmd.Flags().StringVar(&rateLimit, "rate-limit", "", "rate limit (e.g., '10/s' per-client, or '10/s,100/s' for per-client,global)")
	serveCmd.Flags().BoolVar(&serveUI, "ui", false, "enable web UI")
	serveCmd.Flags().BoolVarP(&serveInteractive, "interactive", "i", false, "enable interactive admin REPL")
	serveCmd.Flags().BoolVarP(&serveNoInteract, "no-interact", "I", false, "disable interactive admin REPL (same as --interactive=false)")
}
