package handler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/0funct0ry/xwebs/internal/template"
)

// ErrLimitExceeded is returned by the rate-limit builtin when a message is rejected.
var ErrLimitExceeded = fmt.Errorf("rate limit exceeded")

// ErrDrop is a sentinel error returned by the drop builtin to signal that
// no further actions or handlers should be executed for the current message.
var ErrDrop = fmt.Errorf("drop message")

// ErrClose is a sentinel error that can be used to signal connection closure.
var ErrClose = fmt.Errorf("close connection")

// BuiltinScope defines where a builtin action is allowed to run.
type BuiltinScope string

const (
	// Shared builtins are available in both client and server modes.
	Shared BuiltinScope = "Shared"
	// ClientOnly builtins are only available in client mode connect handlers.
	ClientOnly BuiltinScope = "ClientOnly"
	// ServerOnly builtins are only available in server mode serve/relay handlers.
	ServerOnly BuiltinScope = "ServerOnly"
)

// BuiltinField defines a single configuration field for a builtin action.
type BuiltinField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// BuiltinHelp contains detailed documentation and examples for a builtin action.
type BuiltinHelp struct {
	Description     string            `json:"description"`
	Fields          []BuiltinField    `json:"fields,omitempty"`
	TemplateVars    map[string]string `json:"template_vars,omitempty"`
	YAMLReplExample string            `json:"yaml_example,omitempty"`
	REPLAddExample  string            `json:"repl_example,omitempty"`
}

// DocumentedBuiltin is an optional interface for builtins that provide rich documentation.
type DocumentedBuiltin interface {
	BuiltinHandler
	Help() BuiltinHelp
}

// BuiltinMetadata contains documentation and scoping for a builtin action.
type BuiltinMetadata struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Scope       BuiltinScope   `json:"scope"`
	Fields      []BuiltinField `json:"fields,omitempty"`
}

// BuiltinHandler defines the interface for all built-in actions.
type BuiltinHandler interface {
	Name() string
	Description() string
	Scope() BuiltinScope
	Validate(a Action) error
	Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error
}

var (
	// builtinRegistry stores all defined builtins.
	builtinRegistry   = make(map[string]BuiltinHandler)
	builtinRegistryMu sync.RWMutex
)

// Register adds a new builtin handler to the registry.
// It returns an error if a builtin with the same name already exists.
func Register(h BuiltinHandler) error {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()

	name := strings.ToLower(strings.TrimSpace(h.Name()))
	if _, ok := builtinRegistry[name]; ok {
		return fmt.Errorf("builtin action %q already registered", name)
	}

	builtinRegistry[name] = h
	return nil
}

// MustRegister is like Register but panics if the registration fails.
// This is typically used in init() functions.
func MustRegister(h BuiltinHandler) {
	if err := Register(h); err != nil {
		panic(err)
	}
}

// GetBuiltinResult contains both the handler and its metadata.
type GetBuiltinResult struct {
	Handler  BuiltinHandler
	Metadata BuiltinMetadata
}

// GetBuiltin returns the handler for a builtin by name.
func GetBuiltin(name string) (BuiltinHandler, bool) {
	builtinRegistryMu.RLock()
	defer builtinRegistryMu.RUnlock()

	h, ok := builtinRegistry[strings.ToLower(strings.TrimSpace(name))]
	return h, ok
}

// ListBuiltins returns a sorted list of builtin metadata available for the given mode.
func ListBuiltins(mode RegistryMode) []BuiltinMetadata {
	builtinRegistryMu.RLock()
	defer builtinRegistryMu.RUnlock()

	var results []BuiltinMetadata
	for _, h := range builtinRegistry {
		m := BuiltinMetadata{
			Name:        h.Name(),
			Description: h.Description(),
			Scope:       h.Scope(),
		}

		if db, ok := h.(DocumentedBuiltin); ok {
			help := db.Help()
			m.Fields = help.Fields
		}

		allowed := false
		switch m.Scope {
		case Shared:
			allowed = true
		case ClientOnly:
			allowed = (mode == ClientMode)
		case ServerOnly:
			allowed = (mode == ServerMode)
		}

		if allowed {
			results = append(results, m)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

// IsBuiltinAllowed checks if a builtin is allowed in the given mode.
func IsBuiltinAllowed(name string, mode RegistryMode) (allowed bool, exists bool, scope BuiltinScope) {
	h, ok := GetBuiltin(name)
	if !ok {
		return false, false, ""
	}

	scope = h.Scope()
	switch scope {
	case Shared:
		return true, true, Shared
	case ClientOnly:
		return mode == ClientMode, true, ClientOnly
	case ServerOnly:
		return mode == ServerMode, true, ServerOnly
	default:
		return false, true, scope
	}
}
