package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/0funct0ry/xwebs/internal/shell"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

// Connection defines the required interface for a WebSocket connection.
type Connection interface {
	Write(msg *ws.Message) error
	Subscribe() <-chan *ws.Message
	Unsubscribe(ch <-chan *ws.Message)
	Done() <-chan struct{}
	IsCompressionEnabled() bool
	GetURL() string
	GetSubprotocol() string
	RemoteAddr() string
	LocalAddr() string
	ConnectedAt() time.Time
	MessageCount() uint64
	MsgsIn() uint64
	MsgsOut() uint64
	LastMsgReceivedAt() time.Time
	LastMsgSentAt() time.Time
	RTT() time.Duration
	AvgRTT() time.Duration
}

// Dispatcher coordinates the execution of handlers for a connection.
type Dispatcher struct {
	registry       *Registry
	conn           Connection
	templateEngine *template.Engine
	verbose        bool
	Log            func(string, ...interface{})
	Error          func(string, ...interface{})

	variables        map[string]interface{}
	sessionVariables map[string]interface{}
	systemEnv        map[string]string
	sandbox          bool
	allowlist        []string

	_handlerHits    uint64
	_activeHandlers int32
}

// NewDispatcher creates a new dispatcher.
// NewDispatcher creates a new dispatcher.
func NewDispatcher(registry *Registry, conn Connection, engine *template.Engine, verbose bool, vars map[string]interface{}, session map[string]interface{}, sandbox bool, allowlist []string) *Dispatcher {

	// Initialize system environment
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}

	return &Dispatcher{
		registry:         registry,
		conn:             conn,
		templateEngine:   engine,
		verbose:          verbose,
		variables:        vars,
		sessionVariables: session,
		systemEnv:        env,
		sandbox:          sandbox,
		allowlist:        allowlist,
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
					d.handleMessage(ctx, msg)
				}
			}
		}
	}()
}

func (d *Dispatcher) handleMessage(ctx context.Context, msg *ws.Message) {
	msgStr := string(msg.Data)
	if d.verbose {
		d.errorf("  [handler] debug: matching message %q (%v bytes)\n", msgStr, len(msg.Data))
	}

	// Populate context once for all handlers matching this message
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)

	matches, err := d.registry.Match(msg, d.templateEngine, tmplCtx)
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

		// Apply rate limiting
		if h.RateLimit != "" {
			limiter := d.registry.GetLimiter(h.Name, h.RateLimit)
			if limiter != nil && !limiter.Allow() {
				if d.verbose {
					d.errorf("  [handler] warning: rate limit exceeded for %q (%s), dropping message\n", h.Name, h.RateLimit)
				}
				continue
			}
		}

		// Apply debounce
		if h.Debounce != "" {
			dur, _ := time.ParseDuration(h.Debounce)
			d.registry.Debounce(h.Name, dur, msg, func(m *ws.Message) {
				if d.verbose {
					d.errorf("  [handler] executing debounced handler %q\n", h.Name)
				}
				// Use context from Dispatcher.Start which is passed as ctx
				if err := d.Execute(ctx, h, m); err != nil {
					d.errorf("  [handler] error executing debounced %q: %v\n", h.Name, err)
				}
			})
			continue
		}

		go func(handler Handler) {
			if err := d.Execute(ctx, &handler, msg); err != nil {
				d.errorf("  [handler] error executing %q: %v\n", handler.Name, err)
			}
		}(*h)

		// If the handler is exclusive, stop processing further handlers.
		if h.Exclusive {
			break
		}
	}
}

