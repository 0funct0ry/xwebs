package repl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

// ServerContext provides access to the server state and administration for server-mode commands.
type ServerContext interface {
	GetClientCount() int
	GetUptime() time.Duration
	GetClients() []template.ClientInfo
	Broadcast(msg *ws.Message) error
	Kick(id string) error
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
			r.Printf("%-10s %-20s %-15s %s\n", "ID", "Remote Address", "Uptime", "Connected At")
			r.Printf("%s\n", strings.Repeat("-", 60))
			for _, c := range clients {
				r.Printf("%-10s %-20s %-15s %s\n",
					c.ID,
					c.RemoteAddr,
					c.UptimeStr,
					c.ConnectedAt.Format("15:04:05"),
				)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "broadcast",
		help: "Broadcast a message to all connected clients: :broadcast <message>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :broadcast <message>")
			}
			msgStr := strings.Join(args, " ")
			msg := &ws.Message{
				Type: ws.TextMessage,
				Data: []byte(msgStr),
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
		help: "Disconnect a specific client: :kick <id>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :kick <id>")
			}
			id := args[0]
			if err := sc.Kick(id); err != nil {
				return fmt.Errorf("failed to kick client %s: %w", id, err)
			}
			r.Printf("✓ Client %s kicked\n", id)
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
