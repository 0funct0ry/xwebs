package handler

import (
	"fmt"
	"strings"
)

// Config represents the root structure of a handlers.yaml file.
type Config struct {
	Variables map[string]interface{} `yaml:"variables"`
	Handlers  []Handler              `yaml:"handlers"`
	BaseDir   string                 `yaml:"-"` // Directory from which the config was loaded
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
	BaseDir      string   `yaml:"-"` // Directory from which the handler was loaded
}

// Matcher specifies how to match an incoming WebSocket message.
type Matcher struct {
	Type       string      `yaml:"type,omitempty"`        // "text", "json", "regex", "glob", "jq", "json_schema" (default "text")
	Pattern    string      `yaml:"pattern,omitempty"`     // The pattern to match against
	Regex      string      `yaml:"regex,omitempty"`       // Shorthand for regex matching
	JQ         string      `yaml:"jq,omitempty"`          // Shorthand for jq matching
	JSONPath   string      `yaml:"json_path,omitempty"`   // JSONPath to extract value
	Equals     interface{} `yaml:"equals,omitempty"`      // Value to compare with (string or number)
	JSONSchema string      `yaml:"json_schema,omitempty"` // Path to JSON Schema file
	Template   string      `yaml:"template,omitempty"`    // Go template for complex matching
	Binary     *bool       `yaml:"binary,omitempty"`      // Match binary vs text frames
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
		if h.Match.Pattern == "" && h.Match.Regex == "" && h.Match.JQ == "" && h.Match.JSONPath == "" && h.Match.JSONSchema == "" && h.Match.Template == "" && h.Match.Binary == nil && len(h.Actions) > 0 {
			return fmt.Errorf("handler %q is missing a match condition (pattern, regex, jq, json_path, json_schema, template, or binary)", h.Name)
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
