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
	MsgsIn             uint64            `json:"msgs_in"`
	MsgsOut            uint64            `json:"msgs_out"`
	LastMsgReceivedAt  time.Time         `json:"last_msg_received_at"`
	LastMsgSentAt      time.Time         `json:"last_msg_sent_at"`
	RTT                time.Duration     `json:"rtt"`
	AvgRTT             time.Duration     `json:"avg_rtt"`
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

// ClientInfo provides metadata about a connected client in server mode.
type ClientInfo struct {
	ID          string        `json:"id"`
	RemoteAddr  string        `json:"remote_addr"`
	ConnectedAt time.Time     `json:"connected_at"`
	Uptime      time.Duration `json:"uptime"`
	UptimeStr   string        `json:"uptime_str"`
	MsgsIn      uint64        `json:"msgs_in"`
	MsgsOut     uint64        `json:"msgs_out"`
}

// TopicSubscriberInfo holds per-subscriber metadata for a pub/sub topic.
type TopicSubscriberInfo struct {
	ConnID       string    `json:"conn_id"`
	RemoteAddr   string    `json:"remote_addr"`
	SubscribedAt time.Time `json:"subscribed_at"`
	MsgsSent     uint64    `json:"msgs_sent"`
}

// TopicInfo holds metadata for a single pub/sub topic.
type TopicInfo struct {
	Name        string                `json:"name"`
	Subscribers []TopicSubscriberInfo `json:"subscribers"`
	LastActive  time.Time             `json:"last_active"`
}

// ServerContext provides global metrics when running in server mode.
type ServerContext struct {
	ClientCount     int           `json:"client_count"`
	Clients         []ClientInfo  `json:"clients"`
	Uptime          time.Duration `json:"uptime"`
	UptimeFormatted string        `json:"uptime_formatted"`
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

	// Activity stats
	MsgsIn         uint64 `json:"msgs_in,omitempty"`
	MsgsOut        uint64 `json:"msgs_out,omitempty"`
	HandlerHits    uint64 `json:"handler_hits,omitempty"`
	ActiveHandlers int    `json:"active_handlers,omitempty"`

	// Mode and State Indicators
	Mode           string `json:"mode,omitempty"`
	Status         string `json:"status,omitempty"`
	ReconnectCount int    `json:"reconnect_count,omitempty"`
	Port           int    `json:"port,omitempty"`
	IsSecure       bool   `json:"is_secure,omitempty"`
	RTT            string `json:"rtt,omitempty"`
	AvgRTT         string `json:"avg_rtt,omitempty"`

	// Server-level (convenience at root)
	ClientCount     int           `json:"client_count,omitempty"`
	Clients         []ClientInfo  `json:"clients,omitempty"`
	ServerUptime    time.Duration `json:"server_uptime,omitempty"`
	ServerUptimeStr string        `json:"server_uptime_str,omitempty"`

	// Key-Value Store
	KV         map[string]interface{} `json:"kv,omitempty"`       // Global KV store snapshot
	HttpStatus int                    `json:"http_status,omitempty"` // HTTP status code from http builtin
	HttpBody   string                 `json:"http_body,omitempty"`   // Response body from http builtin
	KvValue    interface{}            `json:"kv_value,omitempty"`    // Result of kv-get builtin
	KvKeys     []string               `json:"kv_keys,omitempty"`     // Result of kv-list builtin

	// Pattern Matching
	Matches []string `json:"matches,omitempty"` // Capture groups from glob/regex matching

	// Upstream Forwarding
	ForwardReply string `json:"forward_reply,omitempty"` // Result of forward builtin

	// Throttling
	DeliveredCount int `json:"delivered_count,omitempty"` // Result of throttle-broadcast builtin
	SkippedCount   int `json:"skipped_count,omitempty"`   // Result of throttle-broadcast builtin

	// Rate Limiting
	RetryAfter   float64 `json:"retry_after,omitempty"`
	RetryAfterMs int64   `json:"retry_after_ms,omitempty"`
	RateLimit    string  `json:"rate_limit,omitempty"`
	LimitScope   string  `json:"limit_scope,omitempty"`

	// Lua Builtin
	LuaError string `json:"lua_error,omitempty"`
}

// NewContext creates a new TemplateContext with initialized maps.
func NewContext() *TemplateContext {
	return &TemplateContext{
		Session: make(map[string]interface{}),
		Vars:    make(map[string]interface{}),
		Env:     make(map[string]string),
	}
}
