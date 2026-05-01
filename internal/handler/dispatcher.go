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
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/0funct0ry/xwebs/internal/shell"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

// Connection defines the required interface for a WebSocket connection.
type Connection interface {
	Write(msg *ws.Message) error
	CloseWithCode(code int, reason string) error
	Subscribe() <-chan *ws.Message
	Unsubscribe(ch <-chan *ws.Message)
	Done() <-chan struct{}
	IsCompressionEnabled() bool
	GetID() string
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

// ServerStatProvider defines the required interface for providing server-level statistics.
type ServerStatProvider interface {
	GetClientCount() int
	GetUptime() time.Duration
	GetClients() []template.ClientInfo
	IsPaused() bool
	WaitIfPaused()
	Broadcast(msg *ws.Message, excludeIDs ...string) int
	Send(id string, msg *ws.Message) error
	SendToSSE(stream, event, data, id string) error
	UpdateSSEStreamConfig(stream, onNoConsumers string, bufferSize int) error
	RegisterHTTPMock(path string, mock template.HTTPMockResponse) error
}

// TopicManager defines the required interface for pub/sub topic operations used
// by the dispatcher when executing subscribe/unsubscribe/publish builtins.
type TopicManager interface {
	Subscribe(connID string, conn Connection, topic string)
	Unsubscribe(connID, topic string) int
	Publish(topic string, msg *ws.Message) (int, error)
	PublishSticky(topic string, msg *ws.Message) (int, error)
	ClearRetained(topic string)
}

// KVManager defines the required interface for key-value store operations.
type KVManager interface {
	ListKV() map[string]interface{}
	GetKV(key string) (interface{}, bool)
	SetKV(key string, val interface{}, ttl time.Duration)
	DeleteKV(key string)
}
 
type RedisManager interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Del(ctx context.Context, key string) error
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
	PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPop(ctx context.Context, key string) (string, error)
	Incr(ctx context.Context, key string, by int64) (int64, error)
	Close() error
}

// Dispatcher coordinates the execution of handlers for a connection.
type Dispatcher struct {
	registry       *Registry
	conn           Connection
	templateEngine *template.Engine
	verbose        bool
	Log            func(string, ...interface{})
	Error          func(string, ...interface{})
	ollamaURL      string

	variables        map[string]interface{}
	sessionVariables map[string]interface{}
	systemEnv        map[string]string
	sandbox          bool
	allowlist        []string

	_handlerHits    uint64
	_activeHandlers int32
	serverStats     ServerStatProvider
	topicManager    TopicManager
	kvManager       KVManager
	redisManager    RedisManager
 
	bufferMu sync.Mutex
	buffer   []*ws.Message
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher(registry *Registry, conn Connection, engine *template.Engine, verbose bool, vars map[string]interface{}, session map[string]interface{}, sandbox bool, allowlist []string, serverStats ServerStatProvider, topicManager TopicManager, kvManager KVManager, redisManager RedisManager, ollamaURL string) *Dispatcher {

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
		sandbox:          sandbox,
		allowlist:        allowlist,
		serverStats:      serverStats,
		topicManager:     topicManager,
		kvManager:        kvManager,
		redisManager:     redisManager,
		ollamaURL:        ollamaURL,
		systemEnv:        env,
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

	// Flush goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-d.conn.Done():
				return
			default:
				if d.serverStats != nil {
					d.serverStats.WaitIfPaused()
				}
				d.flushBuffer(ctx)
				time.Sleep(10 * time.Millisecond) // Don't churn if resumed
			}
		}
	}()

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
					if d.serverStats != nil && d.serverStats.IsPaused() {
						d.addToBuffer(msg)
					} else {
						d.handleMessage(ctx, msg)
					}
				}
			}
		}
	}()
}

func (d *Dispatcher) addToBuffer(msg *ws.Message) {
	d.bufferMu.Lock()
	defer d.bufferMu.Unlock()
	d.buffer = append(d.buffer, msg)
	if d.verbose {
		d.errorf("  [handler] debug: server paused, buffering message (%d in buffer)\n", len(d.buffer))
	}
}

