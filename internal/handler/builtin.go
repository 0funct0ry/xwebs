package handler

import (
	"sort"
	"strings"
)

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

// BuiltinMetadata contains documentation and scoping for a builtin action.
type BuiltinMetadata struct {
	Name        string
	Description string
	Scope       BuiltinScope
}

var (
	// builtinRegistry stores all defined builtins.
	builtinRegistry = map[string]BuiltinMetadata{
		"subscribe": {
			Name:        "subscribe",
			Description: "Subscribe the current connection to a pub/sub topic.",
			Scope:       ServerOnly,
		},
		"unsubscribe": {
			Name:        "unsubscribe",
			Description: "Unsubscribe the current connection from a pub/sub topic.",
			Scope:       ServerOnly,
		},
		"publish": {
			Name:        "publish",
			Description: "Publish a message to a pub/sub topic.",
			Scope:       ServerOnly,
		},
		"kv-set": {
			Name:        "kv-set",
			Description: "Store a value in the server's shared key-value store.",
			Scope:       ServerOnly,
		},
		"kv-get": {
			Name:        "kv-get",
			Description: "Retrieve a value from the server's shared key-value store into .KvValue.",
			Scope:       ServerOnly,
		},
		"kv-del": {
			Name:        "kv-del",
			Description: "Delete a key from the server's shared key-value store.",
			Scope:       ServerOnly,
		},
		"noop": {
			Name:        "noop",
			Description: "A shared builtin that does nothing (useful for testing).",
			Scope:       Shared,
		},
	}
)

// GetBuiltin returns the metadata for a builtin by name.
func GetBuiltin(name string) (BuiltinMetadata, bool) {
	m, ok := builtinRegistry[strings.ToLower(strings.TrimSpace(name))]
	return m, ok
}

// ListBuiltins returns a sorted list of builtins available for the given mode.
func ListBuiltins(mode RegistryMode) []BuiltinMetadata {
	var results []BuiltinMetadata
	for _, m := range builtinRegistry {
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
	m, ok := GetBuiltin(name)
	if !ok {
		return false, false, ""
	}

	switch m.Scope {
	case Shared:
		return true, true, Shared
	case ClientOnly:
		return mode == ClientMode, true, ClientOnly
	case ServerOnly:
		return mode == ServerMode, true, ServerOnly
	default:
		return false, true, m.Scope
	}
}
