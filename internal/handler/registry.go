package handler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/itchyny/gojq"
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
	if h.Match.Pattern == "" {
		return false, nil
	}

	switch strings.ToLower(h.Match.Type) {
	case "text", "":
		// Trim whitespace for more resilient matching in interactive sessions
		return strings.TrimSpace(msg) == h.Match.Pattern, nil
	case "regex":
		matched, err := regexp.MatchString(h.Match.Pattern, msg)
		if err != nil {
			return false, fmt.Errorf("regex error: %w", err)
		}
		return matched, nil
	case "glob":
		return r.matchGlob(h.Match.Pattern, msg)
	case "json":
		return r.matchJSON(h.Match.Pattern, msg)
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
	if err, ok := v.(error); ok {
		return false, fmt.Errorf("executing gojq query: %w", err)
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
