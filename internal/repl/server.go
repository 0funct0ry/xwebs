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
		help: "List server-side handlers",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			handlers := sc.GetHandlers()
			if len(handlers) == 0 {
				r.Printf("No server-side handlers registered.\n")
				return nil
			}

			r.Printf("\nServer Handlers (%d):\n", len(handlers))
			r.Printf("%-20s %-10s %-15s %s\n", "Name", "Type", "Pattern", "Respond")
			r.Printf("%s\n", strings.Repeat("-", 60))
			for _, h := range handlers {
				hType := "match"
				if h.Match.Pattern == "*" {
					hType = "all"
				}
				r.Printf("%-20s %-10s %-15s %s\n",
					h.Name,
					hType,
					h.Match.Pattern,
					h.Respond,
				)
			}
			return nil
		},
	})
}
