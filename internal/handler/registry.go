package handler

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/itchyny/gojq"
	"github.com/spf13/cast"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/time/rate"
)

// Registry manages a collection of message handlers.
type Registry struct {
	handlers   []Handler
	schemas    map[string]*gojsonschema.Schema
	handlerMu  map[string]*sync.Mutex
	limiters   map[string]*rate.Limiter
	debouncers map[string]*debouncer
	mu         sync.RWMutex
}

type debouncer struct {
	mu      sync.Mutex
	timer   *time.Timer
	pending *ws.Message
}

// NewRegistry creates a new handler registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers:  make([]Handler, 0),
		schemas:   make(map[string]*gojsonschema.Schema),
		handlerMu: make(map[string]*sync.Mutex),
		limiters:  make(map[string]*rate.Limiter),
		debouncers: make(map[string]*debouncer),
	}
}

// Add adds a single handler to the registry, subject to name uniqueness.
func (r *Registry) Add(h Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.handlers {
		if existing.Name == h.Name {
			return fmt.Errorf("handler %q already exists", h.Name)
		}
	}

	r.handlers = append(r.handlers, h)
	r.sort()
	return nil
}

// GetHandler returns a copy of a handler by name.
func (r *Registry) GetHandler(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.handlers {
		if h.Name == name {
			return h, true
		}
	}
	return Handler{}, false
}

// UpdateHandler replaces an existing handler with the same name.
func (r *Registry) UpdateHandler(h Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i, existing := range r.handlers {
		if existing.Name == h.Name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("handler %q not found", h.Name)
	}

	// Clean up resources for the old version
	delete(r.handlerMu, h.Name)
	delete(r.limiters, h.Name)
	if d, ok := r.debouncers[h.Name]; ok {
		d.mu.Lock()
		if d.timer != nil {
			d.timer.Stop()
		}
		d.mu.Unlock()
		delete(r.debouncers, h.Name)
	}

	r.handlers[index] = h
	r.sort()
	return nil
}

// AddHandlers adds multiple handlers to the registry and sorts them by priority.
func (r *Registry) AddHandlers(handlers []Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, handlers...)
	r.sort()
}

// ReplaceHandlers replaces all handlers in the registry and cleans up resources.
func (r *Registry) ReplaceHandlers(handlers []Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up all resources
	r.handlerMu = make(map[string]*sync.Mutex)
	r.limiters = make(map[string]*rate.Limiter)
	for _, d := range r.debouncers {
		d.mu.Lock()
		if d.timer != nil {
			d.timer.Stop()
		}
		d.mu.Unlock()
	}
	r.debouncers = make(map[string]*debouncer)
	r.schemas = make(map[string]*gojsonschema.Schema)

	r.handlers = handlers
	r.sort()
}

