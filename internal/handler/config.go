package handler

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the root structure of a handlers.yaml file.
type Config struct {
	Variables map[string]interface{} `yaml:"variables"`
	Handlers  []Handler              `yaml:"handlers"`
	Sandbox   bool                   `yaml:"sandbox"`
	Allowlist []string               `yaml:"allowlist"`
	BaseDir   string                 `yaml:"-"` // Directory from which the config was loaded
}

// Handler defines a single message handler with a name, match conditions, and actions.
type Handler struct {
	Name         string                 `yaml:"name"`
	Priority     int                    `yaml:"priority,omitempty"`
	Exclusive    bool                   `yaml:"exclusive,omitempty"`
	Concurrent   *bool                  `yaml:"concurrent,omitempty"` // Pointer to distinguish between false and not set (default true)
	Match        Matcher                `yaml:"match"`
	Run          string                 `yaml:"run,omitempty"`        // Shorthand for shell action
	Respond      string                 `yaml:"respond,omitempty"`    // Shorthand for send action (after run)
	Builtin      string                 `yaml:"builtin,omitempty"`    // Shorthand for builtin action
	Topic        string                 `yaml:"topic,omitempty"`      // Topic name (template) for subscribe/unsubscribe/publish builtins
	Key          string                 `yaml:"key,omitempty"`        // Key (template) for KV builtins
	Value        string                 `yaml:"value,omitempty"`      // Value (template) for KV builtins
	Target       string                 `yaml:"target,omitempty"`     // Upstream target URL for forward builtin
	Pipeline     []PipelineStep         `yaml:"pipeline,omitempty"`   // Multi-step pipeline
	Timeout      string                 `yaml:"timeout,omitempty"`    // Per-handler timeout
	Retry        *RetryConfig           `yaml:"retry,omitempty"`      // Automatic retry on failure
	RateLimit    string                 `yaml:"rate_limit,omitempty"` // Per-handler rate limit (e.g. "10/s", "100/m")
	Debounce     string                 `yaml:"debounce,omitempty"`   // Per-handler debounce duration (e.g. "500ms")
	Delay        string                 `yaml:"delay,omitempty"`      // Per-handler delay (e.g. "1s")
	Message      string                 `yaml:"message,omitempty"`    // Message content (template) for broadcast/publish builtins
	TTL          string                 `yaml:"ttl,omitempty"`        // TTL (template) for KV builtins
	Default      string                 `yaml:"default,omitempty"`    // Default value (template) for KV builtins
	Responses    []string               `yaml:"responses,omitempty"`  // For sequence builtin
	Loop         bool                   `yaml:"loop,omitempty"`       // For sequence builtin
	PerClient    bool                   `yaml:"per_client,omitempty"` // For sequence builtin
	Path         string                 `yaml:"path,omitempty"`       // For file-write builtin
	Content      string                 `yaml:"content,omitempty"`    // For file-write builtin
	Duration     string                 `yaml:"duration,omitempty"`   // For delay builtin (supports templates)
	Max          string                 `yaml:"max,omitempty"`        // For delay builtin — cap on dynamic duration
	Code         string                 `yaml:"code,omitempty"`       // For close builtin (supports templates)
	Reason       string                 `yaml:"reason,omitempty"`     // For close builtin (supports templates)
	Actions      []Action               `yaml:"actions,omitempty"`
	Variables    map[string]interface{} `yaml:"variables,omitempty"`
	OnConnect    []Action               `yaml:"on_connect,omitempty"`
	OnDisconnect []Action               `yaml:"on_disconnect,omitempty"`
	OnError      []Action               `yaml:"on_error,omitempty"`
	OnErrorMsg   string                 `yaml:"on_error_msg,omitempty"` // Shorthand for on_error send: template
	File         string                 `yaml:"file,omitempty"`         // For template builtin
	Mode         string                 `yaml:"mode,omitempty"`         // For file-send builtin
	URL          string                 `yaml:"url,omitempty"`          // For http builtin
	Method       string                 `yaml:"method,omitempty"`       // For http builtin
	Headers      map[string]string      `yaml:"headers,omitempty"`      // For http builtin
	Body         string                 `yaml:"body,omitempty"`         // For http builtin
	Rate         string                 `yaml:"rate,omitempty"`         // For rate-limit builtin
	Burst        int                    `yaml:"burst,omitempty"`        // For rate-limit builtin
	Scope        string                 `yaml:"scope,omitempty"`        // For rate-limit builtin
	OnLimit      string                 `yaml:"on_limit,omitempty"`     // For rate-limit builtin
	BaseDir      string                 `yaml:"-"`                      // Directory from which the handler was loaded
}

