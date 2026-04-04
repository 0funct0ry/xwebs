package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

// Dispatcher coordinates the execution of handlers for a connection.
type Dispatcher struct {
	registry       *Registry
	conn           *ws.Connection
	templateEngine *template.Engine
	verbose        bool
	Log            func(string, ...interface{})
	Error          func(string, ...interface{})
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher(registry *Registry, conn *ws.Connection, engine *template.Engine, verbose bool) *Dispatcher {
	return &Dispatcher{
		registry:       registry,
		conn:           conn,
		templateEngine: engine,
		verbose:        verbose,
		Log: func(f string, a ...interface{}) {
			fmt.Printf(f, a...)
		},
		Error: func(f string, a ...interface{}) {
			fmt.Fprintf(os.Stderr, f, a...)
		},
	}
}

func (d *Dispatcher) log(f string, a ...interface{}) {
	if d.Log != nil {
		d.Log(f, a...)
	}
}

func (d *Dispatcher) errorf(f string, a ...interface{}) {
	if d.Error != nil {
		d.Error(f, a...)
	}
}

// Start begins the dispatch loop.
func (d *Dispatcher) Start(ctx context.Context) {
	// Subscribe to incoming messages
	msgCh := d.conn.Subscribe()

	go func() {
		defer d.conn.Unsubscribe(msgCh)

		for {
			select {
			case <-ctx.Done():
				return
			case <-d.conn.Done():
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				// Only handle received messages for matching
				if msg.Metadata.Direction == "received" {
					d.handleMessage(msg)
				}
			}
		}
	}()
}

func (d *Dispatcher) handleMessage(msg *ws.Message) {
	msgStr := string(msg.Data)
	if d.verbose {
		d.errorf("  [handler] debug: matching message %q (%v bytes)\n", msgStr, len(msg.Data))
	}

	matches, err := d.registry.Match(msg)
	if err != nil {
		if d.verbose {
			d.errorf("  [handler] error matching message: %v\n", err)
		}
		return
	}

	if d.verbose && len(matches) > 0 {
		d.errorf("  [handler] debug: found %d matches for %q\n", len(matches), msgStr)
	}

	for _, h := range matches {
		if d.verbose {
			d.errorf("  [handler] executing handler %q (priority %d)\n", h.Name, h.Priority)
		}
		d.executeActions(h.Actions, msg)
	}
}

func (d *Dispatcher) executeActions(actions []Action, msg *ws.Message) {
	// Root context for the handler execution
	tmplCtx := template.NewContext()
	d.populateContext(tmplCtx, msg)

	for _, a := range actions {
		if err := d.executeAction(a, tmplCtx); err != nil {
			if d.verbose {
				d.errorf("  [handler] action error: %v\n", err)
			}
		}
	}
}

func (d *Dispatcher) populateContext(tmplCtx *template.TemplateContext, msg *ws.Message) {
	if msg != nil {
		typeStr := "text"
		if msg.Type == ws.BinaryMessage {
			typeStr = "binary"
		} else if msg.Type == ws.PingMessage {
			typeStr = "ping"
		} else if msg.Type == ws.PongMessage {
			typeStr = "pong"
		}

		tmplCtx.Msg = &template.MessageContext{
			Type:      typeStr,
			Data:      string(msg.Data),
			Raw:       msg.Data,
			Length:    len(msg.Data),
			Timestamp: msg.Metadata.Timestamp,
		}
		tmplCtx.Message = string(msg.Data)
		tmplCtx.MessageBytes = msg.Data
		tmplCtx.MessageLen = len(msg.Data)
		tmplCtx.MessageType = typeStr
		tmplCtx.MessageIndex = msg.Metadata.MessageIndex
		tmplCtx.Timestamp = msg.Metadata.Timestamp
		tmplCtx.Direction = msg.Metadata.Direction
		tmplCtx.Last = string(msg.Data)
	}

	// Populate connection context
	tmplCtx.Conn = &template.ConnectionContext{
		URL:                d.conn.URL,
		Subprotocol:        d.conn.NegotiatedSubprotocol,
		CompressionEnabled: d.conn.IsCompressionEnabled(),
	}
}

func (d *Dispatcher) executeAction(a Action, tmplCtx *template.TemplateContext) error {
	switch strings.ToLower(a.Type) {
	case "shell":
		return d.executeShell(a, tmplCtx)
	case "send":
		return d.executeSend(a, tmplCtx)
	case "log":
		return d.executeLog(a, tmplCtx)
	case "builtin":
		return d.executeBuiltin(a, tmplCtx)
	default:
		return fmt.Errorf("unknown action type: %s", a.Type)
	}
}

func (d *Dispatcher) executeShell(a Action, ctx *template.TemplateContext) error {
	cmdStr, err := d.templateEngine.Execute("shell", a.Command, ctx)
	if err != nil {
		return fmt.Errorf("template error in shell command: %w", err)
	}

	start := time.Now()
	cmd := exec.Command("sh", "-c", cmdStr)
	
	var stdout, stderr strings.Builder
	if !a.Silent {
		cmd.Stdout = io.MultiWriter(&stdout, &funcWriter{f: d.log})
		cmd.Stderr = io.MultiWriter(&stderr, &funcWriter{f: d.errorf})
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err = cmd.Run()
	duration := time.Since(start)

	// Update context with execution results for subsequent actions
	ctx.Handler = &template.HandlerContext{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			ctx.Handler.ExitCode = exitErr.ExitCode()
		} else {
			ctx.Handler.ExitCode = -1
		}
		return fmt.Errorf("shell command failed: %w", err)
	}
	ctx.Handler.ExitCode = 0
	return nil
}

func (d *Dispatcher) executeSend(a Action, ctx *template.TemplateContext) error {
	msgStr, err := d.templateEngine.Execute("send", a.Message, ctx)
	if err != nil {
		return fmt.Errorf("template error in send message: %w", err)
	}

	return d.conn.Write(&ws.Message{
		Type: ws.TextMessage,
		Data: []byte(msgStr),
	})
}

func (d *Dispatcher) executeLog(a Action, ctx *template.TemplateContext) error {
	msgStr, err := d.templateEngine.Execute("log", a.Message, ctx)
	if err != nil {
		return fmt.Errorf("template error in log message: %w", err)
	}

	target := strings.ToLower(a.Target)
	switch target {
	case "stderr":
		d.errorf("  [handler] %s\n", msgStr)
	case "stdout", "":
		d.log("  [handler] %s\n", msgStr)
	default:
		// Log to file
		f, err := os.OpenFile(a.Target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("opening log file %s: %w", a.Target, err)
		}
		defer f.Close()
		fmt.Fprintf(f, "%s\n", msgStr)
	}
	return nil
}

func (d *Dispatcher) executeBuiltin(a Action, ctx *template.TemplateContext) error {
	cmdStr, err := d.templateEngine.Execute("builtin", a.Command, ctx)
	if err != nil {
		return fmt.Errorf("template error in builtin command: %w", err)
	}

	if d.verbose {
		d.errorf("  [handler] builtin command requested: %s\n", cmdStr)
	}
	// TODO: Integrate with REPL commands if possible
	return nil
}

// HandleConnect runs all on_connect actions in priority order.
func (d *Dispatcher) HandleConnect() {
	onConnect, _, _ := d.registry.LifecycleHandlers()
	d.sortHandlers(onConnect)

	for _, h := range onConnect {
		if d.verbose {
			d.errorf("  [handler] executing on_connect for handler %q\n", h.Name)
		}
		d.executeActions(h.OnConnect, nil)
	}
}

// HandleDisconnect runs all on_disconnect actions in priority order.
func (d *Dispatcher) HandleDisconnect() {
	_, onDisconnect, _ := d.registry.LifecycleHandlers()
	d.sortHandlers(onDisconnect)

	for _, h := range onDisconnect {
		if d.verbose {
			d.errorf("  [handler] executing on_disconnect for handler %q\n", h.Name)
		}
		d.executeActions(h.OnDisconnect, nil)
	}
}

// HandleError runs all on_error actions in priority order.
func (d *Dispatcher) HandleError(err error) {
	_, _, onError := d.registry.LifecycleHandlers()
	d.sortHandlers(onError)

	for _, h := range onError {
		if d.verbose {
			d.errorf("  [handler] executing on_error for handler %q: %v\n", h.Name, err)
		}
		// Create a temporary context to pass the error if possible
		tmplCtx := template.NewContext()
		d.populateContext(tmplCtx, nil)
		// Future: add err to context
		
		for _, a := range h.OnError {
			if err := d.executeAction(a, tmplCtx); err != nil {
				if d.verbose {
					d.errorf("  [handler] on_error action failure: %v\n", err)
				}
			}
		}
	}
}

func (d *Dispatcher) sortHandlers(hs []*Handler) {
	sort.SliceStable(hs, func(i, j int) bool {
		return hs[i].Priority > hs[j].Priority
	})
}

type funcWriter struct {
	f func(string, ...interface{})
}

func (w *funcWriter) Write(p []byte) (n int, err error) {
	w.f("%s", string(p))
	return len(p), nil
}
