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

// RegistryMode defines the operational mode of the handler registry.
type RegistryMode string

const (
	// ClientMode is used when xwebs is running as a client (connect).
	ClientMode RegistryMode = "client"
	// ServerMode is used when xwebs is running as a server (serve).
	ServerMode RegistryMode = "server"
)

// HandlerStats tracks execution statistics for a handler.
type HandlerStats struct {
	MatchCount   uint64
	TotalLatency time.Duration
	ErrorCount   uint64
	mu           sync.RWMutex
}

// SlowLogEntry records a single slow handler execution.
type SlowLogEntry struct {
	HandlerName string
	Duration    time.Duration
	Error       string
	Timestamp   time.Time
}

// MatchResult contains a matching handler and its captured pattern matches.
type MatchResult struct {
	Handler *Handler
	Matches []string
}

// RegistryStats tracks global execution statistics.
type RegistryStats struct {
	TotalExecutions uint64
	TotalErrors     uint64
	SlowLog         []SlowLogEntry
	maxSlowLog      int
	mu              sync.RWMutex
}

// Registry manages a collection of message handlers.
type Registry struct {
	handlers              []Handler
	schemas               map[string]*gojsonschema.Schema
	handlerMu             map[string]*sync.Mutex
	limiters              map[string]*rate.Limiter
	debouncers            map[string]*debouncer
	stats                 map[string]*HandlerStats
	global                RegistryStats
	disabled              map[string]string         // handlerName -> reason (empty if enabled, non-empty if disabled)
	sequenceIndices       map[string]int            // handlerName -> index
	sequenceClientIndices map[string]map[string]int // handlerName -> connID -> index
	scopedLimiters        map[string]*rate.Limiter  // key -> limiter
	scopedLimiterRates    map[string]string         // key -> rateStr
	scopedLimiterBursts   map[string]int            // key -> burst
	luaPools              map[string]interface{}    // handlerName -> *LuaPool (interface{} to avoid circular dependency or import lua here)
	luaStates             map[string]interface{}    // handlerName -> *lua.LTable
	throttleTimestamps    map[string]time.Time      // handlerName:connID -> last broadcast time
	roundRobinIndices     map[string]int            // actionKey -> index
	sampleIndices         map[string]int            // actionKey -> count
	Mode                  RegistryMode
	mu                    sync.RWMutex
}

type debouncer struct {
	mu      sync.Mutex
	timer   *time.Timer
	pending *ws.Message
}

// NewRegistry creates a new handler registry with the given mode.
func NewRegistry(mode RegistryMode) *Registry {
	return &Registry{
		handlers:              make([]Handler, 0),
		schemas:               make(map[string]*gojsonschema.Schema),
		handlerMu:             make(map[string]*sync.Mutex),
		limiters:              make(map[string]*rate.Limiter),
		debouncers:            make(map[string]*debouncer),
		stats:                 make(map[string]*HandlerStats),
		disabled:              make(map[string]string),
		sequenceIndices:       make(map[string]int),
		sequenceClientIndices: make(map[string]map[string]int),
		scopedLimiters:        make(map[string]*rate.Limiter),
		scopedLimiterRates:    make(map[string]string),
		scopedLimiterBursts:   make(map[string]int),
		luaPools:              make(map[string]interface{}),
		luaStates:             make(map[string]interface{}),
		throttleTimestamps:    make(map[string]time.Time),
		roundRobinIndices:     make(map[string]int),
		sampleIndices:         make(map[string]int),
		Mode:                  mode,
		global: RegistryStats{
			maxSlowLog: 50,
		},
	}
}

