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

	"github.com/0funct0ry/xwebs/internal/replay"
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
			r.SetLastSendTime(time.Now())
			err := conn.Write(&ws.Message{Type: ws.TextMessage, Data: []byte(msg)})
			if err == nil {
				r.PrintMessage(&ws.Message{Type: ws.TextMessage, Data: []byte(msg), Metadata: ws.MessageMetadata{Direction: "sent"}}, conn)
			}
			return err
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
			r.SetLastSendTime(time.Now())
			m := &ws.Message{Type: ws.PingMessage, Data: data}
			err = conn.Write(m)
			if err == nil {
				r.PrintMessage(&ws.Message{Type: ws.PingMessage, Data: data, Metadata: ws.MessageMetadata{Direction: "sent"}}, conn)
			}
			return err
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
			m := &ws.Message{Type: ws.PongMessage, Data: data}
			err = conn.Write(m)
			if err == nil {
				r.PrintMessage(&ws.Message{Type: ws.PongMessage, Data: data, Metadata: ws.MessageMetadata{Direction: "sent"}}, conn)
			}
			return err
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
			r.SetLastSendTime(time.Now())
			m := &ws.Message{Type: ws.BinaryMessage, Data: data}
			err = conn.Write(m)
			if err == nil {
				r.PrintMessage(&ws.Message{Type: ws.BinaryMessage, Data: data, Metadata: ws.MessageMetadata{Direction: "sent"}}, conn)
			}
			return err
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
			r.SetLastSendTime(time.Now())
			m := &ws.Message{Type: ws.TextMessage, Data: []byte(msg)}
			err := conn.Write(m)
			if err == nil {
				r.PrintMessage(&ws.Message{Type: ws.TextMessage, Data: []byte(msg), Metadata: ws.MessageMetadata{Direction: "sent"}}, conn)
			}
			return err
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
			r.SetLastSendTime(time.Now())
			err = conn.Write(&ws.Message{Type: ws.TextMessage, Data: []byte(res)})
			if err == nil {
				r.PrintMessage(&ws.Message{Type: ws.TextMessage, Data: []byte(res), Metadata: ws.MessageMetadata{Direction: "sent"}}, conn)
			}
			return err
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "connect",
		help: "Connect to a new URL: :connect <url>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :connect <url>")
			}
			url := strings.Join(args, " ")
			if strings.Contains(url, "{{") {
				engine := cc.GetTemplateEngine()
				if engine != nil {
					tmplCtx := template.NewContext()
					tmplCtx.Session = r.GetVars()
					evaluated, err := engine.Execute("url", url, tmplCtx)
					if err != nil {
						return fmt.Errorf("evaluating URL template: %w", err)
					}
					url = evaluated
				}
			}
			return cc.Dial(ctx, url)
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

	r.RegisterCommand(&BuiltinCommand{
		name: "log",
		help: "Log traffic to file: :log <file> | :log off",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				if r.Logger.IsActive() {
					r.Printf("Logging to: %s\n", r.Logger.Filename())
				} else {
					r.Printf("log: off\n")
				}
				return nil
			}

			if args[0] == "off" {
				count, filename, err := r.Logger.Stop()
				if err != nil {
					return err
				}
				if filename != "" {
					r.Printf("✓ Stopped logging (%d entries written to %s)\n", count, filename)
				}
				return nil
			}

			if err := r.Logger.Start(args[0]); err != nil {
				return err
			}
			r.Printf("✓ Logging to %s\n", args[0])
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "record",
		help: "Record session to file: :record <file> | :record off",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				if r.Recorder.IsActive() {
					r.Printf("Recording active\n")
				} else {
					r.Printf("record: off\n")
				}
				return nil
			}

			if args[0] == "off" {
				count, filename, err := r.Recorder.Stop()
				if err != nil {
					return err
				}
				if filename != "" {
					r.Printf("✓ Stopped recording (%d messages captured to %s)\n", count, filename)
				}
				return nil
			}

			conn := cc.GetConnection()
			url := ""
			if conn != nil {
				url = conn.URL
			}

			if err := r.Recorder.Start(args[0], url); err != nil {
				return err
			}
			r.Printf("✓ Recording to %s\n", args[0])
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "replay",
		help: "Replay a recorded session: :replay <file>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :replay <file>")
			}
			conn := cc.GetConnection()
			if conn == nil || conn.IsClosed() {
				return fmt.Errorf("no active connection")
			}

			filename := args[0]
			rep := replay.NewReplayer()
			
			r.Printf("▶ Replaying messages from %s...\n", filename)
			
			// We use a background context or a derived one that can be cancelled?
			// The REPL's ExecuteCommand passes a context which is usually cmd.Context()
			
			sent, recv, err := rep.Replay(ctx, conn, filename, 1.0, func(elapsed int64, dir string, msg string) {
				r.Printf("  [%dms]  %s %s\n", elapsed, "→", msg)
			})
			
			if err != nil {
				return err
			}
			
			r.Printf("✓ Replay complete. %d sent, %d received.\n", sent, recv)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "mock",
		help: "Load mock scenario: :mock <file> | :mock off",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				r.Printf("Mock status: %s\n", r.Mocker.GetStatus())
				return nil
			}

			if args[0] == "off" {
				r.Mocker.Stop()
				r.Printf("✓ Mock unloaded\n")
				return nil
			}

			if err := r.Mocker.LoadScenario(args[0]); err != nil {
				return err
			}
			
			conn := cc.GetConnection()
			if conn != nil {
				r.Mocker.StartBackgroundTasks(ctx, conn, func(f string, a ...interface{}) {
					r.Notify(f+"\n", a...)
				})
			}
			
			r.Printf("✓ Mock loaded: %s\n", args[0])
			return nil
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