// Execute runs the actions defined in a handler.
func (d *Dispatcher) Execute(ctx context.Context, h *Handler, msg *ws.Message) error {
	atomic.AddUint64(&d._handlerHits, 1)
	atomic.AddInt32(&d._activeHandlers, 1)
	defer atomic.AddInt32(&d._activeHandlers, -1)

	// Handle concurrency control
	if h.Concurrent != nil && !*h.Concurrent {
		mu := d.registry.GetHandlerMu(h.Name)
		mu.Lock()
		defer mu.Unlock()
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)

	// Merge and evaluate per-handler variables
	if h.Variables != nil {
		evaluated := evaluateVariables(d.templateEngine, h.Variables, tmplCtx, d.verbose, d.Error)
		for k, v := range evaluated {
			tmplCtx.Vars[k] = v
		}
	}

	// Add pipeline steps map
	tmplCtx.Steps = make(map[string]*template.HandlerContext)

	// Determine execution attempts
	maxAttempts := 1
	if h.Retry != nil && h.Retry.Count > 0 {
		maxAttempts = 1 + h.Retry.Count
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = d.executeMainActions(ctx, h, tmplCtx, msg)

		// Check for failure to trigger retry or HandleError
		isFailure := lastErr != nil
		// For concise models (run/builtin), we must manually check ExitCode
		if !isFailure && (h.Run != "" || h.Builtin != "") {
			if tmplCtx.ExitCode != 0 {
				isFailure = true
				lastErr = fmt.Errorf("command failed with exit code %d", tmplCtx.ExitCode)
			}
		}

		if !isFailure {
			break
		}

		// Final attempt failed
		if attempt >= maxAttempts {
			if d.verbose && maxAttempts > 1 {
				d.errorf("  [handler] error: final failure for %q after %d attempts: %v\n", h.Name, attempt, lastErr)
			}
			break
		}

		// Calculate backoff and wait
		backoff := d.calculateBackoff(h.Retry, attempt)
		if d.verbose {
			d.errorf("  [handler] error executing %q (attempt %d/%d): %v. Retrying in %v...\n",
				h.Name, attempt, maxAttempts, lastErr, backoff)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	// Always execute respond if present (after all main actions/retries)
	// Concise handlers always run respond if present (it acts as a completion hook).
	// Pipelines and multi-action handlers only run respond if all steps succeeded.
	isConcise := h.Run != "" || h.Builtin != ""
	if h.Respond != "" && (lastErr == nil || isConcise) {
		action := Action{Type: "send", Message: h.Respond}
		if err := d.ExecuteAction(ctx, &action, tmplCtx, msg); err != nil {
			return err
		}
	}

	if lastErr != nil {
		d.HandleError(lastErr)
	}
	return lastErr
}

// executeMainActions runs the core logic of a handler (Actions, Pipeline, Run, or Builtin).
// It does NOT run Respond:, which is handled by the caller (the retry loop).
func (d *Dispatcher) executeMainActions(ctx context.Context, h *Handler, tmplCtx *template.TemplateContext, msg *ws.Message) error {
	if len(h.Actions) > 0 {
		// Legacy action list
		for _, action := range h.Actions {
			if err := d.ExecuteAction(ctx, &action, tmplCtx, msg); err != nil {
				return err
			}
		}
	} else if len(h.Pipeline) > 0 {
		// New pipeline model
		return d.executePipeline(ctx, h.Pipeline, tmplCtx, msg)
	} else {
		// Concise top-level model (run or builtin)
		if h.Run != "" {
			action := Action{Type: "shell", Run: h.Run, Timeout: h.Timeout}
			return d.ExecuteAction(ctx, &action, tmplCtx, msg)
		} else if h.Builtin != "" {
			action := Action{Type: "builtin", Command: h.Builtin, Timeout: h.Timeout}
			return d.ExecuteAction(ctx, &action, tmplCtx, msg)
		}
	}
	return nil
}

func (d *Dispatcher) calculateBackoff(cfg *RetryConfig, attempt int) time.Duration {
	interval := 1 * time.Second
	if cfg.Interval != "" {
		if dur, err := time.ParseDuration(cfg.Interval); err == nil {
			interval = dur
		}
	}

	var wait time.Duration
	if strings.ToLower(cfg.Backoff) == "exponential" {
		// interval * 2^(attempt-1)
		wait = interval * time.Duration(1<<(attempt-1))
		if cfg.MaxInterval != "" {
			if max, err := time.ParseDuration(cfg.MaxInterval); err == nil && wait > max {
				wait = max
			}
		} else if wait > 30*time.Second {
			wait = 30 * time.Second
		}
	} else {
		// Linear backoff: interval * attempt
		wait = interval * time.Duration(attempt)
	}

	// Return calculated wait duration
	return wait
}

// executePipeline runs a sequence of steps.
func (d *Dispatcher) executePipeline(ctx context.Context, pipeline []PipelineStep, tmplCtx *template.TemplateContext, msg *ws.Message) error {
	for i, step := range pipeline {
		action := Action{Timeout: step.Timeout}
		if step.Run != "" {
			action.Type = "shell"
			action.Run = step.Run
		} else if step.Builtin != "" {
			action.Type = "builtin"
			action.Command = step.Builtin
		}

		if err := d.ExecuteAction(ctx, &action, tmplCtx, msg); err != nil {
			return err
		}

		// Store result if named
		if step.As != "" {
			tmplCtx.Steps[step.As] = &template.HandlerContext{
				Stdout:   tmplCtx.Stdout,
				Stderr:   tmplCtx.Stderr,
				ExitCode: tmplCtx.ExitCode,
				Duration: time.Duration(tmplCtx.DurationMs) * time.Millisecond,
			}
		}

		// Check for pipeline step failure unless ignored
		if tmplCtx.ExitCode != 0 && !step.IgnoreError {
			stepName := step.As
			if stepName == "" {
				stepName = step.Run
				if stepName == "" {
					stepName = fmt.Sprintf("step[%d]", i)
				}
			}
			if d.verbose {
				d.errorf("  [handler] pipeline step %q failed with exit code %d\n", stepName, tmplCtx.ExitCode)
			}
			return fmt.Errorf("pipeline step %q failed: exit code %d", stepName, tmplCtx.ExitCode)
		}
	}
	return nil
}

func (d *Dispatcher) populateTemplateContext(tmplCtx *template.TemplateContext, msg *ws.Message) {
	// Inject system environment
	if d.systemEnv != nil {
		for k, v := range d.systemEnv {
			tmplCtx.Env[k] = v
		}
	}

	// Inject session variables
	if d.sessionVariables != nil {
		for k, v := range d.sessionVariables {
			tmplCtx.Session[k] = v
		}
	}

	// Inject and evaluate global variables
	if d.variables != nil {
		evaluated := evaluateVariables(d.templateEngine, d.variables, tmplCtx, d.verbose, d.Error)
		for k, v := range evaluated {
			tmplCtx.Vars[k] = v
		}
	}

	if msg != nil {
		typeStr := "text"
		if msg.Type == ws.BinaryMessage {
			typeStr = "binary"
		} else if msg.Type == ws.PingMessage {
			typeStr = "ping"
		} else if msg.Type == ws.PongMessage {
			typeStr = "pong"
		}

		var parsedData interface{}
		if err := json.Unmarshal(msg.Data, &parsedData); err != nil {
			parsedData = string(msg.Data)
		}

		tmplCtx.Msg = &template.MessageContext{
			Type:      typeStr,
			Data:      parsedData,
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
	if d.conn != nil {
		tmplCtx.URL = d.conn.GetURL()
		u, err := url.Parse(d.conn.GetURL())
		if err == nil {
			tmplCtx.Host = u.Host
			tmplCtx.Path = u.Path
			tmplCtx.Scheme = u.Scheme
		}
		tmplCtx.Subprotocol = d.conn.GetSubprotocol()
		tmplCtx.RemoteAddr = d.conn.RemoteAddr()
		tmplCtx.LocalAddr = d.conn.LocalAddr()
		tmplCtx.ConnectedSince = d.conn.ConnectedAt()
		tmplCtx.Uptime = time.Since(d.conn.ConnectedAt())
		tmplCtx.UptimeFormatted = template.FormatUptime(tmplCtx.Uptime)
		tmplCtx.MessageCount = d.conn.MessageCount()

		tmplCtx.HandlerHits = atomic.LoadUint64(&d._handlerHits)
		tmplCtx.ActiveHandlers = int(atomic.LoadInt32(&d._activeHandlers))

		tmplCtx.Conn = &template.ConnectionContext{
			URL:                d.conn.GetURL(),
			Subprotocol:        d.conn.GetSubprotocol(),
			CompressionEnabled: d.conn.IsCompressionEnabled(),
			RemoteAddr:         tmplCtx.RemoteAddr,
			LocalAddr:          tmplCtx.LocalAddr,
			ConnectedAt:        tmplCtx.ConnectedSince,
			Uptime:             tmplCtx.Uptime,
			UptimeFormatted:    tmplCtx.UptimeFormatted,
			MessageCount:       tmplCtx.MessageCount,
			MsgsIn:             d.conn.MsgsIn(),
			MsgsOut:            d.conn.MsgsOut(),
			LastMsgReceivedAt:  d.conn.LastMsgReceivedAt(),
			LastMsgSentAt:      d.conn.LastMsgSentAt(),
			RTT:                d.conn.RTT(),
			AvgRTT:             d.conn.AvgRTT(),
		}
	}
}

// evaluateVariables resolves template expressions in a map of variables.
func evaluateVariables(engine *template.Engine, vars map[string]interface{}, ctx *template.TemplateContext, verbose bool, errLogger func(string, ...interface{})) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range vars {
		result[k] = v
	}

	// Max 3 passes to resolve inter-variable dependencies
	for pass := 0; pass < 3; pass++ {
		changed := false

		// Sort keys for deterministic evaluation
		keys := make([]string, 0, len(result))
		for k := range result {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			originalV := vars[k]
			if s, ok := originalV.(string); ok && strings.Contains(s, "{{") {
				evaluated, err := engine.Execute(k, s, ctx)
				if err == nil {
					if evaluated != result[k] {
						result[k] = evaluated
						if ctx.Vars != nil {
							ctx.Vars[k] = evaluated
						}
						changed = true
					}
				} else if verbose && errLogger != nil && pass == 2 {
					// Only log error on the final pass
					errLogger("  [handler] error evaluating variable %q: %v\n", k, err)
				}
			} else {
				// Static value
				if ctx.Vars != nil {
					ctx.Vars[k] = originalV
				}
			}
		}
		if !changed {
			break
		}
	}
	return result
}

func (d *Dispatcher) ExecuteAction(ctx context.Context, a *Action, tmplCtx *template.TemplateContext, msg *ws.Message) error {
	switch strings.ToLower(a.Type) {
	case "shell":
		return d.executeShell(ctx, a, tmplCtx)
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

func (d *Dispatcher) executeShell(ctx context.Context, a *Action, tmplCtx *template.TemplateContext) error {
	runStr := a.Run
	if runStr == "" {
		runStr = a.Command
	}

	cmdStr, err := d.templateEngine.Execute("shell", runStr, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in shell command: %w", err)
	}

	// Parse timeout
	timeout := 30 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		} else {
			d.errorf("  [handler] warning: invalid timeout %q, using default 30s\n", a.Timeout)
		}
	}

	// Create context with timeout
	childCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare stdin from message data
	var stdin io.Reader
	if tmplCtx.Msg != nil && tmplCtx.Msg.Raw != nil {
		stdin = bytes.NewReader(tmplCtx.Msg.Raw)
	}

	// Execute shell command
	var shellAllowlist []string
	if d.sandbox {
		shellAllowlist = d.allowlist
		if shellAllowlist == nil {
			shellAllowlist = []string{} // Empty list means deny all in sandbox mode
		}
	}
	result, err := shell.Execute(childCtx, cmdStr, stdin, a.Env, shellAllowlist)

	// Update template context with execution results for subsequent actions
	tmplCtx.Handler = &template.HandlerContext{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		Duration: result.Duration,
	}
	// Also top-level for spec compliance
	tmplCtx.Stdout = result.Stdout
	tmplCtx.Stderr = result.Stderr
	tmplCtx.ExitCode = result.ExitCode
	tmplCtx.DurationMs = int64(result.Duration / time.Millisecond)

	// Log output if not silent
	if !a.Silent {
		if result.Stdout != "" {
			d.log("%s", result.Stdout)
		}
		if result.Stderr != "" {
			d.errorf("%s", result.Stderr)
		}
	}

	if err != nil {
		if d.verbose {
			d.errorf("  [handler] shell command execution error: %v\n", err)
		}
		return err
	}
	return nil

}

func (d *Dispatcher) executeSend(a *Action, ctx *template.TemplateContext) error {
	msgStr, err := d.templateEngine.Execute("send", a.Message, ctx)
	if err != nil {
		return fmt.Errorf("template error in send message: %w", err)
	}

	return d.conn.Write(&ws.Message{
		Type: ws.TextMessage,
		Data: []byte(msgStr),
	})
}

func (d *Dispatcher) executeLog(a *Action, ctx *template.TemplateContext) error {
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

func (d *Dispatcher) executeBuiltin(a *Action, ctx *template.TemplateContext) error {
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
		d.log("  [hook] on_connect: %s\n", h.Name)

		tmplCtx := template.NewContext()
		d.populateTemplateContext(tmplCtx, nil)

		for _, a := range h.OnConnect {
			if err := d.ExecuteAction(context.Background(), &a, tmplCtx, nil); err != nil {
				d.errorf("  [hook] error in on_connect for %s: %v\n", h.Name, err)
			}
		}
	}
}

// HandleDisconnect runs all on_disconnect actions in priority order.
func (d *Dispatcher) HandleDisconnect() {
	_, onDisconnect, _ := d.registry.LifecycleHandlers()
	d.sortHandlers(onDisconnect)

	for _, h := range onDisconnect {
		d.log("  [hook] on_disconnect: %s\n", h.Name)

		tmplCtx := template.NewContext()
		d.populateTemplateContext(tmplCtx, nil)

		for _, a := range h.OnDisconnect {
			if err := d.ExecuteAction(context.Background(), &a, tmplCtx, nil); err != nil {
				d.errorf("  [hook] error in on_disconnect for %s: %v\n", h.Name, err)
			}
		}
	}
}

// HandleError runs all on_error actions in priority order.
func (d *Dispatcher) HandleError(err error) {
	_, _, onError := d.registry.LifecycleHandlers()
	d.sortHandlers(onError)

	for _, h := range onError {
		d.log("  [hook] on_error: %s (%v)\n", h.Name, err)

		tmplCtx := template.NewContext()
		d.populateTemplateContext(tmplCtx, nil)
		tmplCtx.Error = err.Error()

		for _, a := range h.OnError {
			if err := d.ExecuteAction(context.Background(), &a, tmplCtx, nil); err != nil {
				d.errorf("  [hook] error in on_error for %s: %v\n", h.Name, err)
			}
		}
	}
}

func (d *Dispatcher) sortHandlers(hs []*Handler) {
	sort.SliceStable(hs, func(i, j int) bool {
		return hs[i].Priority > hs[j].Priority
	})
}

// HandlerHits returns the total number of handler executions.
func (d *Dispatcher) HandlerHits() uint64 {
	return atomic.LoadUint64(&d._handlerHits)
}

// ActiveHandlers returns the number of currently executing handlers.
func (d *Dispatcher) ActiveHandlers() int32 {
	return atomic.LoadInt32(&d._activeHandlers)
}
