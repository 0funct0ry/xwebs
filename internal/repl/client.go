package repl

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
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
	CloseConnectionWithCode(code int, reason string) error
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

func (c *DefaultClientContext) CloseConnectionWithCode(code int, reason string) error {
	if c.conn != nil {
		return c.conn.CloseWithCode(code, reason)
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
		help: "Send a ping frame: :ping [text|hex:data|base64:data]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			data, err := parsePayload(args)
			if err != nil {
				return err
			}
			return conn.Write(&ws.Message{Type: ws.PingMessage, Data: data})
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "pong",
		help: "Send a pong frame: :pong [text|hex:data|base64:data]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}
			data, err := parsePayload(args)
			if err != nil {
				return err
			}
			return conn.Write(&ws.Message{Type: ws.PongMessage, Data: data})
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

			code := 1000
			reason := "Normal Closure"

			if len(args) > 0 {
				// Try to parse first arg as a code
				if c, err := strconv.Atoi(args[0]); err == nil {
					code = c
					if len(args) > 1 {
						reason = strings.Join(args[1:], " ")
					}
				} else {
					// Treat all args as reason
					reason = strings.Join(args, " ")
				}
			}

			return cc.CloseConnectionWithCode(code, reason)
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

// parsePayload is a helper to parse CLI arguments into a byte slice,
// supporting hex: and base64: prefixes, defaulting to plain text.
func parsePayload(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, nil
	}
	raw := strings.Join(args, " ")
	if strings.HasPrefix(raw, "base64:") {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, "base64:"))
		if err != nil {
			return nil, fmt.Errorf("decoding base64: %w", err)
		}
		return data, nil
	}
	if strings.HasPrefix(raw, "hex:") {
		// Remove spaces from hex data if present
		hexStr := strings.ReplaceAll(strings.TrimPrefix(raw, "hex:"), " ", "")
		data, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("decoding hex: %w", err)
		}
		return data, nil
	}
	return []byte(raw), nil
}