// Delete removes a handler by name and cleans up associated resources.
func (r *Registry) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, h := range r.handlers {
		if h.Name == name {
			r.handlers = append(r.handlers[:i], r.handlers[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("handler %q not found", name)
	}

	// Clean up associated resources
	delete(r.handlerMu, name)
	delete(r.limiters, name)

	if d, ok := r.debouncers[name]; ok {
		d.mu.Lock()
		if d.timer != nil {
			d.timer.Stop()
		}
		d.mu.Unlock()
		delete(r.debouncers, name)
	}

	return nil
}

// Sort orders handlers by priority (descending) and then original order.
func (r *Registry) sort() {
	sort.SliceStable(r.handlers, func(i, j int) bool {
		return r.handlers[i].Priority > r.handlers[j].Priority
	})
}

// Match returns all handlers that match the given message, in priority order.
func (r *Registry) Match(msg *ws.Message, engine *template.Engine, ctx *template.TemplateContext) ([]*Handler, error) {
	var matches []*Handler

	for i := range r.handlers {
		h := &r.handlers[i]
		matched, err := r.matchHandler(h, msg, engine, ctx)
		if err != nil {
			return nil, fmt.Errorf("matching handler %q: %w", h.Name, err)
		}
		if matched {
			matches = append(matches, h)
			// Short-circuit: if the handler is exclusive, stop matching further handlers.
			if h.Exclusive {
				break
			}
		}
	}

	return matches, nil
}

func (r *Registry) matchHandler(h *Handler, msg *ws.Message, engine *template.Engine, ctx *template.TemplateContext) (bool, error) {
	if len(h.Variables) > 0 && engine != nil && ctx != nil {
		// Clone Vars to avoid polluting other handlers
		originalVars := ctx.Vars
		ctx.Vars = make(map[string]interface{}, len(originalVars)+len(h.Variables))
		for k, v := range originalVars {
			ctx.Vars[k] = v
		}

		// Evaluate handler variables
		evaluated := evaluateVariables(engine, h.Variables, ctx, false, nil)
		for k, v := range evaluated {
			ctx.Vars[k] = v
		}

		// Ensure we restore original vars after matching
		defer func() {
			ctx.Vars = originalVars
		}()
	}
	return r.matchMatcher(&h.Match, h.BaseDir, msg, engine, ctx)
}

func (r *Registry) matchMatcher(m *Matcher, baseDir string, msg *ws.Message, engine *template.Engine, ctx *template.TemplateContext) (bool, error) {
	// 1. Handle Composite Matchers
	if len(m.All) > 0 {
		for _, sub := range m.All {
			matched, err := r.matchMatcher(&sub, baseDir, msg, engine, ctx)
			if err != nil || !matched {
				return matched, err
			}
		}
		// If all matched, we still need to check if OTHER fields in this matcher (like binary or regex) also match.
	}

	if len(m.Any) > 0 {
		anyMatched := false
		for _, sub := range m.Any {
			matched, err := r.matchMatcher(&sub, baseDir, msg, engine, ctx)
			if err != nil {
				return false, err
			}
			if matched {
				anyMatched = true
				break
			}
		}
		if !anyMatched {
			return false, nil
		}
		// If one matched, we still need to check if OTHER fields in this matcher also match.
	}

	// 2. Handle Binary filter
	if m.Binary != nil {
		isBinary := msg.Type == ws.BinaryMessage
		if *m.Binary != isBinary {
			return false, nil
		}
	}

	// 3. Check if ANY specific pattern match condition is present.
	hasPatternMatch := m.Pattern != "" || m.Regex != "" || m.JQ != "" ||
		m.JSONPath != "" || m.JSONSchema != "" || m.Template != ""

	if !hasPatternMatch {
		// If nothing else to check, it's a match if we got this far.
		// Note: Validation ensures that if a handler has actions, it MUST have at least one match condition.
		return true, nil
	}

	msgStr := string(msg.Data)
	// Trim whitespace for more resilient matching in interactive sessions (e.g. echo servers with newlines)
	trimmedMsg := strings.TrimSpace(msgStr)

	// Support regex shorthand: match.regex: "pattern"
	if m.Regex != "" {
		matched, err := regexp.MatchString(m.Regex, trimmedMsg)
		if err != nil {
			return false, fmt.Errorf("regex shorthand error: %w", err)
		}
		return matched, nil
	}

	// Support jq shorthand: match.jq: "query"
	if m.JQ != "" {
		return r.matchJSON(m.JQ, trimmedMsg)
	}

	// Support json_path + equals: match.json_path: "path", match.equals: value
	if m.JSONPath != "" {
		return r.matchJSONPath(m.JSONPath, m.Equals, trimmedMsg)
	}

	// Support json_schema: match.json_schema: "path/to/schema.json"
	if m.JSONSchema != "" {
		return r.matchJSONSchema(m.JSONSchema, baseDir, trimmedMsg)
	}

	// Support template: match.template: "{{ eq .msg.data.type 'alert' }}"
	if m.Template != "" {
		return r.matchTemplate(engine, ctx, m.Template)
	}

	if m.Pattern == "" {
		// Should not happen if hasPatternMatch is true, but for safety:
		return true, nil
	}

	switch strings.ToLower(m.Type) {
	case "text", "":
		if strings.ContainsAny(m.Pattern, "*?") {
			return r.matchGlob(m.Pattern, trimmedMsg)
		}
		// Trim whitespace for more resilient matching in interactive sessions
		return strings.TrimSpace(msgStr) == m.Pattern, nil
	case "regex":
		matched, err := regexp.MatchString(m.Pattern, trimmedMsg)
		if err != nil {
			return false, fmt.Errorf("regex error: %w", err)
		}
		return matched, nil
	case "glob":
		return r.matchGlob(m.Pattern, trimmedMsg)
	case "json", "jq":
		return r.matchJSON(m.Pattern, trimmedMsg)
	case "json_schema":
		return r.matchJSONSchema(m.Pattern, baseDir, trimmedMsg)
	case "template":
		return r.matchTemplate(engine, ctx, m.Pattern)
	default:
		return false, fmt.Errorf("unknown matcher type: %s", m.Type)
	}
}

func (r *Registry) matchGlob(pattern, msg string) (bool, error) {
	// Convert glob pattern to regex
	// 1. Quote all regex metacharacters
	regexStr := regexp.QuoteMeta(pattern)
	
	// 2. Unescape and convert glob wildcards
	// QuoteMeta escapes '*' as '\*' and '?' as '\?'
	regexStr = strings.ReplaceAll(regexStr, "\\*", ".*")
	regexStr = strings.ReplaceAll(regexStr, "\\?", ".")

	// 3. Anchor and enable single-line mode (so '.' matches newlines)
	regexStr = "^(?s:" + regexStr + ")$"

	matched, err := regexp.MatchString(regexStr, msg)
	if err != nil {
		return false, fmt.Errorf("glob to regex error: %w", err)
	}
	return matched, nil
}

func (r *Registry) matchJSON(query, msg string) (bool, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(msg), &data); err != nil {
		return false, nil // Not JSON, so doesn't match JSON query
	}

	q, err := gojq.Parse(query)
	if err != nil {
		return false, fmt.Errorf("parsing gojq query: %w", err)
	}

	iter := q.Run(data)
	v, ok := iter.Next()
	if !ok {
		return false, nil
	}
	if _, ok := v.(error); ok {
		return false, nil // Evaluation failed (e.g. applied to null), treat as no match
	}

	// Match if the result is truthy (not false, not nil, not empty string, etc.)
	switch val := v.(type) {
	case bool:
		return val, nil
	case nil:
		return false, nil
	case string:
		return val != "", nil
	case float64:
		return val != 0, nil
	case int:
		return val != 0, nil
	default:
		return true, nil
	}
}

func (r *Registry) matchJSONPath(jsonPath string, equals interface{}, msg string) (bool, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(msg), &data); err != nil {
		// If unmarshalling fails, treat the whole message as a raw string
		// This allows matching on root values ($) even for non-JSON payloads.
		data = msg
	}

	// Prepare jq query from JSONPath
	query := jsonPath
	if strings.HasPrefix(query, "$.") {
		query = "." + strings.TrimPrefix(query, "$.")
	} else if strings.HasPrefix(query, "$") {
		query = "." + strings.TrimPrefix(query, "$")
	} else if !strings.HasPrefix(query, ".") {
		query = "." + query
	}

	q, err := gojq.Parse(query)
	if err != nil {
		return false, fmt.Errorf("parsing json_path query %q: %w", query, err)
	}

	iter := q.Run(data)
	v, ok := iter.Next()
	if !ok {
		return false, nil // Path does not exist
	}
	if _, ok := v.(error); ok {
		return false, nil // Path evaluation failed (e.g. key missing in path)
	}

	// Compare extracted value with the expected value
	if equals == nil {
		return v == nil, nil
	}

	return r.compareValues(v, equals), nil
}