func (d *Dispatcher) flushBuffer(ctx context.Context) {
	d.bufferMu.Lock()
	if len(d.buffer) == 0 {
		d.bufferMu.Unlock()
		return
	}

	pending := d.buffer
	d.buffer = nil
	d.bufferMu.Unlock()

	if d.verbose {
		d.errorf("  [handler] debug: server resumed, flushing %d buffered messages\n", len(pending))
	}

	for _, msg := range pending {
		d.handleMessage(ctx, msg)
	}
}

func (d *Dispatcher) handleMessage(ctx context.Context, msg *ws.Message) {
	msgStr := string(msg.Data)
	if d.verbose {
		d.errorf("  [handler] debug: matching message %q (%v bytes)\n", msgStr, len(msg.Data))
	}

	// Populate context once for all handlers matching this message
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)

	results, err := d.registry.Match(msg, d.templateEngine, tmplCtx)
	if err != nil {
		if d.verbose {
			d.errorf("  [handler] error matching message: %v\n", err)
		}
		return
	}

	if len(results) == 0 {
		if d.verbose {
			d.errorf("  [handler] debug: no handlers matched message %q\n", msgStr)
		}
		return
	}

	if d.verbose {
		d.errorf("  [handler] debug: found %d matches for %q\n", len(results), msgStr)
	}

	// Execute matching handlers sequentially in a single goroutine per message.
	// This allows 'drop' actions and 'exclusive' handlers to short-circuit
	// subsequent handlers for the same message.
	go func() {
		for _, res := range results {
			h := res.Handler
			matches := res.Matches

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
					// Note: Debounced handlers execute in their own async flow
					if err := d.Execute(ctx, h, m, matches); err != nil {
						if err != ErrDrop {
							d.errorf("  [handler] error executing debounced %q: %v\n", h.Name, err)
						}
					}
				})
				// Debounce doesn't block the sequence but effectively replaces current execution
				continue
			}

			err := d.Execute(ctx, h, msg, matches)
			if err == ErrDrop {
				if d.verbose {
					d.errorf("  [handler] short-circuiting handlers: message dropped by %q\n", h.Name)
				}
				break
			}
			if err != nil {
				d.errorf("  [handler] error executing %q: %v\n", h.Name, err)
			}

			// If the handler is exclusive, stop processing further handlers.
			if h.Exclusive {
				if d.verbose {
					d.errorf("  [handler] short-circuiting handlers: %q is exclusive\n", h.Name)
				}
				break
			}
		}
	}()
}

