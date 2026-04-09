package template

import (
	"time"
)

// ConnectionContext provides metadata about the WebSocket connection.
type ConnectionContext struct {
	URL                string            `json:"url"`
	Subprotocol        string            `json:"subprotocol"`
	Headers            map[string]string `json:"headers"`
	LocalAddr          string            `json:"local_addr"`
	RemoteAddr         string            `json:"remote_addr"`
	ClientIP           string            `json:"client_ip"`
	ConnectedAt        time.Time         `json:"connected_at"`
	Uptime             time.Duration     `json:"uptime"`
	UptimeFormatted    string            `json:"uptime_formatted"`
	MessageCount       uint64            `json:"message_count"`
	CompressionEnabled bool              `json:"compression_enabled"`
}

// MessageContext provides details about a specific WebSocket message.
type MessageContext struct {
	Type      string      `json:"type"`      // "text", "binary", "ping", "pong"
	Data      interface{} `json:"data"`      // parsed data if JSON, otherwise string
	Raw       []byte      `json:"raw"`       // raw byte slice
	Length    int         `json:"length"`    // size of the data in bytes
	Timestamp time.Time   `json:"timestamp"` // when the message was received/sent
}

// HandlerContext captures the results of a handler execution.
type HandlerContext struct {
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
}

// ServerContext provides global metrics when running in server mode.
type ServerContext struct {
	ClientCount int           `json:"client_count"`
	Uptime      time.Duration `json:"uptime"`
}

// TemplateContext is the root object passed to all templates.
type TemplateContext struct {
	// Root-level markers (optional but spec-mandated in some contexts)
	Conn    *ConnectionContext `json:"conn,omitempty"`
	Msg     *MessageContext    `json:"msg,omitempty"`
	Handler *HandlerContext    `json:"handler,omitempty"`
	Server  *ServerContext     `json:"server,omitempty"`

	// Connection Context (root-level convenience)
	URL          string            `json:"url,omitempty"`
	ConnectionID string            `json:"connection_id,omitempty"`
	Host         string            `json:"host,omitempty"`
	Path         string            `json:"path,omitempty"`
	Scheme       string            `json:"scheme,omitempty"`
	RemoteAddr   string            `json:"remote_addr,omitempty"`
	Subprotocol  string            `json:"subprotocol,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`

	// Message context (root-level convenience)
	Message      string    `json:"message,omitempty"`
	MessageBytes []byte    `json:"message_bytes,omitempty"`
	MessageLen   int       `json:"message_len,omitempty"`
	MessageType  string    `json:"message_type,omitempty"`
	MessageIndex uint64    `json:"message_index,omitempty"`
	Timestamp    time.Time `json:"timestamp,omitempty"`
	Direction    string    `json:"direction,omitempty"`

	// Session-level stats
	SessionID       string        `json:"session_id,omitempty"`
	MessageCount    uint64        `json:"message_count,omitempty"`
	ConnectedSince  time.Time     `json:"connected_since,omitempty"`
	Uptime          time.Duration `json:"uptime,omitempty"`
	UptimeFormatted string        `json:"uptime_formatted,omitempty"`
	ClientIP        string        `json:"client_ip,omitempty"`
	LocalAddr       string        `json:"local_addr,omitempty"`

	// Execution results (root-level convenience for single action)
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`

	// Pipeline steps results
	Steps map[string]*HandlerContext `json:"steps,omitempty"`

	// Session/Environment
	Session map[string]interface{} `json:"session"`
	Vars    map[string]interface{} `json:"vars"`
	Env     map[string]string      `json:"env"`

	// Scripting context
	Last          string `json:"last,omitempty"`
	LastLatencyMs int64  `json:"last_latency_ms,omitempty"`
	Error         string `json:"error,omitempty"`
}

// NewContext creates a new TemplateContext with initialized maps.
func NewContext() *TemplateContext {
	return &TemplateContext{
		Session: make(map[string]interface{}),
		Vars:    make(map[string]interface{}),
		Env:     make(map[string]string),
	}
}
