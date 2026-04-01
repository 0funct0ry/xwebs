package repl

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

// ClientContext provides access to the active connection and environment for client-mode commands.
type ClientContext interface {
	GetConnection() *ws.Connection
	SetConnection(conn *ws.Connection)
	Dial(ctx context.Context, url string) error
	CloseConnection() error
	GetTemplateEngine() *template.Engine
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

func (c *DefaultClientContext) Dial(ctx context.Context, url string) error {
	return fmt.Errorf("dial not implemented in DefaultClientContext")
}

func (c *DefaultClientContext) CloseConnection() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *DefaultClientContext) GetTemplateEngine() *template.Engine {
	return nil
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
		name: "sendb",
		help: "Send binary message: :sendb <hex|base64:data>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			if len(args) == 0 {
				return fmt.Errorf("usage: :sendb <hex|base64:data>")
			}
			raw := strings.Join(args, "")
			var data []byte
			var err error
			if strings.HasPrefix(raw, "base64:") {
				data, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, "base64:"))
			} else {
				data, err = hex.DecodeString(raw)
			}
			if err != nil {
				return fmt.Errorf("decoding binary data: %w", err)
			}
			return conn.Write(&ws.Message{Type: ws.BinaryMessage, Data: data})
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "sendj",
		help: "Send JSON message: :sendj <json>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			if len(args) == 0 {
				return fmt.Errorf("usage: :sendj <json>")
			}
			msg := strings.Join(args, " ")
			if !json.Valid([]byte(msg)) {
				return fmt.Errorf("invalid JSON: %s", msg)
			}
			return conn.Write(&ws.Message{Type: ws.TextMessage, Data: []byte(msg)})
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "sendt",
		help: "Send rendered template: :sendt <template>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			if len(args) == 0 {
				return fmt.Errorf("usage: :sendt <template>")
			}
			tmpl := strings.Join(args, " ")
			engine := cc.GetTemplateEngine()
			if engine == nil {
				return fmt.Errorf("template engine not available")
			}
			
			tmplCtx := template.NewContext()
			tmplCtx.Session = r.GetVars()
			
			res, err := engine.Execute("repl", tmpl, tmplCtx)
			if err != nil {
				return fmt.Errorf("rendering template: %w", err)
			}
			return conn.Write(&ws.Message{Type: ws.TextMessage, Data: []byte(res)})
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "connect",
		help: "Connect to a new URL: :connect <url>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :connect <url>")
			}
			return cc.Dial(ctx, args[0])
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "reconnect",
		help: "Reconnect to the current URL",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			return cc.Dial(ctx, "") // Empty string means reconnect
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
			return cc.CloseConnection()
		},
	})
}
