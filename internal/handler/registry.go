package handler

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/itchyny/gojq"
	"github.com/spf13/cast"
	"github.com/xeipuuv/gojsonschema"
)

// Registry manages a collection of message handlers.
type Registry struct {
	handlers []Handler
	schemas  map[string]*gojsonschema.Schema
	mu       sync.RWMutex
}

// NewRegistry creates a new handler registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make([]Handler, 0),
		schemas:  make(map[string]*gojsonschema.Schema),
	}
}

// AddHandlers adds multiple handlers to the registry and sorts them by priority.
func (r *Registry) AddHandlers(handlers []Handler) {
	r.handlers = append(r.handlers, handlers...)
	r.sort()
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
	msgStr := string(msg.Data)

	for i := range r.handlers {
		h := &r.handlers[i]
		matched, err := r.matchHandler(h, msgStr, engine, ctx)
		if err != nil {
			return nil, fmt.Errorf("matching handler %q: %w", h.Name, err)
		}
		if matched {
			matches = append(matches, h)
		}
	}

	return matches, nil
}

func (r *Registry) matchHandler(h *Handler, msg string, engine *template.Engine, ctx *template.TemplateContext) (bool, error) {
	// Trim whitespace for more resilient matching in interactive sessions (e.g. echo servers with newlines)
	trimmedMsg := strings.TrimSpace(msg)

	// Support regex shorthand: match.regex: "pattern"
	if h.Match.Regex != "" {
		matched, err := regexp.MatchString(h.Match.Regex, trimmedMsg)
		if err != nil {
			return false, fmt.Errorf("regex shorthand error: %w", err)
		}
		return matched, nil
	}

	// Support jq shorthand: match.jq: "query"
	if h.Match.JQ != "" {
		return r.matchJSON(h.Match.JQ, trimmedMsg)
	}
	
	// Support json_path + equals: match.json_path: "path", match.equals: value
	if h.Match.JSONPath != "" {
		return r.matchJSONPath(h.Match.JSONPath, h.Match.Equals, trimmedMsg)
	}

	// Support json_schema: match.json_schema: "path/to/schema.json"
	if h.Match.JSONSchema != "" {
		return r.matchJSONSchema(h.Match.JSONSchema, h.BaseDir, trimmedMsg)
	}

	// Support template: match.template: "{{ eq .msg.data.type 'alert' }}"
	if h.Match.Template != "" {
		return r.matchTemplate(engine, ctx, h.Match.Template)
	}

	if h.Match.Pattern == "" {
		return false, nil
	}

	switch strings.ToLower(h.Match.Type) {
	case "text", "":
		// Trim whitespace for more resilient matching in interactive sessions
		return strings.TrimSpace(msg) == h.Match.Pattern, nil
	case "regex":
		matched, err := regexp.MatchString(h.Match.Pattern, trimmedMsg)
		if err != nil {
			return false, fmt.Errorf("regex error: %w", err)
		}
		return matched, nil
	case "glob":
		return r.matchGlob(h.Match.Pattern, trimmedMsg)
	case "json", "jq":
		return r.matchJSON(h.Match.Pattern, trimmedMsg)
	case "json_schema":
		return r.matchJSONSchema(h.Match.Pattern, h.BaseDir, trimmedMsg)
	case "template":
		return r.matchTemplate(engine, ctx, h.Match.Pattern)
	default:
		return false, fmt.Errorf("unknown matcher type: %s", h.Match.Type)
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
