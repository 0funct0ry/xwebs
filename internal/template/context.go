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
	CompressionEnabled bool              `json:"compression_enabled"`
}

// MessageContext provides details about a specific WebSocket message.
type MessageContext struct {
	Type      string    `json:"type"`      // "text", "binary", "ping", "pong"
	Data      string    `json:"data"`      // string representation of the data
	Raw       []byte    `json:"raw"`       // raw byte slice
	Length    int       `json:"length"`    // size of the data in bytes
	Timestamp time.Time `json:"timestamp"` // when the message was received/sent
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
	Conn    *ConnectionContext     `json:"conn,omitempty"`
	Msg     *MessageContext        `json:"msg,omitempty"`
	Handler *HandlerContext        `json:"handler,omitempty"`
	Server  *ServerContext         `json:"server,omitempty"`
	Session map[string]interface{} `json:"session"`
	Env     map[string]string      `json:"env"`
}

// NewContext creates a new TemplateContext with initialized maps.
func NewContext() *TemplateContext {
	return &TemplateContext{
		Session: make(map[string]interface{}),
		Env:     make(map[string]string),
	}
}
