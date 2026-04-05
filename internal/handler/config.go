package handler

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the root structure of a handlers.yaml file.
type Config struct {
	Variables map[string]interface{} `yaml:"variables"`
	Handlers  []Handler              `yaml:"handlers"`
	BaseDir   string                 `yaml:"-"` // Directory from which the config was loaded
}

// Handler defines a single message handler with a name, match conditions, and actions.
type Handler struct {
	Name         string         `yaml:"name"`
	Priority     int            `yaml:"priority,omitempty"`
	Exclusive    bool           `yaml:"exclusive,omitempty"`
	Concurrent   *bool          `yaml:"concurrent,omitempty"` // Pointer to distinguish between false and not set (default true)
	Match        Matcher        `yaml:"match"`
	Run          string         `yaml:"run,omitempty"`      // Shorthand for shell action
	Respond      string         `yaml:"respond,omitempty"`  // Shorthand for send action (after run)
	Builtin      string         `yaml:"builtin,omitempty"`  // Shorthand for builtin action
	Pipeline     []PipelineStep `yaml:"pipeline,omitempty"` // Multi-step pipeline
	Timeout      string         `yaml:"timeout,omitempty"`  // Per-handler timeout
	Actions      []Action       `yaml:"actions,omitempty"`
	OnConnect    []Action       `yaml:"on_connect,omitempty"`
	OnDisconnect []Action       `yaml:"on_disconnect,omitempty"`
	OnError      []Action       `yaml:"on_error,omitempty"`
	BaseDir      string         `yaml:"-"` // Directory from which the handler was loaded
}

// PipelineStep defines a single step in a multi-step handler execution.
type PipelineStep struct {
	Run         string `yaml:"run,omitempty"`
	Builtin     string `yaml:"builtin,omitempty"`
	As          string `yaml:"as,omitempty"`      // Key to store results in .Steps.<name>
	Timeout     string `yaml:"timeout,omitempty"`
	IgnoreError bool   `yaml:"ignore_error,omitempty"`
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
	All        []Matcher   `yaml:"all,omitempty"`         // AND logic
	Any        []Matcher   `yaml:"any,omitempty"`         // OR logic
}

// UnmarshalYAML implements custom unmarshaling for Matcher to support shorthands.
func (m *Matcher) UnmarshalYAML(value *yaml.Node) error {
	// If it's a string, it's a glob shorthand: match: "*"
	if value.Kind == yaml.ScalarNode {
		m.Type = "glob"
		m.Pattern = value.Value
		return nil
	}

	// Otherwise, unmarshal as the full struct
	type alias Matcher
	var a alias
	if err := value.Decode(&a); err != nil {
		return err
	}
	*m = Matcher(a)
	return nil
}

// Action defines an operation to perform when a handler matches or a lifecycle event occurs.
type Action struct {
	Type    string            `yaml:"action,omitempty"`  // "shell", "send", "log", "builtin"
	Run     string            `yaml:"run,omitempty"`      // Shorthand for shell action
	Send    string            `yaml:"send,omitempty"`     // Shorthand for send action
	Builtin string            `yaml:"builtin,omitempty"`  // Shorthand for builtin action
	Log     string            `yaml:"log,omitempty"`      // Shorthand for log message
	Message string            `yaml:"message,omitempty"`  // For legacy "send" action (shorthand preferred)
	Command string            `yaml:"command,omitempty"`  // For legacy "shell" action (shorthand preferred)
	Target  string            `yaml:"target,omitempty"`   // For "log" action (e.g. filename or "stdout", "stderr")
	Timeout string            `yaml:"timeout,omitempty"`  // Timeout for shell/builtin actions
	Env     map[string]string `yaml:"env,omitempty"`      // Environment variables for shell actions
	Silent  bool              `yaml:"silent,omitempty"`   // Suppress output for shell actions
}

// UnmarshalYAML implements custom unmarshaling for Action to support shorthand keys.
func (a *Action) UnmarshalYAML(value *yaml.Node) error {
	type alias Action
	var tmp alias
	if err := value.Decode(&tmp); err != nil {
		return err
	}
	*a = Action(tmp)

	// Infer Type from shorthand keys if not explicitly set
	if a.Type == "" {
		if a.Run != "" {
			a.Type = "shell"
		} else if a.Send != "" {
			a.Type = "send"
			a.Message = a.Send
		} else if a.Builtin != "" {
			a.Type = "builtin"
			a.Command = a.Builtin
		} else if a.Log != "" {
			a.Type = "log"
			a.Message = a.Log
		}
	}
	return nil
}

// Validate checks the configuration for common errors.
func (c *Config) Validate() error {
	for i, h := range c.Handlers {
		if h.Name == "" {
			return fmt.Errorf("handler[%d] is missing a name", i)
		}
		
		// Match is required if we have execution logic
		hasMatch := h.Match.Pattern != "" || h.Match.Regex != "" || h.Match.JQ != "" || 
			h.Match.JSONPath != "" || h.Match.JSONSchema != "" || h.Match.Template != "" || 
			h.Match.Binary != nil || len(h.Match.All) > 0 || len(h.Match.Any) > 0
		
		hasExecution := len(h.Actions) > 0 || h.Run != "" || h.Respond != "" || 
			h.Builtin != "" || len(h.Pipeline) > 0
		
		if !hasMatch && hasExecution {
			return fmt.Errorf("handler %q is missing a match condition (pattern, regex, jq, json_path, json_schema, template, binary, all, or any)", h.Name)
		}

		if !hasExecution && len(h.OnConnect) == 0 && len(h.OnDisconnect) == 0 && len(h.OnError) == 0 {
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
		if a.Run == "" && a.Command == "" {
			return fmt.Errorf("shell action missing run or command")
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
