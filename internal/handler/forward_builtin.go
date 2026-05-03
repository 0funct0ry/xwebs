package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

// ForwardBuiltin relays incoming messages to another WebSocket endpoint.
type ForwardBuiltin struct {
	conns   map[string]*ws.Connection
	connsMu sync.Mutex
}

func (b *ForwardBuiltin) Name() string { return "forward" }
func (b *ForwardBuiltin) Description() string {
	return "Relay the incoming message to another WebSocket endpoint and capture the reply."
}
func (b *ForwardBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *ForwardBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Relay the incoming message to another WebSocket endpoint and capture the reply.",
		Fields: []BuiltinField{
			{Name: "target", Type: "string", Required: true, Description: "Upstream WebSocket URL (supports templates)."},
			{Name: "timeout", Type: "string", Default: "30s", Required: false, Description: "Time to wait for upstream response."},
		},
		TemplateVars: map[string]string{
			".ForwardReply": "Captured response body from upstream",
		},
		YAMLReplExample: "builtin: forward\ntarget: 'ws://echo.websocket.org'\nrespond: 'Relayed: {{.ForwardReply}}'",
		REPLAddExample:  ":handler add -m '*' --builtin forward --target 'ws://echo.websocket.org' -R 'Echo: {{.ForwardReply}}'",
	}
}

func (b *ForwardBuiltin) Validate(a Action) error {
	if a.Target == "" {
		return fmt.Errorf("builtin forward missing target")
	}
	return nil
}

func (b *ForwardBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	target, err := d.templateEngine.Execute("forward-target", a.Target, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in target: %w", err)
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("builtin forward: target evaluates to empty string")
	}

	// Get or create persistent connection
	conn, err := b.getConn(ctx, d, target)
	if err != nil {
		return fmt.Errorf("connecting to upstream %s: %w", target, err)
	}

	// Determine original message type
	mt := ws.TextMessage
	if tmplCtx.MessageType == "binary" {
		mt = ws.BinaryMessage
	}

	// Create a subscriber for the response
	// Note: We subscribe BEFORE sending to avoid missing a very fast reply.
	replyCh := conn.Subscribe()
	defer conn.Unsubscribe(replyCh)

	// Send message to upstream
	if d.verbose {
		d.errorf("  [handler] builtin forward: sending message to %s\n", target)
	}
	err = conn.Write(&ws.Message{
		Type: mt,
		Data: tmplCtx.MessageBytes,
	})
	if err != nil {
		// Connection might have died, try once more with a fresh connection
		b.removeConn(target)
		conn, err = b.getConn(ctx, d, target)
		if err != nil {
			return fmt.Errorf("reconnecting to upstream after write failure: %w", err)
		}
		// Reset subscriber for the new connection
		replyCh = conn.Subscribe()
		defer conn.Unsubscribe(replyCh)

		err = conn.Write(&ws.Message{
			Type: mt,
			Data: tmplCtx.MessageBytes,
		})
		if err != nil {
			return fmt.Errorf("writing to upstream %s: %w", target, err)
		}
	}

	// Wait for reply with timeout
	timeout := 30 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("timeout waiting for upstream response from %s", target)
	case msg, ok := <-replyCh:
		if !ok {
			return fmt.Errorf("upstream connection closed while waiting for response")
		}
		// Capture the reply
		tmplCtx.ForwardReply = string(msg.Data)
		if d.verbose {
			d.errorf("  [handler] builtin forward: received %d bytes from %s\n", len(msg.Data), target)
		}
	}

	return nil
}

func (b *ForwardBuiltin) getConn(ctx context.Context, d *Dispatcher, target string) (*ws.Connection, error) {
	b.connsMu.Lock()
	defer b.connsMu.Unlock()

	if conn, ok := b.conns[target]; ok {
		// Check if connection is still active
		select {
		case <-conn.Done():
			// Connection is closed, remove it
			delete(b.conns, target)
		default:
			return conn, nil
		}
	}

	// Dial new connection
	if d.verbose {
		d.errorf("  [handler] builtin forward: dialing upstream %s\n", target)
	}
	conn, err := ws.Dial(ctx, target)
	if err != nil {
		return nil, err
	}

	if d.verbose {
		d.errorf("  [handler] builtin forward: successfully connected to %s\n", target)
	}
	b.conns[target] = conn
	return conn, nil
}

func (b *ForwardBuiltin) removeConn(target string) {
	b.connsMu.Lock()
	defer b.connsMu.Unlock()
	delete(b.conns, target)
}