// Execute runs the actions defined in a handler.
func (d *Dispatcher) Execute(ctx context.Context, h *Handler, msg *ws.Message, matches []string) error {
	atomic.AddUint64(&d._handlerHits, 1)
	atomic.AddInt32(&d._activeHandlers, 1)
	defer atomic.AddInt32(&d._activeHandlers, -1)

	// Measure execution time
	start := time.Now()
	var lastErr error
	defer func() {
		d.registry.RecordExecution(h.Name, time.Since(start), lastErr)
	}()

	// Handle concurrency control
	if h.Concurrent != nil && !*h.Concurrent {
		mu := d.registry.GetHandlerMu(h.Name)
		mu.Lock()
		defer mu.Unlock()
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)
	tmplCtx.Matches = matches

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

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = d.executeMainActions(ctx, h, tmplCtx, msg)

		// Short-circuit on rate limit exceeded or drop signal
		if lastErr == ErrLimitExceeded || lastErr == ErrDrop {
			if d.verbose && lastErr == ErrDrop {
				d.errorf("  [handler] short-circuiting actions in %q due to drop signal\n", h.Name)
			}
			return lastErr
		}

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
	// NOTE: For concise handlers (h.Run or h.Builtin), Respond is now handled
	// INSIDE ExecuteAction to maintain consistency. We only send here if it's
	// NOT a concise handler and we have a top-level Respond to send.
	isConcise := h.Run != "" || h.Builtin != ""
	if h.Respond != "" && !isConcise && (lastErr == nil) {
		action := Action{Type: "send", Message: h.Respond}
		if err := d.ExecuteAction(ctx, &action, tmplCtx, msg); err != nil {
			// We track respondent error as well
			lastErr = err
			return err
		}
	}

	if lastErr != nil {
		d.executeHandlerError(ctx, h, tmplCtx, lastErr)
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
			action.HandlerName = h.Name
			if err := d.ExecuteAction(ctx, &action, tmplCtx, msg); err != nil {
				return err
			}
		}
	} else if len(h.Pipeline) > 0 {
		// New pipeline model
		return d.executePipeline(ctx, h.Name, h.Pipeline, tmplCtx, msg)
	} else {
		// Concise top-level model (run or builtin)
		if h.Run != "" {
			action := Action{
				Type:        "shell",
				Run:         h.Run,
				Timeout:     h.Timeout,
				Delay:       h.Delay,
				Respond:     h.Respond,
				Loop:        h.Loop,
				PerClient:   h.PerClient,
				Scope:       h.Scope,
				OnLimit:     h.OnLimit,
				Rules:       h.Rules,
				BaseDir:     h.BaseDir,
				HandlerName: h.Name,
			}
			return d.ExecuteAction(ctx, &action, tmplCtx, msg)
		} else if h.Builtin != "" {
			action := Action{
				Type:        "builtin",
				Command:     h.Builtin,
				Topic:       h.Topic,
				Key:         h.Key,
				Value:       h.Value,
				By:          h.By,
				Secret:      h.Secret,
				Channel:     h.Channel,
				Target:      h.Target,
				Message:     h.Message,
				Timeout:     h.Timeout,
				Delay:       h.Delay,
				Respond:     h.Respond,
				TTL:         h.TTL,
				Default:     h.Default,
				Responses:   h.Responses,
				Loop:        h.Loop,
				PerClient:   h.PerClient,
				File:        h.File,
				Path:        h.Path,
				Content:     h.Content,
				Mode:        h.Mode,
				Rate:        h.Rate,
				Burst:       h.Burst,
				Scope:       h.Scope,
				OnLimit:     h.OnLimit,
				Expect:      h.Expect,
				OnClosed:    h.OnClosed,
				Window:      h.Window,
				Duration:    h.Duration,
				Max:         h.Max,
				Code:        h.Code,
				Reason:      h.Reason,
				URL:         h.URL,
				Method:      h.Method,
				Headers:     h.Headers,
				Body:        h.Body,
				Status:      h.Status,
				Name:        h.Name,
				Labels:      h.Labels,
				Script:      h.Script,
				MaxMemory:   h.MaxMemory,
				Targets:     h.Targets,
				Pool:        h.Pool,
				OnEmpty:     h.OnEmpty,
				Field:       h.Field,
				Split:       h.Split,
				HandlerA:    h.HandlerA,
				HandlerB:    h.HandlerB,
				Rules:       h.Rules,
				Query:       h.Query,
				Variables:   h.GraphQLVariables,
				Model:       h.Model,
				Prompt:      h.Prompt,
				OllamaURL:   h.OllamaURL,
				Stream:      h.Stream,
				Event:       h.Event,
				ID:          h.ID,
				OnNoConsumers: h.OnNoConsumers,
				System:      h.System,
				Input:       h.Input,
				BufferSize:  h.BufferSize,
				BaseDir:     h.BaseDir,
				HandlerName: h.Name,
			}
			return d.ExecuteAction(ctx, &action, tmplCtx, msg)
		} else if h.Script != "" || h.File != "" {
			action := Action{
				Type:        "builtin",
				Command:     "lua",
				Script:      h.Script,
				File:        h.File,
				MaxMemory:   h.MaxMemory,
				Timeout:     h.Timeout,
				Delay:       h.Delay,
				Respond:     h.Respond,
				BaseDir:     h.BaseDir,
				HandlerName: h.Name,
			}
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
func (d *Dispatcher) executePipeline(ctx context.Context, handlerName string, pipeline []PipelineStep, tmplCtx *template.TemplateContext, msg *ws.Message) error {
	for i, step := range pipeline {
		action := Action{
			Timeout: step.Timeout,
			Delay:   step.Delay,
			Respond: step.Respond,
		}
		if step.Run != "" {
			action.Type = "shell"
			action.Run = step.Run
		} else if step.Builtin != "" {
			action.Type = "builtin"
			action.Command = step.Builtin
			action.Topic = step.Topic
			action.Key = step.Key
			action.Value = step.Value
			action.By = step.By
			action.Secret = step.Secret
			action.Channel = step.Channel
			action.Target = step.Target
			action.Message = step.Message
			action.TTL = step.TTL
			action.Default = step.Default
			action.Responses = step.Responses
			action.Loop = step.Loop
			action.PerClient = step.PerClient
			action.File = step.File
			action.Path = step.Path
			action.Content = step.Content
			action.Mode = step.Mode
			action.Rate = step.Rate
			action.Burst = step.Burst
			action.Scope = step.Scope
			action.OnLimit = step.OnLimit
			action.Expect = step.Expect
			action.OnClosed = step.OnClosed
			action.Window = step.Window
			action.Duration = step.Duration
			action.Max = step.Max
			action.Code = step.Code
			action.Reason = step.Reason
			action.URL = step.URL
			action.Method = step.Method
			action.Headers = step.Headers
			action.Body = step.Body
			action.Status = step.Status
			action.Name = step.Name
			action.Labels = step.Labels
			action.Script = step.Script
			action.MaxMemory = step.MaxMemory
			action.Targets = step.Targets
			action.Pool = step.Pool
			action.OnEmpty = step.OnEmpty
			action.Field = step.Field
			action.Split = step.Split
			action.HandlerA = step.HandlerA
			action.HandlerB = step.HandlerB
			action.Rules = step.Rules
			action.Query = step.Query
			action.Variables = step.Variables
			action.Model = step.Model
			action.Prompt = step.Prompt
			action.OllamaURL = step.OllamaURL
			action.Stream = step.Stream
			action.Event = step.Event
			action.ID = step.ID
			action.OnNoConsumers = step.OnNoConsumers
			action.System = step.System
			action.Input = step.Input
			action.BufferSize = step.BufferSize
			action.BaseDir = d.registry.GetHandlerBaseDir(handlerName)
			action.HandlerName = handlerName
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
		tmplCtx.ConnectionID = d.conn.GetID()
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

	// Populate server context if available
	if d.serverStats != nil {
		count := d.serverStats.GetClientCount()
		clients := d.serverStats.GetClients()
		uptime := d.serverStats.GetUptime()
		uptimeStr := template.FormatUptime(uptime)

		tmplCtx.Server = &template.ServerContext{
			ClientCount:     count,
			Clients:         clients,
			Uptime:          uptime,
			UptimeFormatted: uptimeStr,
		}

		// Root-level convenience
		tmplCtx.ClientCount = count
		tmplCtx.Clients = clients
		tmplCtx.ServerUptime = uptime
		tmplCtx.ServerUptimeStr = uptimeStr
	}

	// Populate KV context
	d.refreshKVSnapshot(tmplCtx)
}

// refreshKVSnapshot updates the KV snapshot in the template context from the global store.
func (d *Dispatcher) refreshKVSnapshot(ctx *template.TemplateContext) {
	if d.kvManager != nil {
		ctx.KV = d.kvManager.ListKV()
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
	// 1. Handle Delay
	if a.Delay != "" {
		dur, err := time.ParseDuration(a.Delay)
		if err == nil {
			if d.verbose {
				d.errorf("  [handler] waiting %v (delay)...\n", dur)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(dur):
				// Continue
			}
		} else {
			d.errorf("  [handler] warning: invalid delay %q: %v\n", a.Delay, err)
		}
	}

	// 2. Execute Primary Action
	var err error
	switch strings.ToLower(a.Type) {
	case "shell":
		err = d.executeShell(ctx, a, tmplCtx)
	case "send":
		err = d.executeSend(a, tmplCtx)
	case "log":
		err = d.executeLog(a, tmplCtx)
	case "builtin":
		err = d.executeBuiltin(ctx, a, tmplCtx)
	default:
		return fmt.Errorf("unknown action type: %s", a.Type)
	}

	if err != nil {
		if err == ErrDrop {
			return err
		}
		return err
	}

	// 3. Handle Respond (transformation override or follow-up)
	if a.Respond != "" {
		respAction := &Action{Type: "send", Message: a.Respond}
		return d.executeSend(respAction, tmplCtx)
	}

	return nil
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
	} else if err != nil {
		// System-level execution failure (e.g. command not found)
		d.errorf("  [handler] error: command execution failed: %v\n", err)
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

func (d *Dispatcher) executeLog(a *Action, tmplCtx *template.TemplateContext) error {
	// Delegate to the new LogBuiltin for consistent JSONL formatting and features
	h, ok := GetBuiltin("log")
	if !ok {
		// Fallback to simple stderr if builtin is somehow missing
		msgStr, _ := d.templateEngine.Execute("log", a.Message, tmplCtx)
		d.errorf("  [handler] log: %s\n", msgStr)
		return nil
	}

	return h.Execute(context.Background(), d, a, tmplCtx)
}

func (d *Dispatcher) executeBuiltin(ctx context.Context, a *Action, tmplCtx *template.TemplateContext) error {
	cmdStr, err := d.templateEngine.Execute("builtin", a.Command, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in builtin command: %w", err)
	}

	if d.verbose {
		d.errorf("  [handler] builtin command requested: %s\n", cmdStr)
	}

	h, ok := GetBuiltin(cmdStr)
	if !ok {
		return fmt.Errorf("unknown builtin: %s", cmdStr)
	}

	return h.Execute(ctx, d, a, tmplCtx)
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
	// Clean up registry resources for this connection
	if d.registry != nil && d.conn != nil {
		d.registry.ClearConnResources(d.conn.GetID())
	}

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

// executeHandlerError runs actions defined in OnError or OnErrorMsg for a specific handler.
func (d *Dispatcher) executeHandlerError(ctx context.Context, h *Handler, tmplCtx *template.TemplateContext, err error) {
	if h.OnErrorMsg == "" && len(h.OnError) == 0 {
		return
	}

	d.log("  [handler] error hook for %q: %v\n", h.Name, err)

	// Ensure error is in context
	if tmplCtx.Error == "" {
		tmplCtx.Error = err.Error()
	}

	// 1. Run OnErrorMsg if present
	if h.OnErrorMsg != "" {
		action := Action{Type: "send", Message: h.OnErrorMsg}
		if err := d.ExecuteAction(ctx, &action, tmplCtx, nil); err != nil {
			d.errorf("  [handler] error matching OnErrorMsg for %q: %v\n", h.Name, err)
		}
	}

	// 2. Run OnError actions
	for _, a := range h.OnError {
		if err := d.ExecuteAction(ctx, &a, tmplCtx, nil); err != nil {
			d.errorf("  [handler] error in OnError action for %q: %v\n", h.Name, err)
		}
	}
}

// connToMessage reconstructs a ws.Message from the template context.
func (d *Dispatcher) connToMessage(ctx *template.TemplateContext) *ws.Message {
	mt := ws.TextMessage
	switch ctx.MessageType {
	case "binary":
		mt = ws.BinaryMessage
	case "ping":
		mt = ws.PingMessage
	case "pong":
		mt = ws.PongMessage
	}

	return &ws.Message{
		Type: mt,
		Data: ctx.MessageBytes,
		Metadata: ws.MessageMetadata{
			Timestamp:    ctx.Timestamp,
			Direction:    ctx.Direction,
			MessageIndex: ctx.MessageIndex,
		},
	}
}

// silentConn wraps a Connection and discards all writes and closures.
type silentConn struct {
	Connection
}

func (s *silentConn) Write(msg *ws.Message) error {
	return nil // Discard
}

func (s *silentConn) CloseWithCode(code int, reason string) error {
	return nil // Discard
}

func (d *Dispatcher) cloneWithConn(conn Connection) *Dispatcher {
	return &Dispatcher{
		registry:         d.registry,
		conn:             conn,
		templateEngine:   d.templateEngine,
		verbose:          d.verbose,
		Log:              d.Log,
		Error:            d.Error,
		variables:        d.variables,
		sessionVariables: d.sessionVariables,
		systemEnv:        d.systemEnv,
		sandbox:          d.sandbox,
		allowlist:        d.allowlist,
		_handlerHits:     d._handlerHits,
		_activeHandlers:  d._activeHandlers,
		serverStats:      d.serverStats,
		topicManager:     d.topicManager,
		kvManager:        d.kvManager,
		redisManager:     d.redisManager,
	}
}
