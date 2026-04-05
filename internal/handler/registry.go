package handler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/itchyny/gojq"
	"github.com/spf13/cast"
)

// Registry manages a collection of message handlers.
type Registry struct {
	handlers []Handler
}

// NewRegistry creates a new handler registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make([]Handler, 0),
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
func (r *Registry) Match(msg *ws.Message) ([]*Handler, error) {
	var matches []*Handler
	msgStr := string(msg.Data)

	for i := range r.handlers {
		h := &r.handlers[i]
		matched, err := r.matchHandler(h, msgStr)
		if err != nil {
			return nil, fmt.Errorf("matching handler %q: %w", h.Name, err)
		}
		if matched {
			matches = append(matches, h)
		}
	}

	return matches, nil
}

func (r *Registry) matchHandler(h *Handler, msg string) (bool, error) {
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