func (r *Registry) compareValues(actual, expected interface{}) bool {
	// Use cast for flexible type-safe comparison
	switch e := expected.(type) {
	case string:
		return cast.ToString(actual) == e
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// Convert both to float64 for numeric comparison to handle precision
		return cast.ToFloat64(actual) == cast.ToFloat64(expected)
	case float32, float64:
		return cast.ToFloat64(actual) == cast.ToFloat64(expected)
	case bool:
		return cast.ToBool(actual) == e
	default:
		// Fallback to string comparison for objects/slices (though plan says primitives)
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	}
}

func (r *Registry) matchJSONSchema(schemaPath, baseDir, msg string) (bool, error) {
	// Resolve schema path
	fullPath := schemaPath
	if !filepath.IsAbs(fullPath) && baseDir != "" {
		fullPath = filepath.Join(baseDir, schemaPath)
	}
	
	// Ensure absolute path for canonical file:// URI
	if !filepath.IsAbs(fullPath) {
		if abs, err := filepath.Abs(fullPath); err == nil {
			fullPath = abs
		}
	}

	// Check cache
	r.mu.RLock()
	schema, ok := r.schemas[fullPath]
	r.mu.RUnlock()

	if !ok {
		// Compile and cache
		r.mu.Lock()
		// Double check after lock
		schema, ok = r.schemas[fullPath]
		if !ok {
			// Load from file
			schemaLoader := gojsonschema.NewReferenceLoader("file://" + fullPath)
			var err error
			schema, err = gojsonschema.NewSchema(schemaLoader)
			if err != nil {
				r.mu.Unlock()
				return false, fmt.Errorf("loading JSON schema from %s: %w", fullPath, err)
			}
			r.schemas[fullPath] = schema
		}
		r.mu.Unlock()
	}

	// Validate message
	documentLoader := gojsonschema.NewStringLoader(msg)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		// If it's not valid JSON at all, it fails validation
		return false, nil
	}

	return result.Valid(), nil
}

