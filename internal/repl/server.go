package repl

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/observability"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// ServerContext provides access to the server state and administration for server-mode commands.
type ServerContext interface {
	GetClientCount() int
	GetUptime() time.Duration
	GetClients() []template.ClientInfo
	GetClient(id string) (template.ClientInfo, bool)
	Broadcast(msg *ws.Message) error
	Send(id string, msg *ws.Message) error
	Kick(id string, code int, reason string) error
	GetStatus() string
	GetTemplateEngine() *template.Engine
	GetHandlers() []handler.Handler
	GetVariables() map[string]interface{}
	GetHandlersFile() string
	EnableHandler(name string) error
	DisableHandler(name string) error
	ReloadHandlers() error
	GetHandlerStats(name string) (uint64, time.Duration, uint64, bool)
	IsHandlerDisabled(name string) bool
	AddHandler(h handler.Handler) error
	UpdateHandler(h handler.Handler) error
	DeleteHandler(name string) error
	RenameHandler(oldName, newName string) error
	ApplyHandlers(handlers []handler.Handler, variables map[string]interface{}) error

	// Topic / pub-sub operations
	GetTopics() []template.TopicInfo
	GetTopic(name string) (template.TopicInfo, bool)
	PublishToTopic(topic string, msg *ws.Message) (int, error)
	SubscribeClientToTopic(clientID, topic string) error
	UnsubscribeClientFromTopic(clientID, topic string) (int, error)
	UnsubscribeClientFromAllTopics(clientID string) ([]string, error)
	
	// KV operations
	ListKV() map[string]interface{}
	GetKV(key string) (interface{}, bool)
	SetKV(key string, val interface{})
	DeleteKV(key string)

	// Observability
	GetGlobalStats() observability.GlobalStats
	GetRegistryStats() (total uint64, errors uint64)
	GetSlowLog(limit int) []handler.SlowLogEntry
	
	// Administrative
	Drain()
	Pause()
	Resume()
	IsPaused() bool
}

