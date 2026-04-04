package handler

import (
	"fmt"
	"strings"
)

// Config represents the root structure of a handlers.yaml file.
type Config struct {
	Variables map[string]interface{} `yaml:"variables"`
	Handlers  []Handler              `yaml:"handlers"`
}

// Handler defines a single message handler with a name, match conditions, and actions.
type Handler struct {
	Name         string   `yaml:"name"`
	Priority     int      `yaml:"priority,omitempty"`
	Match        Matcher  `yaml:"match"`
	Actions      []Action `yaml:"actions"`
	OnConnect    []Action `yaml:"on_connect,omitempty"`
	OnDisconnect []Action `yaml:"on_disconnect,omitempty"`
	OnError      []Action `yaml:"on_error,omitempty"`
}

// Matcher specifies how to match an incoming WebSocket message.
type Matcher struct {
	Type    string `yaml:"type"`    // "text", "json", "regex", "glob" (default "text")
	Pattern string `yaml:"pattern"` // The pattern to match against (can be a template)
}

// Action defines an operation to perform when a handler matches or a lifecycle event occurs.
type Action struct {
	Type    string `yaml:"action"`            // "shell", "send", "log", "builtin"
	Command string `yaml:"command,omitempty"`  // For "shell" and "builtin" actions (can be a template)
	Message string `yaml:"message,omitempty"`  // For "send" action (can be a template)
	Target  string `yaml:"target,omitempty"`   // For "log" action (e.g. filename or "stdout", "stderr")
	Silent  bool   `yaml:"silent,omitempty"`   // Suppress output for shell actions
}

// Validate checks the configuration for common errors.
func (c *Config) Validate() error {
	for i, h := range c.Handlers {
		if h.Name == "" {
			return fmt.Errorf("handler[%d] is missing a name", i)
		}
		
		// Match is required if there are actions (normal handler)
		// But if it only has lifecycle events, maybe match is not required?
		// Usually a handler is either matching messages or lifecycle events.
		if h.Match.Pattern == "" && len(h.Actions) > 0 {
			return fmt.Errorf("handler %q is missing a match pattern", h.Name)
		}

		if len(h.Actions) == 0 && len(h.OnConnect) == 0 && len(h.OnDisconnect) == 0 && len(h.OnError) == 0 {
			return fmt.Errorf("handler %q has no actions or lifecycle events", h.Name)
		}

		for j, a := range h.Actions {
			if err := a.Validate(); err != nil {
				return fmt.Errorf("handler %q action[%d]: %w", h.Name, j, err)
			}
		}
		for j, a := range h.OnConnect {
			if err := a.Validate(); err != nil {
				return fmt.Errorf("handler %q on_connect action[%d]: %w", h.Name, j, err)
			}
		}
		for j, a := range h.OnDisconnect {
			if err := a.Validate(); err != nil {
				return fmt.Errorf("handler %q on_disconnect action[%d]: %w", h.Name, j, err)
			}
		}
		for j, a := range h.OnError {
			if err := a.Validate(); err != nil {
				return fmt.Errorf("handler %q on_error action[%d]: %w", h.Name, j, err)
			}
		}
	}

	return nil
}

// Validate checks a single action for required fields based on its type.
func (a *Action) Validate() error {
	switch strings.ToLower(a.Type) {
	case "shell":
		if a.Command == "" {
			return fmt.Errorf("shell action missing command")
		}
	case "send":
		if a.Message == "" {
			return fmt.Errorf("send action missing message")
		}
	case "log":
		if a.Message == "" {
			return fmt.Errorf("log action missing message")
		}
	case "builtin":
		if a.Command == "" {
			return fmt.Errorf("builtin action missing command")
		}
	case "":
		return fmt.Errorf("missing action type")
	default:
		return fmt.Errorf("unknown action type: %s", a.Type)
	}
	return nil
}
