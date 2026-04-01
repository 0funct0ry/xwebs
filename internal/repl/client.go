package repl

import (
	"context"
	"fmt"
	"strings"

	"github.com/0funct0ry/xwebs/internal/ws"
)

// ClientContext provides access to the active connection for client-mode commands.
type ClientContext interface {
	GetConnection() *ws.Connection
	SetConnection(conn *ws.Connection)
}

// DefaultClientContext is a simple implementation of ClientContext.
type DefaultClientContext struct {
	conn *ws.Connection
}

func (c *DefaultClientContext) GetConnection() *ws.Connection {
	return c.conn
}

func (c *DefaultClientContext) SetConnection(conn *ws.Connection) {
	c.conn = conn
}

// RegisterClientCommands adds WebSocket client-specific commands to the REPL.
func (r *REPL) RegisterClientCommands(cc ClientContext) {
	r.RegisterCommand(&BuiltinCommand{
		name: "send",
		help: "Send a message: :send <message>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			msg := strings.Join(args, " ")
			return conn.Write(&ws.Message{Type: ws.TextMessage, Data: []byte(msg)})
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "ping",
		help: "Send a ping message: :ping [data]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			data := strings.Join(args, " ")
			return conn.Write(&ws.Message{Type: ws.PingMessage, Data: []byte(data)})
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "pong",
		help: "Send a pong message: :pong [data]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			data := strings.Join(args, " ")
			return conn.Write(&ws.Message{Type: ws.PongMessage, Data: []byte(data)})
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "close",
		help: "Close the connection: :close [code] [reason]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}

			// Parse optional code and reason (simplistic version)
			return conn.Close()
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "status",
		help: "Show connection status",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil {
				r.Printf("No active connection.\n")
				return nil
			}

			r.Printf("\nConnection Status:\n")
			r.Printf("  URL:            %s\n", conn.URL)
			r.Printf("  Subprotocol:    %s\n", conn.NegotiatedSubprotocol)
			r.Printf("  Compression:    %v\n", conn.IsCompressionEnabled())
			r.Printf("  Closed:         %v\n", conn.IsClosed())
			if conn.IsClosed() {
				code, reason := conn.CloseStatus()
				r.Printf("  Close Code:     %d\n", code)
				r.Printf("  Close Reason:   %s\n", reason)
				if err := conn.Err(); err != nil {
					r.Printf("  Last Error:     %v\n", err)
				}
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "disconnect",
		help: "Disconnect from the current server",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				r.Printf("No active connection.\n")
				return nil
			}
			return conn.Close()
		},
	})
}
