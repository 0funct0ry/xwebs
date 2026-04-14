package repl

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/pflag"
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
	EnableHandler(name string) error
	DisableHandler(name string) error
	ReloadHandlers() error
	GetHandlerStats(name string) (uint64, time.Duration, uint64, bool)
	IsHandlerDisabled(name string) bool
}

// RegisterServerCommands adds WebSocket server-specific commands to the REPL.
func (r *REPL) RegisterServerCommands(sc ServerContext) {
	r.serverCtx = sc
	r.RegisterCommand(&BuiltinCommand{
		name: "status",
		help: "Show server status and uptime",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			r.Printf("\nServer Status:\n")
			r.Printf("  Status:      %s\n", sc.GetStatus())
			r.Printf("  Uptime:      %v\n", sc.GetUptime().Round(time.Second))
			r.Printf("  Clients:     %d\n", sc.GetClientCount())
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
		help: "Show detailed information about a handler: :handler <name>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :handler <name>")
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
}