// RegisterServerCommands adds WebSocket server-specific commands to the REPL.
func (r *REPL) RegisterServerCommands(sc ServerContext) {
	r.serverCtx = sc
	r.RegisterCommand(&BuiltinCommand{
		name: "status",
		help: "Show server status and uptime",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			status := sc.GetStatus()
			statusColor := text.FgGreen
			if status == "paused" {
				statusColor = text.FgYellow
			} else if status == "draining" {
				statusColor = text.FgCyan
			}

			r.Printf("\nServer Status:\n")
			r.Printf("  Status:      %s\n", statusColor.Sprint(status))
			r.Printf("  Uptime:      %v\n", sc.GetUptime().Round(time.Second))
			r.Printf("  Clients:     %d\n", sc.GetClientCount())

			stats := sc.GetGlobalStats()
			r.Printf("  Connections: %d\n", stats.TotalConnections)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "drain",
		help: "Gracefully stop accepting connections and wait for existing ones to close",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			sc.Drain()
			r.Printf("✓ Server set to DRAINING. No new connections will be accepted.\n")
			
			// Optional immediate feedback loop in a goroutine
			count := sc.GetClientCount()
			if count > 0 {
				r.Printf("Waiting for %d connection(s) to close...\n", count)
				go func() {
					lastCount := count
					ticker := time.NewTicker(2 * time.Second)
					defer ticker.Stop()
					for {
						select {
						case <-ticker.C:
							current := sc.GetClientCount()
							if current == 0 {
								r.Notify("\n✓ All connections drained. Server is ready for maintenance/shutdown.\n")
								return
							}
							if current != lastCount {
								r.Notify("[drain] %d connections remaining...\n", current)
								lastCount = current
							}
						case <-ctx.Done():
							return
						}
					}
				}()
			} else {
				r.Printf("✓ Server is already empty.\n")
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "pause",
		help: "Temporarily pause message processing (incoming messages will be buffered)",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if sc.IsPaused() {
				r.Printf("Server is already paused.\n")
				return nil
			}
			sc.Pause()
			r.Printf("✓ Server PAUSED. Incoming messages from all clients will be buffered.\n")
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "resume",
		help: "Resume normal message processing and flush buffered messages",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if !sc.IsPaused() {
				r.Printf("Server is not paused.\n")
				return nil
			}
			sc.Resume()
			r.Printf("✓ Server RESUMED. Processing buffered messages...\n")
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "stats",
		help: "Show global server statistics",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			stats := sc.GetGlobalStats()
			hHits, hErrs := sc.GetRegistryStats()

			tw := table.NewWriter()
			tw.SetOutputMirror(nil)
			tw.AppendHeader(table.Row{"Category", "Metric", "Value"})
			tw.AppendRows([]table.Row{
				{"Connections", "Current Active", sc.GetClientCount()},
				{"Connections", "Total Lifetime", stats.TotalConnections},
				{"Messages", "Received", stats.MessagesReceived},
				{"Messages", "Sent", stats.MessagesSent},
				{"Handlers", "Total Executions", hHits},
				{"Handlers", "Errors", hErrs},
				{"Server", "Global Errors", stats.TotalErrors},
				{"Server", "Uptime", sc.GetUptime().Round(time.Second)},
			})

			tw.SetStyle(table.StyleColoredDark)
			tw.Style().Options.SeparateRows = false
			tw.Style().Options.SeparateColumns = true
			tw.Style().Options.DrawBorder = true

			r.Printf("\nServer Observability Statistics:\n%s\n", tw.Render())
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "slow",
		help: "Show slowest handler executions: :slow [limit]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			limit := 10
			if len(args) > 0 {
				if l, err := strconv.Atoi(args[0]); err == nil {
					limit = l
				}
			}

			slowLog := sc.GetSlowLog(limit)
			if len(slowLog) == 0 {
				r.Printf("No slow executions recorded yet.\n")
				return nil
			}

			tw := table.NewWriter()
			tw.SetOutputMirror(nil)
			tw.AppendHeader(table.Row{"Timestamp", "Handler", "Duration", "Error"})

			for _, entry := range slowLog {
				errStr := "-"
				if entry.Error != "" {
					errStr = text.FgRed.Sprint(entry.Error)
				}
				tw.AppendRow(table.Row{
					entry.Timestamp.Format("15:04:05"),
					entry.HandlerName,
					entry.Duration.Round(time.Microsecond),
					errStr,
				})
			}

			tw.SetStyle(table.StyleColoredDark)
			tw.Style().Options.SeparateRows = false
			tw.Style().Options.SeparateColumns = true
			tw.Style().Options.DrawBorder = true

			r.Printf("\nSlowest Handler Executions (Top %d):\n%s\n", limit, tw.Render())
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "clients",
		help: "List active client connections",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			clients := sc.GetClients()
			if len(clients) == 0 {
				r.Printf("No active clients connected.\n")
				return nil
			}

			r.Printf("\nActive Clients (%d):\n", len(clients))
			r.Printf("%-15s %-20s %-12s %-6s %-6s %s\n", "ID", "Remote Address", "Uptime", "IN", "OUT", "Connected At")
			r.Printf("%s\n", strings.Repeat("-", 80))
			for _, c := range clients {
				r.Printf("%-15s %-20s %-12s %-6d %-6d %s\n",
					c.ID,
					c.RemoteAddr,
					c.UptimeStr,
					c.MsgsIn,
					c.MsgsOut,
					c.ConnectedAt.Format("15:04:05"),
				)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "client",
		help: "Show detailed information about a specific client: :client <id>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :client <id>")
			}
			id := args[0]
			c, ok := sc.GetClient(id)
			if !ok {
				return fmt.Errorf("client %s not found", id)
			}

			r.Printf("\nClient Information: %s\n", id)
			r.Printf("  Remote Address: %s\n", c.RemoteAddr)
			r.Printf("  Connected At:   %s (%s ago)\n", c.ConnectedAt.Format("2006-01-02 15:04:05"), c.UptimeStr)
			r.Printf("  Messages IN:    %d\n", c.MsgsIn)
			r.Printf("  Messages OUT:   %d\n", c.MsgsOut)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "broadcast",
		help: "Broadcast a message to all connected clients: :broadcast [flags] <message>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var isJSON, isTemplate, isBinary bool
			fs := pflag.NewFlagSet("broadcast", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVarP(&isJSON, "json", "j", false, "Send as JSON")
			fs.BoolVarP(&isTemplate, "template", "t", false, "Send as rendered template")
			fs.BoolVarP(&isBinary, "binary", "b", false, "Send as binary (hex or base64: prefix)")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			remaining := fs.Args()
			if len(remaining) == 0 {
				return fmt.Errorf("usage: :broadcast [flags] <message>")
			}
			msgStr := strings.Join(remaining, " ")

			var data []byte
			var msgType ws.MessageType = ws.TextMessage
			var err error

			if isJSON {
				if !json.Valid([]byte(msgStr)) {
					return fmt.Errorf("invalid JSON: %s", msgStr)
				}
				data = []byte(msgStr)
			} else if isTemplate {
				engine := sc.GetTemplateEngine()
				if engine == nil {
					return fmt.Errorf("template engine not available")
				}
				tmplCtx := template.NewContext()
				r.PopulateContext(tmplCtx)
				res, err := engine.Execute("repl", msgStr, tmplCtx)
				if err != nil {
					return fmt.Errorf("rendering template: %w", err)
				}
				data = []byte(res)
			} else if isBinary {
				msgType = ws.BinaryMessage
				if strings.HasPrefix(msgStr, "base64:") {
					data, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(msgStr, "base64:"))
				} else {
					data, err = hex.DecodeString(strings.ReplaceAll(msgStr, " ", ""))
				}
				if err != nil {
					return fmt.Errorf("decoding binary data: %w", err)
				}
			} else {
				data = []byte(msgStr)
			}

			msg := &ws.Message{
				Type: msgType,
				Data: data,
				Metadata: ws.MessageMetadata{
					Direction: "sent",
					Timestamp: time.Now(),
				},
			}
			if err := sc.Broadcast(msg); err != nil {
				return fmt.Errorf("broadcast failed: %w", err)
			}
			r.Printf("✓ Broadcasted message to %d clients\n", sc.GetClientCount())
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "kick",
		help: "Disconnect a specific client: :kick <id> [code] [reason]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :kick <id> [code] [reason]")
			}
			id := args[0]
			code := 0
			reason := ""
			if len(args) > 1 {
				c, err := strconv.Atoi(args[1])
				if err != nil {
					return fmt.Errorf("invalid close code: %s", args[1])
				}
				code = c
			}
			if len(args) > 2 {
				reason = strings.Join(args[2:], " ")
			}

			if err := sc.Kick(id, code, reason); err != nil {
				return fmt.Errorf("failed to kick client %s: %w", id, err)
			}
			r.Printf("✓ Client %s kicked\n", id)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "send",
		help: "Send a message to a specific client: :send [flags] <id> <message>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var isJSON, isTemplate, isBinary bool
			fs := pflag.NewFlagSet("send", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVarP(&isJSON, "json", "j", false, "Send as JSON")
			fs.BoolVarP(&isTemplate, "template", "t", false, "Send as rendered template")
			fs.BoolVarP(&isBinary, "binary", "b", false, "Send as binary (hex or base64: prefix)")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			remaining := fs.Args()
			if len(remaining) < 2 {
				return fmt.Errorf("usage: :send [flags] <id> <message>")
			}
			id := remaining[0]
			msgStr := strings.Join(remaining[1:], " ")

			var data []byte
			var msgType ws.MessageType = ws.TextMessage
			var err error

			if isJSON {
				if !json.Valid([]byte(msgStr)) {
					return fmt.Errorf("invalid JSON: %s", msgStr)
				}
				data = []byte(msgStr)
			} else if isTemplate {
				engine := sc.GetTemplateEngine()
				if engine == nil {
					return fmt.Errorf("template engine not available")
				}
				tmplCtx := template.NewContext()
				r.PopulateContext(tmplCtx)
				res, err := engine.Execute("repl", msgStr, tmplCtx)
				if err != nil {
					return fmt.Errorf("rendering template: %w", err)
				}
				data = []byte(res)
			} else if isBinary {
				msgType = ws.BinaryMessage
				if strings.HasPrefix(msgStr, "base64:") {
					data, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(msgStr, "base64:"))
				} else {
					data, err = hex.DecodeString(strings.ReplaceAll(msgStr, " ", ""))
				}
				if err != nil {
					return fmt.Errorf("decoding binary data: %w", err)
				}
			} else {
				data = []byte(msgStr)
			}

			msg := &ws.Message{
				Type: msgType,
				Data: data,
				Metadata: ws.MessageMetadata{
					Direction: "sent",
					Timestamp: time.Now(),
				},
			}
			if err := sc.Send(id, msg); err != nil {
				return fmt.Errorf("failed to send message to %s: %w", id, err)
			}
			r.Printf("✓ Message sent to client %s\n", id)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "handlers",
		help: "List server-side handlers and statistics",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			handlers := sc.GetHandlers()
			if len(handlers) == 0 {
				r.Printf("No server-side handlers registered.\n")
				return nil
			}

			tw := table.NewWriter()
			tw.SetOutputMirror(nil)
			tw.AppendHeader(table.Row{"Name", "Matches", "Avg Latency", "Errors", "Status", "Pattern"})

			for _, h := range handlers {
				matches, totalLatency, errors, ok := sc.GetHandlerStats(h.Name)
				avgLatencyStr := "-"
				if ok && matches > 0 {
					avgLatencyStr = (totalLatency / time.Duration(matches)).Round(time.Microsecond).String()
				}
				matchesStr := "-"
				if ok {
					matchesStr = fmt.Sprintf("%d", matches)
				}
				errorsStr := "-"
				if ok {
					errorsStr = fmt.Sprintf("%d", errors)
				}

				status := text.FgGreen.Sprint("enabled")
				if sc.IsHandlerDisabled(h.Name) {
					status = text.FgRed.Sprint("disabled")
				}

				tw.AppendRow(table.Row{
					h.Name,
					matchesStr,
					avgLatencyStr,
					errorsStr,
					status,
					h.Match.Pattern,
				})
			}

			tw.SetStyle(table.StyleColoredDark)
			tw.Style().Options.SeparateRows = false
			tw.Style().Options.SeparateColumns = true
			tw.Style().Options.DrawBorder = true

			r.Printf("\nServer Handlers (%d):\n%s\n", len(handlers), tw.Render())
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "handler",
		help: "Manage and show handlers: :handler (add <flags> | delete <name> | <name>)",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				r.Printf("Usage:\n")
				r.Printf("  :handler add <flags>  Add a new handler\n")
				r.Printf("  :handler delete <name> Remove an existing handler\n")
				r.Printf("  :handler rename <old> <new> Rename a handler\n")
				r.Printf("  :handler edit [name]  Edit a handler or full configuration\n")
				r.Printf("  :handler save [file] [--force|-f] Save handlers to YAML file\n")
				r.Printf("  :handler <name>       Show detailed information about a handler\n")
				r.Printf("\nFlags for 'add':\n")
				r.Printf("  -n, --name <name>         Unique handler name (auto-generated if missing)\n")
				r.Printf("  -m, --match <pattern>     (required) Match pattern\n")
				r.Printf("  -t, --match-type <type>   Match type (text, glob, regex, jq, etc.)\n")
				r.Printf("  -p, --priority <n>        Numeric priority (higher runs first)\n")
				r.Printf("  -r, --run <cmd>           Shell command to run on match\n")
				r.Printf("  -R, --respond <tmpl>      Response template to send back\n")
				r.Printf("  -B, --builtin <name>      Builtin action (subscribe, unsubscribe, publish)\n")
				r.Printf("      --topic <template>    Topic name template for builtin actions\n")
				r.Printf("  -e, --exclusive           Stop further matching if this handler matches\n")
				r.Printf("  -s, --sequential          Run handler actions sequentially\n")
				r.Printf("  -l, --rate-limit <limit>  Rate limit (e.g. '10/s')\n")
				r.Printf("  -d, --debounce <duration> Debounce time (e.g. '500ms')\n")
				return nil
			}

			if args[0] == "add" {
				// Parse flags using pflag
				fs := pflag.NewFlagSet("handler add", pflag.ContinueOnError)
				fs.SetOutput(nil)

				var name, match, matchType, run, respond, builtin, topic, rateLimit, debounce string
				var priority int
				var exclusive, sequential bool

				fs.StringVarP(&name, "name", "n", "", "Name of the handler")
				fs.StringVarP(&match, "match", "m", "", "Match pattern")
				fs.StringVarP(&matchType, "match-type", "t", "", "Match type")
				fs.IntVarP(&priority, "priority", "p", 0, "Priority")
				fs.StringVarP(&run, "run", "r", "", "Shell command")
				fs.StringVarP(&respond, "respond", "R", "", "Response template")
				fs.StringVarP(&builtin, "builtin", "B", "", "Builtin action (subscribe, unsubscribe, publish)")
				fs.StringVar(&topic, "topic", "", "Topic name template for builtin actions")
				fs.BoolVarP(&exclusive, "exclusive", "e", false, "Short-circuit match")
				fs.BoolVarP(&sequential, "sequential", "s", false, "Run actions sequentially")
				fs.StringVarP(&rateLimit, "rate-limit", "l", "", "Rate limit")
				fs.StringVarP(&debounce, "debounce", "d", "", "Debounce duration")

				if err := fs.Parse(args[1:]); err != nil {
					return fmt.Errorf("parsing flags: %w", err)
				}

				if match == "" {
					return fmt.Errorf("-m/--match is required")
				}

				if name == "" {
					name = namesgenerator.GetRandomName(0)
				}

				if builtin != "" && topic == "" {
					return fmt.Errorf("--topic is required when --builtin is set")
				}

				h := handler.Handler{
					Name:      name,
					Priority:  priority,
					Exclusive: exclusive,
					Run:       run,
					Respond:   respond,
					Builtin:   builtin,
					Topic:     topic,
					Match: handler.Matcher{
						Pattern: match,
						Type:    matchType,
					},
					RateLimit: rateLimit,
					Debounce:  debounce,
				}

				if sequential {
					f := false
					h.Concurrent = &f
				}

				if err := sc.AddHandler(h); err != nil {
					return err
				}

				r.Printf("Handler %q added successfully.\n", name)
				return nil
			}

			if args[0] == "delete" {
				if len(args) < 2 {
					return fmt.Errorf("usage: :handler delete <name>")
				}
				name := args[1]
				if err := sc.DeleteHandler(name); err != nil {
					return err
				}
				r.Printf("✓ Handler %q deleted successfully\n", name)
				return nil
			}

			if args[0] == "rename" {
				if len(args) < 3 {
					return fmt.Errorf("usage: :handler rename <old-name> <new-name>")
				}
				oldName := args[1]
				newName := args[2]

				if err := sc.RenameHandler(oldName, newName); err != nil {
					return err
				}
				r.Printf("✓ Handler %q renamed to %q\n", oldName, newName)
				return nil
			}

			if args[0] == "edit" {
				if len(args) > 1 {
					// Edit specific handler
					name := args[1]
					handlers := sc.GetHandlers()
					var target *handler.Handler
					for _, h := range handlers {
						if h.Name == name {
							target = &h
							break
						}
					}
					if target == nil {
						return fmt.Errorf("handler %q not found", name)
					}

					data, err := yaml.Marshal(target)
					if err != nil {
						return fmt.Errorf("marshaling handler: %w", err)
					}

					edited, err := r.openInEditor(ctx, string(data))
					if err != nil {
						return err
					}

					// Check if any changes were made
					if strings.TrimSpace(edited) == "" || strings.TrimSpace(edited) == strings.TrimSpace(string(data)) {
						r.Printf("No changes made.\n")
						return nil
					}

					var updatedh handler.Handler
					if err := yaml.Unmarshal([]byte(edited), &updatedh); err != nil {
						return fmt.Errorf("unmarshaling edited handler: %w", err)
					}

					// Validate
					cfg := handler.Config{Handlers: []handler.Handler{updatedh}}
					if err := cfg.Validate(); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}

					if err := sc.UpdateHandler(updatedh); err != nil {
						return err
					}
					r.Printf("✓ Handler %q updated successfully\n", updatedh.Name)
					return nil
				} else {
					// Edit full current configuration
					handlers := sc.GetHandlers()
					// We don't have a direct way to get top-level 'Variables' from sc yet if they were loaded from file,
					// but ReloadHandlers re-applies them. For now, we'll edit handlers.
					// If we want parity with client REPL, we should probably add GetVariables to sc.
					cfg := handler.Config{
						Handlers: handlers,
						// Variables: sc.GetVariables(), // TODO: add if needed
					}

					data, err := yaml.Marshal(cfg)
					if err != nil {
						return fmt.Errorf("marshaling configuration: %w", err)
					}

					edited, err := r.openInEditor(ctx, string(data))
					if err != nil {
						return err
					}

					// Check if any changes were made
					if strings.TrimSpace(edited) == "" || strings.TrimSpace(edited) == strings.TrimSpace(string(data)) {
						r.Printf("No changes made.\n")
						return nil
					}

					var newCfg handler.Config
					if err := yaml.Unmarshal([]byte(edited), &newCfg); err != nil {
						return fmt.Errorf("unmarshaling edited configuration: %w", err)
					}

					if err := newCfg.Validate(); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}

					// Apply changes
					if err := sc.ApplyHandlers(newCfg.Handlers, newCfg.Variables); err != nil {
						return err
					}

					r.Printf("✓ Handler configuration applied successfully\n")
					return nil
				}
			}

			if args[0] == "save" {
				var filename string
				saveArgs := args[1:]
				if len(saveArgs) > 0 && !strings.HasPrefix(saveArgs[0], "-") {
					filename = saveArgs[0]
					saveArgs = saveArgs[1:]
				}
				usedDefaultHandlersFile := false
				if filename == "" {
					filename = sc.GetHandlersFile()
					if filename == "" {
						return fmt.Errorf("usage: :handler save [filename] [--force|-f] (or start with --handlers)")
					}
					usedDefaultHandlersFile = true
				}

				var force bool
				fs := pflag.NewFlagSet("handler save", pflag.ContinueOnError)
				fs.SetOutput(nil)
				fs.BoolVarP(&force, "force", "f", false, "Overwrite existing file")
				if err := fs.Parse(saveArgs); err != nil {
					return fmt.Errorf("parsing flags: %w", err)
				}

				if _, err := os.Stat(filename); err == nil && !force {
					if !usedDefaultHandlersFile {
						return fmt.Errorf("file %q already exists (use --force or -f to overwrite)", filename)
					}
					r.Printf("File %q already exists. Overwrite? (y/N): ", filename)
					var answer string
					_, _ = fmt.Scanln(&answer)
					answer = strings.TrimSpace(strings.ToLower(answer))
					if answer != "y" && answer != "yes" {
						r.Printf("Save cancelled.\n")
						return nil
					}
				}

				cfg := handler.Config{
					Variables: sc.GetVariables(),
					Handlers:  sc.GetHandlers(),
				}
				data, err := yaml.Marshal(cfg)
				if err != nil {
					return fmt.Errorf("marshaling handlers: %w", err)
				}
				if err := os.WriteFile(filename, data, 0644); err != nil {
					return fmt.Errorf("writing to file: %w", err)
				}

				r.Printf("Saved %d handlers to %q.\n", len(cfg.Handlers), filename)
				return nil
			}

			name := args[0]
			handlers := sc.GetHandlers()
			var target *handler.Handler
			for _, h := range handlers {
				if h.Name == name {
					target = &h
					break
				}
			}
			if target == nil {
				return fmt.Errorf("handler %q not found", name)
			}

			r.Printf("\nHandler Details: %s\n", name)
			r.Printf("  Priority:    %d\n", target.Priority)
			r.Printf("  Exclusive:   %v\n", target.Exclusive)
			r.Printf("  Status:      %s\n", map[bool]string{false: "enabled", true: "disabled"}[sc.IsHandlerDisabled(name)])
			r.Printf("  Match:\n")
			if target.Match.Pattern != "" {
				r.Printf("    Pattern:   %s (%s)\n", target.Match.Pattern, target.Match.Type)
			}
			if target.Match.Regex != "" {
				r.Printf("    Regex:     %s\n", target.Match.Regex)
			}
			if target.Match.JQ != "" {
				r.Printf("    JQ:        %s\n", target.Match.JQ)
			}

			matches, totalLatency, errors, ok := sc.GetHandlerStats(name)
			if ok {
				avgLatency := time.Duration(0)
				if matches > 0 {
					avgLatency = totalLatency / time.Duration(matches)
				}
				r.Printf("  Statistics:\n")
				r.Printf("    Matches:     %d\n", matches)
				r.Printf("    Avg Latency: %v\n", avgLatency.Round(time.Microsecond))
				r.Printf("    Errors:      %d\n", errors)
			}

			if target.Run != "" {
				r.Printf("  Actions:\n")
				r.Printf("    Run:       %s\n", target.Run)
			}
			if target.Respond != "" {
				r.Printf("    Respond:   %s\n", target.Respond)
			}
			if len(target.Pipeline) > 0 {
				r.Printf("  Pipeline:\n")
				for i, step := range target.Pipeline {
					if step.Run != "" {
						r.Printf("    %d. Run:    %s\n", i+1, step.Run)
					} else {
						r.Printf("    %d. Builtin: %s\n", i+1, step.Builtin)
					}
				}
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "reload",
		help: "Hot-reload handler configuration from disk",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if err := sc.ReloadHandlers(); err != nil {
				return err
			}
			r.Printf("✓ Handlers reloaded successfully\n")
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "enable",
		help: "Enable a disabled handler: :enable <name>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :enable <name>")
			}
			name := args[0]
			if err := sc.EnableHandler(name); err != nil {
				return err
			}
			r.Printf("✓ Handler %q enabled\n", name)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "disable",
		help: "Disable a handler at runtime: :disable <name>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :disable <name>")
			}
			name := args[0]
			if err := sc.DisableHandler(name); err != nil {
				return err
			}
			r.Printf("✓ Handler %q disabled\n", name)
			return nil
		},
	})

	// ── Topic / pub-sub commands ─────────────────────────────────────────────

	r.RegisterCommand(&BuiltinCommand{
		name: "topics",
		help: "List all active pub/sub topics with subscriber counts",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			topics := sc.GetTopics()
			if len(topics) == 0 {
				r.Printf("No active topics.\n")
				return nil
			}

			totalSubs := 0
			for _, t := range topics {
				totalSubs += len(t.Subscribers)
			}

			tw := table.NewWriter()
			tw.SetOutputMirror(nil)
			tw.AppendHeader(table.Row{"TOPIC", "SUBSCRIBERS", "LAST ACTIVE"})

			for _, t := range topics {
				tw.AppendRow(table.Row{
					t.Name,
					len(t.Subscribers),
					formatLastActive(t.LastActive),
				})
			}

			tw.SetStyle(table.StyleColoredDark)
			tw.Style().Options.SeparateRows = false
			tw.Style().Options.SeparateColumns = true
			tw.Style().Options.DrawBorder = true

			r.Printf("\n%s\n\n%d %s, %d total %s\n",
				tw.Render(),
				len(topics),
				pluralise(len(topics), "topic", "topics"),
				totalSubs,
				pluralise(totalSubs, "subscription", "subscriptions"),
			)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "topic",
		help: "Show detailed state for a single topic: :topic <name>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :topic <name>")
			}
			name := args[0]
			info, ok := sc.GetTopic(name)
			if !ok {
				return fmt.Errorf("topic %q not found. Run :topics to see active topics", name)
			}

			r.Printf("\nTopic: %s\nSubscribers: %d\n", info.Name, len(info.Subscribers))

			if len(info.Subscribers) == 0 {
				r.Printf("  (no subscribers)\n")
				return nil
			}

			tw := table.NewWriter()
			tw.SetOutputMirror(nil)
			tw.AppendHeader(table.Row{"ID", "REMOTE ADDR", "SUBSCRIBED", "MESSAGES SENT"})

			for _, sub := range info.Subscribers {
				tw.AppendRow(table.Row{
					sub.ConnID,
					sub.RemoteAddr,
					formatLastActive(sub.SubscribedAt),
					sub.MsgsSent,
				})
			}

			tw.SetStyle(table.StyleColoredDark)
			tw.Style().Options.SeparateRows = false
			tw.Style().Options.SeparateColumns = true
			tw.Style().Options.DrawBorder = true

			r.Printf("\n%s\n", tw.Render())
			return nil
		},
	})

	// ── Key-Value Store commands ──────────────────────────────────────────────

	r.RegisterCommand(&BuiltinCommand{
		name: "kv",
		help: "Manage the server-side key-value store: :kv (list|get|set|del)",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				r.Printf("Usage:\n")
				r.Printf("  :kv list             List all keys in the store\n")
				r.Printf("  :kv get <key>        Get the value for a key\n")
				r.Printf("  :kv set <key> <val>  Set a key-value pair\n")
				r.Printf("  :kv del <key>        Delete a key\n")
				r.Printf("\nFlags for 'set':\n")
				r.Printf("  -t, --template       Render value as a template before storing\n")
				r.Printf("  -j, --json           Parse value as JSON before storing\n")
				return nil
			}

			sub := args[0]
			switch sub {
			case "list", "ls":
				data := sc.ListKV()
				if len(data) == 0 {
					r.Printf("KV store is empty.\n")
					return nil
				}

				tw := table.NewWriter()
				tw.SetOutputMirror(nil)
				tw.AppendHeader(table.Row{"KEY", "VALUE TYPE", "VALUE"})

				// Sort keys for deterministic output
				keys := make([]string, 0, len(data))
				for k := range data {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				for _, k := range keys {
					v := data[k]
					valStr := fmt.Sprintf("%v", v)
					if len(valStr) > 50 {
						valStr = valStr[:47] + "..."
					}
					tw.AppendRow(table.Row{k, fmt.Sprintf("%T", v), valStr})
				}

				tw.SetStyle(table.StyleColoredDark)
				tw.Style().Options.SeparateRows = false
				tw.Style().Options.SeparateColumns = true
				tw.Style().Options.DrawBorder = true

				r.Printf("\nKV Store Entries (%d):\n%s\n", len(data), tw.Render())
				return nil

			case "get":
				if len(args) < 2 {
					return fmt.Errorf("usage: :kv get <key>")
				}
				key := args[1]
				val, ok := sc.GetKV(key)
				if !ok {
					return fmt.Errorf("key %q not found", key)
				}
				r.Printf("%s = %v\n", key, val)
				return nil

			case "set":
				var asTemplate, asJSON bool
				fs := pflag.NewFlagSet("kv set", pflag.ContinueOnError)
				fs.SetOutput(nil)
				fs.BoolVarP(&asTemplate, "template", "t", false, "Render as template")
				fs.BoolVarP(&asJSON, "json", "j", false, "Parse as JSON")

				if err := fs.Parse(args[1:]); err != nil {
					return fmt.Errorf("parsing flags: %w", err)
				}

				remaining := fs.Args()
				if len(remaining) < 2 {
					return fmt.Errorf("usage: :kv set [-t/-j] <key> <value>")
				}

				key := remaining[0]
				valStr := strings.Join(remaining[1:], " ")
				var val interface{}

				if asTemplate {
					engine := sc.GetTemplateEngine()
					if engine == nil {
						return fmt.Errorf("template engine not available")
					}
					tmplCtx := template.NewContext()
					r.PopulateContext(tmplCtx)
					res, err := engine.Execute("repl-kv", valStr, tmplCtx)
					if err != nil {
						return fmt.Errorf("rendering template: %w", err)
					}
					valStr = res
				}

				if asJSON {
					if err := json.Unmarshal([]byte(valStr), &val); err != nil {
						return fmt.Errorf("invalid JSON: %w", err)
					}
				} else {
					val = valStr
				}

				sc.SetKV(key, val)
				r.Printf("✓ Key %q set\n", key)
				return nil

			case "del", "rm":
				if len(args) < 2 {
					return fmt.Errorf("usage: :kv del <key>")
				}
				key := args[1]
				sc.DeleteKV(key)
				r.Printf("✓ Key %q deleted\n", key)
				return nil

			default:
				return fmt.Errorf("unknown kv subcommand: %s", sub)
			}
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "publish",
		help: "Publish a message to a topic: :publish [-t] [--allow-empty] <topic> <message>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var isTemplate, allowEmpty bool
			fs := pflag.NewFlagSet("publish", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVarP(&isTemplate, "template", "t", false, "Expand message as a Go template before sending")
			fs.BoolVar(&allowEmpty, "allow-empty", false, "Send even when no subscribers are present")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			remaining := fs.Args()
			if len(remaining) < 2 {
				return fmt.Errorf("usage: :publish [-t] [--allow-empty] <topic> <message>")
			}

			topic := remaining[0]
			msgStr := strings.Join(remaining[1:], " ")

			if isTemplate {
				engine := sc.GetTemplateEngine()
				if engine == nil {
					return fmt.Errorf("template engine not available")
				}
				tmplCtx := template.NewContext()
				r.PopulateContext(tmplCtx)
				rendered, err := engine.Execute("repl-publish", msgStr, tmplCtx)
				if err != nil {
					return fmt.Errorf("rendering template: %w", err)
				}
				msgStr = rendered
			}

			msg := &ws.Message{
				Type: ws.TextMessage,
				Data: []byte(msgStr),
				Metadata: ws.MessageMetadata{
					Direction: "sent",
					Timestamp: time.Now(),
				},
			}

			delivered, err := sc.PublishToTopic(topic, msg)
			if err != nil {
				if allowEmpty {
					// Topic has no subscribers yet — create it by doing nothing but report success.
					r.Printf("✓ Published to %q → 0 clients (no subscribers)\n", topic)
					return nil
				}
				return fmt.Errorf("topic %q has no subscribers. Message not sent", topic)
			}

			r.Printf("✓ Published to %q → %d %s\n", topic, delivered, pluralise(delivered, "client", "clients"))
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "subscribe",
		help: "Manually subscribe a connected client to a topic: :subscribe <client-id> <topic>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: :subscribe <client-id> <topic>")
			}
			clientID := args[0]
			topic := args[1]

			if err := sc.SubscribeClientToTopic(clientID, topic); err != nil {
				return err
			}

			// Fetch remote addr for the confirmation message (best-effort).
			remoteAddr := ""
			if info, ok := sc.GetClient(clientID); ok {
				remoteAddr = info.RemoteAddr
			}
			if remoteAddr != "" {
				r.Printf("✓ %s (%s) subscribed to %q\n", clientID, remoteAddr, topic)
			} else {
				r.Printf("✓ %s subscribed to %q\n", clientID, topic)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "unsubscribe",
		help: "Remove a client from a topic (or all topics): :unsubscribe <client-id> <topic> | :unsubscribe <client-id> --all",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var all bool
			fs := pflag.NewFlagSet("unsubscribe", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVar(&all, "all", false, "Remove client from every subscribed topic")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			remaining := fs.Args()
			if len(remaining) == 0 {
				return fmt.Errorf("usage: :unsubscribe <client-id> <topic> | :unsubscribe <client-id> --all")
			}

			clientID := remaining[0]

			if all {
				topics, err := sc.UnsubscribeClientFromAllTopics(clientID)
				if err != nil {
					return err
				}
				if len(topics) == 0 {
					r.Printf("%s was not subscribed to any topics\n", clientID)
					return nil
				}
				r.Printf("✓ %s removed from %d %s: %s\n",
					clientID,
					len(topics),
					pluralise(len(topics), "topic", "topics"),
					strings.Join(topics, ", "),
				)
				return nil
			}

			if len(remaining) < 2 {
				return fmt.Errorf("usage: :unsubscribe <client-id> <topic> | :unsubscribe <client-id> --all")
			}
			topic := remaining[1]

			remaining2, err := sc.UnsubscribeClientFromTopic(clientID, topic)
			if err != nil {
				return err
			}

			if remaining2 == 0 {
				r.Printf("✓ %s removed from %q (no subscribers remain)\n", clientID, topic)
			} else {
				r.Printf("✓ %s removed from %q (%d %s remain)\n",
					clientID, topic, remaining2,
					pluralise(remaining2, "subscriber", "subscribers"),
				)
			}
			return nil
		},
	})
}

// formatLastActive returns a human-readable description of how long ago t was.
func formatLastActive(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t).Round(time.Second)
	if d < 2*time.Second {
		return "just now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	if secs == 0 {
		return fmt.Sprintf("%dm ago", mins)
	}
	return fmt.Sprintf("%dm%02ds ago", mins, secs)
}

// pluralise returns singular when n==1, plural otherwise.
func pluralise(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
