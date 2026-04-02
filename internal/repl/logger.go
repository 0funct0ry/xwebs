package repl

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
)

// Logger handles structured JSONL logging of WebSocket traffic and events.
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	filename string
	count    int
}

// LogEntry represents a single line in the JSONL log file.
type LogEntry struct {
	TS      string `json:"ts"`
	Dir     string `json:"dir,omitempty"`
	Type    string `json:"type,omitempty"`
	Len     int    `json:"len,omitempty"`
	Msg     string `json:"msg,omitempty"`
	MsgBase64 string `json:"msg_b64,omitempty"`
	Conn    string `json:"conn,omitempty"`
	Event   string `json:"event,omitempty"`
	URL     string `json:"url,omitempty"`
	SubProto string `json:"subprotocol,omitempty"`
	Code    int    `json:"code,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// NewLogger creates a new Logger instance.
func NewLogger() *Logger {
	return &Logger{}
}

// Start opens the specified file in append mode.
func (l *Logger) Start(filename string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		_ = l.file.Close()
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	l.file = f
	l.filename = filename
	l.count = 0
	_ = l.logEventLocked("logging-started", map[string]interface{}{"file": filename})
	return nil
}

// Stop closes the log file and returns the number of entries written.
func (l *Logger) Stop() (int, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return 0, "", nil
	}

	count := l.count
	filename := l.filename
	err := l.file.Close()
	l.file = nil
	l.filename = ""
	l.count = 0

	return count, filename, err
}

// IsActive returns true if the logger is currently active.
func (l *Logger) IsActive() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file != nil
}

// Filename returns the current log file path.
func (l *Logger) Filename() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.filename
}

// LogMessage writes a message entry to the log.
func (l *Logger) LogMessage(msg *ws.Message) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logMessageLocked(msg)
}

func (l *Logger) logMessageLocked(msg *ws.Message) error {
	if l.file == nil {
		return nil
	}

	entry := LogEntry{
		TS:   msg.Metadata.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z"),
		Dir:  msg.Metadata.Direction,
		Len:  msg.Metadata.Length,
		Conn: msg.Metadata.ID,
	}

	// Normalizing direction to match story requirements ("send"/"recv")
	if entry.Dir == "sent" {
		entry.Dir = "send"
	} else if entry.Dir == "received" {
		entry.Dir = "recv"
	}

	switch msg.Type {
	case ws.TextMessage:
		entry.Type = "text"
		entry.Msg = string(msg.Data)
	case ws.BinaryMessage:
		entry.Type = "binary"
		entry.MsgBase64 = base64.StdEncoding.EncodeToString(msg.Data)
	case ws.PingMessage:
		entry.Type = "ping"
		entry.Msg = string(msg.Data)
	case ws.PongMessage:
		entry.Type = "pong"
		entry.Msg = string(msg.Data)
	}

	return l.writeEntry(entry)
}

// LogEvent writes a connection event entry to the log.
func (l *Logger) LogEvent(event string, details map[string]interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logEventLocked(event, details)
}

func (l *Logger) logEventLocked(event string, details map[string]interface{}) error {
	if l.file == nil {
		return nil
	}

	entry := LogEntry{
		TS:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		Event: event,
	}

	if url, ok := details["url"].(string); ok {
		entry.URL = url
	}
	if sub, ok := details["subprotocol"].(string); ok {
		entry.SubProto = sub
	}
	if code, ok := details["code"].(int); ok {
		entry.Code = code
	}
	if reason, ok := details["reason"].(string); ok {
		entry.Reason = reason
	}

	return l.writeEntry(entry)
}

func (l *Logger) writeEntry(entry LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}

	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing log entry: %w", err)
	}

	l.count++
	return nil
}