// RetryConfig defines the settings for automatic retries.
type RetryConfig struct {
	Count       int    `yaml:"count"`                  // Number of retry attempts
	Backoff     string `yaml:"backoff,omitempty"`      // Strategy: "linear" or "exponential" (default "linear")
	Interval    string `yaml:"interval,omitempty"`     // Initial wait duration (default "1s")
	MaxInterval string `yaml:"max_interval,omitempty"` // Cap for exponential backoff (default "30s")
}

// PipelineStep defines a single step in a multi-step handler execution.
type PipelineStep struct {
	Run         string   `yaml:"run,omitempty"`
	Builtin     string   `yaml:"builtin,omitempty"`
	Topic       string            `yaml:"topic,omitempty"`  // Topic name (template) for subscribe/unsubscribe/publish builtins
	Key         string            `yaml:"key,omitempty"`    // Key (template) for KV builtins
	Value       string            `yaml:"value,omitempty"`  // Value (template) for KV builtins
	Target      string            `yaml:"target,omitempty"` // Upstream target URL for forward builtin
	As          string            `yaml:"as,omitempty"`     // Key to store results in .Steps.<name>
	Timeout     string            `yaml:"timeout,omitempty"`
	Delay       string            `yaml:"delay,omitempty"`
	Respond     string            `yaml:"respond,omitempty"`
	Message     string            `yaml:"message,omitempty"`    // Message content (template) for broadcast/publish builtins
	TTL         string            `yaml:"ttl,omitempty"`        // TTL (template) for KV builtins
	Default     string            `yaml:"default,omitempty"`    // Default value (template) for KV builtins
	Responses   []string          `yaml:"responses,omitempty"`  // For sequence builtin
	Loop        bool              `yaml:"loop,omitempty"`       // For sequence builtin
	PerClient   bool              `yaml:"per_client,omitempty"` // For sequence builtin
	File        string            `yaml:"file,omitempty"`       // For template builtin
	Path        string            `yaml:"path,omitempty"`       // For file-write builtin
	Content     string            `yaml:"content,omitempty"`    // For file-write builtin
	Mode        string            `yaml:"mode,omitempty"`       // For file-send builtin
	URL         string            `yaml:"url,omitempty"`        // For http builtin
	Method      string            `yaml:"method,omitempty"`     // For http builtin
	Headers     map[string]string `yaml:"headers,omitempty"`    // For http builtin
	Body        string            `yaml:"body,omitempty"`       // For http builtin
	IgnoreError bool              `yaml:"ignore_error,omitempty"`
	Rate        string            `yaml:"rate,omitempty"`     // For rate-limit builtin
	Burst       int               `yaml:"burst,omitempty"`    // For rate-limit builtin
	Scope       string            `yaml:"scope,omitempty"`    // For rate-limit builtin
	OnLimit     string            `yaml:"on_limit,omitempty"` // For rate-limit builtin
	Duration    string            `yaml:"duration,omitempty"` // For delay builtin (supports templates)
	Max         string            `yaml:"max,omitempty"`      // For delay builtin — cap on dynamic duration
	Code        string            `yaml:"code,omitempty"`     // For close builtin
	Reason      string            `yaml:"reason,omitempty"`   // For close builtin
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
	Type        string            `yaml:"action,omitempty"`  // "shell", "send", "log", "builtin"
	Run         string            `yaml:"run,omitempty"`     // Shorthand for shell action
	Send        string            `yaml:"send,omitempty"`    // Shorthand for send action
	Builtin     string            `yaml:"builtin,omitempty"` // Shorthand for builtin action
	Log         string            `yaml:"log,omitempty"`     // Shorthand for log message
	Message     string            `yaml:"message,omitempty"` // For legacy "send" action (shorthand preferred)
	Command     string            `yaml:"command,omitempty"` // For legacy "shell" action (shorthand preferred)
	Topic       string            `yaml:"topic,omitempty"`   // Topic name (template) for subscribe/unsubscribe/publish builtins
	Key         string            `yaml:"key,omitempty"`     // Key (template) for KV builtins
	Value       string            `yaml:"value,omitempty"`   // Value (template) for KV builtins
	Target      string            `yaml:"target,omitempty"`  // For "log" action (e.g. filename or "stdout", "stderr")
	Timeout     string            `yaml:"timeout,omitempty"` // Timeout for shell/builtin actions
	Delay       string            `yaml:"delay,omitempty"`   // Delay before execution
	Respond     string            `yaml:"respond,omitempty"` // Override response for echo or generic follow-up
	TTL         string            `yaml:"ttl,omitempty"`     // TTL (template) for KV builtins
	Default     string            `yaml:"default,omitempty"` // Default value (template) for KV builtins
	Env         map[string]string `yaml:"env,omitempty"`     // Environment variables for shell actions
	Silent      bool              `yaml:"silent,omitempty"`  // Suppress output for shell actions
	Responses   []string          `yaml:"responses,omitempty"`
	Loop        bool              `yaml:"loop,omitempty"`
	PerClient   bool              `yaml:"per_client,omitempty"`
	File        string            `yaml:"file,omitempty"`     // For template builtin
	Path        string            `yaml:"path,omitempty"`     // For file-write builtin
	Content     string            `yaml:"content,omitempty"`  // For file-write builtin
	URL         string            `yaml:"url,omitempty"`      // For http builtin
	Method      string            `yaml:"method,omitempty"`   // For http builtin
	Headers     map[string]string `yaml:"headers,omitempty"`  // For http builtin
	Body        string            `yaml:"body,omitempty"`     // For http builtin
	Mode        string            `yaml:"mode,omitempty"`     // For file-send builtin
	Rate        string            `yaml:"rate,omitempty"`     // For rate-limit builtin
	Burst       int               `yaml:"burst,omitempty"`    // For rate-limit builtin
	Scope       string            `yaml:"scope,omitempty"`    // For rate-limit builtin
	OnLimit     string            `yaml:"on_limit,omitempty"` // For rate-limit builtin
	Duration    string            `yaml:"duration,omitempty"` // For delay builtin (supports templates)
	Max         string            `yaml:"max,omitempty"`      // For delay builtin — cap on dynamic duration
	Code        string            `yaml:"code,omitempty"`     // For close builtin
	Reason      string            `yaml:"reason,omitempty"`   // For close builtin
	BaseDir     string            `yaml:"-"`                  // For relative path resolution in builtins
	HandlerName string            `yaml:"-"`                  // Internal use only
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

// Validate checks the configuration for common errors within the given mode context.
func (c *Config) Validate(mode RegistryMode) error {
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
			if err := a.Validate(mode); err != nil {
				return fmt.Errorf("handler %q action[%d]: %w", h.Name, j, err)
			}
		}
		for j, a := range h.OnConnect {
			if err := a.Validate(mode); err != nil {
				return fmt.Errorf("handler %q on_connect action[%d]: %w", h.Name, j, err)
			}
		}
		for j, a := range h.OnDisconnect {
			if err := a.Validate(mode); err != nil {
				return fmt.Errorf("handler %q on_disconnect action[%d]: %w", h.Name, j, err)
			}
		}
		for j, a := range h.OnError {
			if err := a.Validate(mode); err != nil {
				return fmt.Errorf("handler %q on_error action[%d]: %w", h.Name, j, err)
			}
		}

		if h.Retry != nil {
			if err := h.Retry.Validate(); err != nil {
				return fmt.Errorf("handler %q retry config: %w", h.Name, err)
			}
		}

		if h.RateLimit != "" {
			if _, _, err := ParseRateLimit(h.RateLimit); err != nil {
				return fmt.Errorf("handler %q invalid rate_limit %q: %w", h.Name, h.RateLimit, err)
			}
		}

		if h.Debounce != "" {
			if _, err := time.ParseDuration(h.Debounce); err != nil {
				return fmt.Errorf("handler %q invalid debounce %q: %w", h.Name, h.Debounce, err)
			}
		}
		if h.Delay != "" {
			if _, err := time.ParseDuration(h.Delay); err != nil {
				return fmt.Errorf("handler %q invalid delay %q: %w", h.Name, h.Delay, err)
			}
		}

		// Validate top-level builtin (shorthand)
		if h.Builtin != "" {
			bh, exists := GetBuiltin(h.Builtin)
			if !exists {
				return fmt.Errorf("handler %q: unknown builtin action: %s", h.Name, h.Builtin)
			}

			allowed, _, _ := IsBuiltinAllowed(h.Builtin, mode)
			if !allowed {
				if mode == ClientMode {
					return fmt.Errorf("handler %q: builtin %q is only available in server mode", h.Name, h.Builtin)
				}
				return fmt.Errorf("handler %q: builtin %q is only available in client mode", h.Name, h.Builtin)
			}

			// Validate builtin-specific fields using the handler's Validate method.
			// Wrap the shorthand fields into a temporary Action for validation.
			tmpAction := Action{
				Type:      "builtin",
				Command:   h.Builtin,
				Topic:     h.Topic,
				Key:       h.Key,
				Value:     h.Value,
				Target:    h.Target,
				Timeout:   h.Timeout,
				Delay:     h.Delay,
				Respond:   h.Respond,
				Responses: h.Responses,
				Loop:      h.Loop,
				PerClient: h.PerClient,
				File:      h.File,
				Path:      h.Path,
				Content:   h.Content,
				Mode:      h.Mode,
				Rate:      h.Rate,
				Burst:     h.Burst,
				Scope:     h.Scope,
				OnLimit:   h.OnLimit,
				Duration:  h.Duration,
				Max:       h.Max,
				Code:      h.Code,
				Reason:    h.Reason,
				URL:       h.URL,
				Method:    h.Method,
				Headers:   h.Headers,
				Body:      h.Body,
			}
			if err := bh.Validate(tmpAction); err != nil {
				return fmt.Errorf("handler %q: %w", h.Name, err)
			}
		}

		// Validate top-level target (shorthand usage)
		if h.Target != "" && h.Builtin != "forward" {
			return fmt.Errorf("handler %q: target property is only allowed for 'forward' builtin", h.Name)
		}
		if h.Builtin == "forward" && h.Target == "" {
			return fmt.Errorf("handler %q: 'forward' builtin requires a target", h.Name)
		}
	}

	return nil
}