// Add adds a single handler to the registry, subject to name uniqueness.
func (r *Registry) Add(h Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate the handler against the current registry mode
	cfg := Config{Handlers: []Handler{h}}
	if err := cfg.Validate(r.Mode); err != nil {
		return err
	}

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

func (r *Registry) GetHandlerBaseDir(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range r.handlers {
		if h.Name == name {
			return h.BaseDir
		}
	}
	return ""
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

	// Validate the handler against the current registry mode
	cfg := Config{Handlers: []Handler{h}}
	if err := cfg.Validate(r.Mode); err != nil {
		return err
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

	// Clean up scoped limiters for this handler
	for k := range r.scopedLimiters {
		if strings.HasPrefix(k, "handler:"+h.Name+":") || strings.HasPrefix(k, "client:"+h.Name+":") {
			delete(r.scopedLimiters, k)
			delete(r.scopedLimiterRates, k)
			delete(r.scopedLimiterBursts, k)
		}
	}
	// Clean up throttle timestamps for this handler
	for k := range r.throttleTimestamps {
		if strings.HasPrefix(k, h.Name+":") {
			delete(r.throttleTimestamps, k)
		}
	}
	// Clean up round-robin indices for this handler
	for k := range r.roundRobinIndices {
		if strings.HasPrefix(k, h.Name+":") {
			delete(r.roundRobinIndices, k)
		}
	}
	// Clean up sample indices for this handler
	for k := range r.sampleIndices {
		if strings.HasPrefix(k, h.Name+":") {
			delete(r.sampleIndices, k)
		}
	}

	r.handlers[index] = h
	r.sort()
	return nil
}

// AddHandlers adds multiple handlers to the registry and sorts them by priority.
func (r *Registry) AddHandlers(handlers []Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate all handlers first
	cfg := Config{Handlers: handlers}
	if err := cfg.Validate(r.Mode); err != nil {
		return err
	}

	r.handlers = append(r.handlers, handlers...)
	r.sort()
	return nil
}

// ReplaceHandlers replaces all handlers in the registry and cleans up resources.
func (r *Registry) ReplaceHandlers(handlers []Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate all handlers first
	cfg := Config{Handlers: handlers}
	if err := cfg.Validate(r.Mode); err != nil {
		return err
	}

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
	r.stats = make(map[string]*HandlerStats)
	r.disabled = make(map[string]string)
	r.sequenceIndices = make(map[string]int)
	r.sequenceClientIndices = make(map[string]map[string]int)
	r.scopedLimiters = make(map[string]*rate.Limiter)
	r.scopedLimiterRates = make(map[string]string)
	r.scopedLimiterBursts = make(map[string]int)
	r.luaPools = make(map[string]interface{})
	r.luaStates = make(map[string]interface{})
	r.throttleTimestamps = make(map[string]time.Time)
	r.roundRobinIndices = make(map[string]int)
	r.sampleIndices = make(map[string]int)

	r.handlers = handlers
	r.sort()
	return nil
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
	delete(r.sequenceIndices, name)
	delete(r.sequenceClientIndices, name)
	delete(r.luaPools, name)
	delete(r.luaStates, name)
	delete(r.disabled, name)

	// Clean up throttle timestamps for this handler
	for k := range r.throttleTimestamps {
		if strings.HasPrefix(k, name+":") {
			delete(r.throttleTimestamps, k)
		}
	}
	for k := range r.roundRobinIndices {
		if strings.HasPrefix(k, name+":") {
			delete(r.roundRobinIndices, k)
		}
	}
	// Clean up sample indices
	for k := range r.sampleIndices {
		if strings.HasPrefix(k, name+":") {
			delete(r.sampleIndices, k)
		}
	}

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

// RenameHandler renames an existing handler and migrates its associated resources.
func (r *Registry) RenameHandler(oldName, newName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if oldName == newName {
		return nil
	}

	index := -1
	for i, h := range r.handlers {
		if h.Name == oldName {
			index = i
		}
		if h.Name == newName {
			return fmt.Errorf("handler %q already exists", newName)
		}
	}

	if index == -1 {
		return fmt.Errorf("handler %q not found", oldName)
	}

	// Update name in slices
	r.handlers[index].Name = newName

	// Migrate resources
	if mu, ok := r.handlerMu[oldName]; ok {
		r.handlerMu[newName] = mu
		delete(r.handlerMu, oldName)
	}
	if lim, ok := r.limiters[oldName]; ok {
		r.limiters[newName] = lim
		delete(r.limiters, oldName)
	}
	if deb, ok := r.debouncers[oldName]; ok {
		r.debouncers[newName] = deb
		delete(r.debouncers, oldName)
	}
	if stats, ok := r.stats[oldName]; ok {
		r.stats[newName] = stats
		delete(r.stats, oldName)
	}
	if dis, ok := r.disabled[oldName]; ok {
		r.disabled[newName] = dis
		delete(r.disabled, oldName)
	}
	if idx, ok := r.sequenceIndices[oldName]; ok {
		r.sequenceIndices[newName] = idx
		delete(r.sequenceIndices, oldName)
	}
	if cIdx, ok := r.sequenceClientIndices[oldName]; ok {
		r.sequenceClientIndices[newName] = cIdx
		delete(r.sequenceClientIndices, oldName)
	}

	// Migrate throttle timestamps
	for k, v := range r.throttleTimestamps {
		if strings.HasPrefix(k, oldName+":") {
			newKey := newName + ":" + strings.TrimPrefix(k, oldName+":")
			r.throttleTimestamps[newKey] = v
			delete(r.throttleTimestamps, k)
		}
	}
	// Migrate round-robin indices
	for k, v := range r.roundRobinIndices {
		if strings.HasPrefix(k, oldName+":") {
			newKey := newName + ":" + strings.TrimPrefix(k, oldName+":")
			r.roundRobinIndices[newKey] = v
			delete(r.roundRobinIndices, k)
		}
	}
	// Migrate sample indices
	for k, v := range r.sampleIndices {
		if strings.HasPrefix(k, oldName+":") {
			newKey := newName + ":" + strings.TrimPrefix(k, oldName+":")
			r.sampleIndices[newKey] = v
			delete(r.sampleIndices, k)
		}
	}

	return nil
}

// Sort orders handlers by priority (descending) and then original order.
func (r *Registry) sort() {
	sort.SliceStable(r.handlers, func(i, j int) bool {
		return r.handlers[i].Priority > r.handlers[j].Priority
	})
}

// Match returns all handlers that match the given message, along with their capture groups, in priority order.
func (r *Registry) Match(msg *ws.Message, engine *template.Engine, ctx *template.TemplateContext) ([]MatchResult, error) {
	var results []MatchResult

	r.mu.RLock()
	handlers := r.handlers
	disabled := make(map[string]string)
	for k, v := range r.disabled {
		disabled[k] = v
	}
	r.mu.RUnlock()

	for i := range handlers {
		h := &handlers[i]

		// Skip disabled handlers
		if disabled[h.Name] != "" {
			continue
		}

		matched, matches, err := r.matchHandler(h, msg, engine, ctx)
		if err != nil {
			return nil, fmt.Errorf("matching handler %q: %w", h.Name, err)
		}
		if matched {
			results = append(results, MatchResult{
				Handler: h,
				Matches: matches,
			})

			// Record match hit
			r.RecordMatch(h.Name)

			// Short-circuit: if the handler is exclusive, stop matching further handlers.
			if h.Exclusive {
				break
			}
		}
	}
	return results, nil
}

func (r *Registry) matchHandler(h *Handler, msg *ws.Message, engine *template.Engine, ctx *template.TemplateContext) (bool, []string, error) {
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

func (r *Registry) matchMatcher(m *Matcher, baseDir string, msg *ws.Message, engine *template.Engine, ctx *template.TemplateContext) (bool, []string, error) {
	// 1. Handle Composite Matchers (All)
	var compositeMatches []string
	if len(m.All) > 0 {
		for _, sub := range m.All {
			matched, matches, err := r.matchMatcher(&sub, baseDir, msg, engine, ctx)
			if err != nil || !matched {
				return matched, nil, err
			}
			if len(matches) > 0 && len(compositeMatches) == 0 {
				compositeMatches = matches
			}
		}
	}

	// 1b. Handle Composite Matchers (Any)
	if len(m.Any) > 0 {
		anyMatched := false
		for _, sub := range m.Any {
			matched, matches, err := r.matchMatcher(&sub, baseDir, msg, engine, ctx)
			if err != nil {
				return false, nil, err
			}
			if matched {
				anyMatched = true
				if len(matches) > 0 && len(compositeMatches) == 0 {
					compositeMatches = matches
				}
				break
			}
		}
		if !anyMatched {
			return false, nil, nil
		}
	}

	// 2. Handle Binary filter
	if m.Binary != nil {
		isBinary := msg.Type == ws.BinaryMessage
		if *m.Binary != isBinary {
			return false, nil, nil
		}
	}

	// 3. Check if ANY specific pattern match condition is present.
	hasPatternMatch := m.Pattern != "" || m.Regex != "" || m.JQ != "" ||
		m.JSONPath != "" || m.JSONSchema != "" || m.Template != ""

	if !hasPatternMatch {
		return true, compositeMatches, nil
	}

	msgStr := string(msg.Data)
	trimmedMsg := strings.TrimSpace(msgStr)

	// Support regex shorthand
	if m.Regex != "" {
		re, err := regexp.Compile(m.Regex)
		if err != nil {
			return false, nil, fmt.Errorf("regex shorthand error: %w", err)
		}
		matches := re.FindStringSubmatch(trimmedMsg)
		if matches == nil {
			return false, nil, nil
		}
		// Return submatches including the full match at index 0
		return true, matches, nil
	}

	// Support jq shorthand
	if m.JQ != "" {
		matched, err := r.matchJSON(m.JQ, trimmedMsg)
		if !matched || err != nil {
			return matched, nil, err
		}
		return true, compositeMatches, nil
	}

	// Support json_path + equals
	if m.JSONPath != "" {
		matched, err := r.matchJSONPath(m.JSONPath, m.Equals, trimmedMsg)
		if !matched || err != nil {
			return matched, nil, err
		}
		return true, compositeMatches, nil
	}

	// Support json_schema
	if m.JSONSchema != "" {
		matched, err := r.matchJSONSchema(m.JSONSchema, baseDir, trimmedMsg)
		if !matched || err != nil {
			return matched, nil, err
		}
		return true, compositeMatches, nil
	}

	// Support template
	if m.Template != "" {
		matched, err := r.matchTemplate(engine, ctx, m.Template)
		if !matched || err != nil {
			return matched, nil, err
		}
		return true, compositeMatches, nil
	}

	if m.Pattern == "" {
		return true, nil, nil
	}

	switch strings.ToLower(m.Type) {
	case "text", "":
		if strings.ContainsAny(m.Pattern, "*?") {
			return r.matchGlob(m.Pattern, trimmedMsg)
		}
		return strings.TrimSpace(msgStr) == m.Pattern, nil, nil
	case "regex":
		re, err := regexp.Compile(m.Pattern)
		if err != nil {
			return false, nil, fmt.Errorf("regex error: %w", err)
		}
		matches := re.FindStringSubmatch(trimmedMsg)
		if matches == nil {
			return false, nil, nil
		}
		if len(matches) > 1 {
			return true, matches[1:], nil
		}
		return true, nil, nil
	case "glob":
		return r.matchGlob(m.Pattern, trimmedMsg)
	case "json", "jq":
		matched, err := r.matchJSON(m.Pattern, trimmedMsg)
		return matched, nil, err
	case "json_schema":
		matched, err := r.matchJSONSchema(m.Pattern, baseDir, trimmedMsg)
		return matched, nil, err
	case "template":
		matched, err := r.matchTemplate(engine, ctx, m.Pattern)
		return matched, nil, err
	default:
		return false, nil, fmt.Errorf("unknown matcher type: %s", m.Type)
	}
}

func (r *Registry) matchGlob(pattern, msg string) (bool, []string, error) {
	// Convert glob pattern to regex by wrapping wildcards in capture groups
	// 1. Quote all regex metacharacters
	regexStr := regexp.QuoteMeta(pattern)

	// 2. Unescape and convert glob wildcards to capture groups: (.*) and (.)
	// QuoteMeta escapes '*' as '\*' and '?' as '\?'
	regexStr = strings.ReplaceAll(regexStr, "\\*", "(.*)")
	regexStr = strings.ReplaceAll(regexStr, "\\?", "(.)")

	// 3. Anchor and enable single-line mode
	regexStr = "^(?s:" + regexStr + ")$"

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return false, nil, fmt.Errorf("glob to regex error: %w", err)
	}

	matches := re.FindStringSubmatch(msg)
	if matches == nil {
		return false, nil, nil
	}

	if len(matches) > 1 {
		return true, matches[1:], nil
	}
	return true, nil, nil
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

// RecordMatch increments the match count for a handler.
func (r *Registry) RecordMatch(name string) {
	s := r.getOrCreateStats(name)
	s.mu.Lock()
	s.MatchCount++
	s.mu.Unlock()
}

// RecordExecution records the duration and error status of a handler execution.
func (r *Registry) RecordExecution(name string, duration time.Duration, err error) {
	s := r.getOrCreateStats(name)
	s.mu.Lock()
	s.TotalLatency += duration
	if err != nil {
		s.ErrorCount++
	}
	s.mu.Unlock()

	// Update global stats
	r.global.mu.Lock()
	defer r.global.mu.Unlock()

	r.global.TotalExecutions++
	if err != nil {
		r.global.TotalErrors++
	}

	// Update SlowLog
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	entry := SlowLogEntry{
		HandlerName: name,
		Duration:    duration,
		Error:       errStr,
		Timestamp:   time.Now(),
	}

	// Insert into SlowLog and keep sorted (slowest first)
	inserted := false
	for i, existing := range r.global.SlowLog {
		if duration > existing.Duration {
			// Insert at index i
			r.global.SlowLog = append(r.global.SlowLog[:i], append([]SlowLogEntry{entry}, r.global.SlowLog[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted && len(r.global.SlowLog) < r.global.maxSlowLog {
		r.global.SlowLog = append(r.global.SlowLog, entry)
	}

	// Trim if needed
	if len(r.global.SlowLog) > r.global.maxSlowLog {
		r.global.SlowLog = r.global.SlowLog[:r.global.maxSlowLog]
	}
}

// GetGlobalStats returns global execution statistics.
func (r *Registry) GetGlobalStats() (total uint64, errors uint64) {
	r.global.mu.RLock()
	defer r.global.mu.RUnlock()
	return r.global.TotalExecutions, r.global.TotalErrors
}

// GetSlowLog returns the slowest handler executions.
func (r *Registry) GetSlowLog(limit int) []SlowLogEntry {
	r.global.mu.RLock()
	defer r.global.mu.RUnlock()

	if limit <= 0 || limit > len(r.global.SlowLog) {
		limit = len(r.global.SlowLog)
	}

	res := make([]SlowLogEntry, limit)
	copy(res, r.global.SlowLog[:limit])
	return res
}

// getOrCreateStats returns the statistics for a handler, creating them if they don't exist.
func (r *Registry) getOrCreateStats(name string) *HandlerStats {
	r.mu.RLock()
	s, ok := r.stats[name]
	r.mu.RUnlock()

	if ok {
		return s
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check after acquiring write lock
	if s, ok = r.stats[name]; ok {
		return s
	}

	s = &HandlerStats{}
	r.stats[name] = s
	return s
}

// GetStats returns execution statistics for a handler.
func (r *Registry) GetStats(name string) (uint64, time.Duration, uint64, bool) {
	r.mu.RLock()
	s, ok := r.stats[name]
	if !ok {
		// Check if handler exists at all
		exists := false
		for _, h := range r.handlers {
			if h.Name == name {
				exists = true
				break
			}
		}
		r.mu.RUnlock()
		if exists {
			return 0, 0, 0, true
		}
		return 0, 0, 0, false
	}
	r.mu.RUnlock()

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.MatchCount, s.TotalLatency, s.ErrorCount, true
}

// EnableHandler enables a handler at runtime.
func (r *Registry) EnableHandler(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if handler exists
	found := false
	for _, h := range r.handlers {
		if h.Name == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("handler %q not found", name)
	}

	delete(r.disabled, name)
	return nil
}

// DisableHandlerWithReason disables a handler at runtime with a specific reason.
func (r *Registry) DisableHandlerWithReason(name, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if handler exists
	found := false
	for _, h := range r.handlers {
		if h.Name == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("handler %q not found", name)
	}

	if reason == "" {
		reason = "user"
	}
	r.disabled[name] = reason
	return nil
}

// DisableHandler disables a handler at runtime.
func (r *Registry) DisableHandler(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if handler exists
	found := false
	for _, h := range r.handlers {
		if h.Name == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("handler %q not found", name)
	}

	r.disabled[name] = "user"
	return nil
}

// GetDisabledReason returns the reason why a handler is disabled.
func (r *Registry) GetDisabledReason(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.disabled[name]
}

// IsDisabled returns true if the handler is currently disabled.
func (r *Registry) IsDisabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.disabled[name] != ""
}

// GetNextSequenceIndex returns the next index for a sequence builtin, potentially tracking it per-client.
func (r *Registry) GetNextSequenceIndex(handlerName, connID string, count int, loop, perClient bool) int {
	if count <= 0 {
		return 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var current int
	if perClient {
		if r.sequenceClientIndices[handlerName] == nil {
			r.sequenceClientIndices[handlerName] = make(map[string]int)
		}
		current = r.sequenceClientIndices[handlerName][connID]
	} else {
		current = r.sequenceIndices[handlerName]
	}

	idx := current % count
	next := current + 1

	// Handle loop behavior
	if !loop && next >= count {
		next = count - 1 // Stay on last item
	} else {
		next = next % count
	}

	if perClient {
		r.sequenceClientIndices[handlerName][connID] = next
	} else {
		r.sequenceIndices[handlerName] = next
	}

	return idx
}

// ResetSequence clears the sequence index for a specific handler.
func (r *Registry) ResetSequence(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sequenceIndices, name)
	delete(r.sequenceClientIndices, name)
}

// GetScopedLimiter returns or creates a limiter based on the given key and rate parameters.
func (r *Registry) GetScopedLimiter(key, rateStr string, burst int) *rate.Limiter {
	r.mu.RLock()
	limiter, ok := r.scopedLimiters[key]
	oldRate := r.scopedLimiterRates[key]
	oldBurst := r.scopedLimiterBursts[key]
	r.mu.RUnlock()

	if ok && oldRate == rateStr && oldBurst == burst {
		return limiter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check after lock
	if limiter, ok = r.scopedLimiters[key]; ok {
		if r.scopedLimiterRates[key] == rateStr && r.scopedLimiterBursts[key] == burst {
			return limiter
		}
	}

	// Parse and create/update
	perSec, burstVal, err := ParseRateLimit(rateStr)
	if err != nil {
		return nil
	}

	// Use provided burst if positive, otherwise use parsed burst
	if burst > 0 {
		burstVal = burst
	}

	if limiter != nil {
		// Update existing limiter
		limiter.SetLimit(rate.Limit(perSec))
		limiter.SetBurst(burstVal)
	} else {
		// Create new limiter
		limiter = rate.NewLimiter(rate.Limit(perSec), burstVal)
		r.scopedLimiters[key] = limiter
	}

	r.scopedLimiterRates[key] = rateStr
	r.scopedLimiterBursts[key] = burst
	return limiter
}

// GetLastThrottleBroadcast returns the last time a message was broadcasted to a client by a specific handler.
func (r *Registry) GetLastThrottleBroadcast(handlerName, connID string) time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.throttleTimestamps[handlerName+":"+connID]
}

// SetLastThrottleBroadcast updates the last broadcast time for a client by a specific handler.
func (r *Registry) SetLastThrottleBroadcast(handlerName, connID string, t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.throttleTimestamps[handlerName+":"+connID] = t
}

// GetRoundRobinIndex returns the current index for a round-robin action without advancing it.
func (r *Registry) GetRoundRobinIndex(key string, count int) int {
	if count <= 0 {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.roundRobinIndices[key] % count
}

// SetRoundRobinIndex sets the current index for a round-robin action.
func (r *Registry) SetRoundRobinIndex(key string, idx int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roundRobinIndices[key] = idx
}

// GetNextRoundRobinIndex returns the next index for a round-robin action, atomically.
func (r *Registry) GetNextRoundRobinIndex(key string, count int) int {
	if count <= 0 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	idx := r.roundRobinIndices[key] % count
	r.roundRobinIndices[key] = (idx + 1) % count
	return idx
}

// ResetRoundRobin clears the round-robin index for a specific handler or action key.
func (r *Registry) ResetRoundRobin(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.roundRobinIndices, key)
}

// GetNextSampleCount returns the next count for a sample action, atomically.
func (r *Registry) GetNextSampleCount(key string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sampleIndices[key]++
	return r.sampleIndices[key]
}

// ResetSample clears the sample count for a specific handler or action key.
func (r *Registry) ResetSample(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sampleIndices, key)
}

// ClearConnResources removes all resources associated with a specific connection ID.
func (r *Registry) ClearConnResources(connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear sequence indices
	for hName := range r.sequenceClientIndices {
		delete(r.sequenceClientIndices[hName], connID)
	}

	// Clear throttle timestamps
	suffix := ":" + connID
	for k := range r.throttleTimestamps {
		if strings.HasSuffix(k, suffix) {
			delete(r.throttleTimestamps, k)
		}
	}

	// Clear scoped limiters
	for k := range r.scopedLimiters {
		if strings.HasSuffix(k, suffix) {
			delete(r.scopedLimiters, k)
			delete(r.scopedLimiterRates, k)
			delete(r.scopedLimiterBursts, k)
		}
	}

	// Clear debouncers
	for k, d := range r.debouncers {
		if strings.HasSuffix(k, suffix) {
			d.mu.Lock()
			if d.timer != nil {
				d.timer.Stop()
			}
			d.mu.Unlock()
			delete(r.debouncers, k)
		}
	}
}