func (r *Registry) matchTemplate(engine *template.Engine, ctx *template.TemplateContext, tmpl string) (bool, error) {
	if engine == nil || ctx == nil {
		return false, nil
	}

	result, err := engine.Execute("match", tmpl, ctx)
	if err != nil {
		return false, fmt.Errorf("template match error: %w", err)
	}

	// Truthiness logic:
	// - empty string is false
	// - "false" is false (case insensitive)
	// - "0" is false
	// - "null" is false
	trimmed := strings.ToLower(strings.TrimSpace(result))
	if trimmed == "" || trimmed == "false" || trimmed == "0" || trimmed == "null" {
		return false, nil
	}

	return true, nil
}

// Handlers returns the list of registered handlers in their current order.
func (r *Registry) Handlers() []Handler {
	return r.handlers
}

// LifecycleHandlers returns handlers that have actions for specific lifecycle events.
func (r *Registry) LifecycleHandlers() (onConnect, onDisconnect, onError []*Handler) {
	for i := range r.handlers {
		h := &r.handlers[i]
		if len(h.OnConnect) > 0 {
			onConnect = append(onConnect, h)
		}
		if len(h.OnDisconnect) > 0 {
			onDisconnect = append(onDisconnect, h)
		}
		if len(h.OnError) > 0 {
			onError = append(onError, h)
		}
	}
	return
}

// GetHandlerMu returns the mutex for a specific handler, creating it if necessary.
func (r *Registry) GetHandlerMu(name string) *sync.Mutex {
	r.mu.RLock()
	mu, ok := r.handlerMu[name]
	r.mu.RUnlock()
	if ok {
		return mu
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Double check after acquiring write lock
	if mu, ok := r.handlerMu[name]; ok {
		return mu
	}
	mu = &sync.Mutex{}
	r.handlerMu[name] = mu
	return mu
}

// GetLimiter returns the rate limiter for a specific handler, creating it if necessary.
func (r *Registry) GetLimiter(name, rateStr string) *rate.Limiter {
	if rateStr == "" {
		return nil
	}

	r.mu.RLock()
	limiter, ok := r.limiters[name]
	r.mu.RUnlock()
	if ok {
		return limiter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check after acquiring write lock
	if limiter, ok = r.limiters[name]; ok {
		return limiter
	}

	// Parse and create limiter
	perSec, burst, err := ParseRateLimit(rateStr)
	if err != nil {
		// Should have been validated already, but for safety:
		return nil
	}

	limiter = rate.NewLimiter(rate.Limit(perSec), burst)
	r.limiters[name] = limiter
	return limiter
}

// Debounce handles trailing-edge debouncing for a specific handler.
// It resets the timer on each call and executes the callback with the most recent message.
func (r *Registry) Debounce(name string, duration time.Duration, msg *ws.Message, execute func(*ws.Message)) {
	r.mu.Lock()
	d, ok := r.debouncers[name]
	if !ok {
		d = &debouncer{}
		r.debouncers[name] = d
	}
	r.mu.Unlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.pending = msg
	d.timer = time.AfterFunc(duration, func() {
		d.mu.Lock()
		m := d.pending
		d.timer = nil
		d.pending = nil
		d.mu.Unlock()

		if m != nil {
			execute(m)
		}
	})
}