// Validate checks a single action for required fields and mode compatibility.
func (a *Action) Validate(mode RegistryMode) error {
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
		bh, exists := GetBuiltin(a.Command)
		if !exists {
			return fmt.Errorf("unknown builtin action: %s", a.Command)
		}

		allowed, _, scope := IsBuiltinAllowed(a.Command, mode)
		if !allowed {
			m := "server"
			if scope == ClientOnly {
				m = "client"
			}
			return fmt.Errorf("builtin %q is only available in %s mode", a.Command, m)
		}

		if err := bh.Validate(*a); err != nil {
			return err
		}
	case "":
		return fmt.Errorf("missing action type")
	default:
		return fmt.Errorf("unknown action type: %s", a.Type)
	}

	if a.Delay != "" {
		if _, err := time.ParseDuration(a.Delay); err != nil {
			return fmt.Errorf("invalid delay %q: %w", a.Delay, err)
		}
	}

	return nil
}

// Validate checks retry settings for common errors.
func (r *RetryConfig) Validate() error {
	if r.Count < 0 {
		return fmt.Errorf("retry count cannot be negative")
	}
	if r.Interval != "" {
		if _, err := time.ParseDuration(r.Interval); err != nil {
			return fmt.Errorf("invalid retry interval %q: %w", r.Interval, err)
		}
	}
	if r.MaxInterval != "" {
		if _, err := time.ParseDuration(r.MaxInterval); err != nil {
			return fmt.Errorf("invalid retry max_interval %q: %w", r.MaxInterval, err)
		}
	}
	if r.Backoff != "" {
		b := strings.ToLower(r.Backoff)
		if b != "linear" && b != "exponential" {
			return fmt.Errorf("unknown backoff strategy %q (must be 'linear' or 'exponential')", r.Backoff)
		}
	}
	return nil
}

// ParseRateLimit parses a rate limit string like "10/s", "100/m", "5/h"
// and returns the tokens per second and the burst size.
func ParseRateLimit(s string) (float64, int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format (expected 'N/unit')")
	}

	valStr := parts[0]
	unitStr := parts[1]

	var val float64
	if _, err := fmt.Sscanf(valStr, "%f", &val); err != nil {
		return 0, 0, fmt.Errorf("invalid number %q: %w", valStr, err)
	}

	if val <= 0 {
		return 0, 0, fmt.Errorf("rate must be positive")
	}

	var perSec float64
	switch unitStr {
	case "s", "sec", "second":
		perSec = val
	case "m", "min", "minute":
		perSec = val / 60.0
	case "h", "hr", "hour":
		perSec = val / 3600.0
	default:
		return 0, 0, fmt.Errorf("unknown unit %q", unitStr)
	}

	// For burst, we use the value itself as a sensible default for the token bucket.
	// This means you can burst up to the count within the time window.
	burst := int(val)
	if burst < 1 {
		burst = 1
	}

	return perSec, burst, nil
}
