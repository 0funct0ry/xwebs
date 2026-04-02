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

// RecordEntry represents a single line in the JSONL recording.
type RecordEntry struct {
	TS      int64  `json:"ts"`                // Milliseconds since recording start
	Dir     string `json:"dir,omitempty"`     // "send" or "recv"
	Msg     string `json:"msg,omitempty"`     // Raw message text
	MsgBase64 string `json:"msg_b64,omitempty"` // Base64 payload (binary frames)
}

// RecordHeader defines the first line of the recording session.
type RecordHeader struct {
	XR       int    `json:"xwebs_recording"`
	URL      string `json:"url"`
	Started  string `json:"started"`
	Version  string `json:"version"`
}

// Recorder captures a WebSocket session for deterministic replay.
type Recorder struct {
	mu        sync.Mutex
	file      *os.File
	filename  string
	startTime time.Time
	count     int
}

// NewRecorder creates a new Recorder instance.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// Start opens the file in overwrite mode and writes the header.
func (r *Recorder) Start(filename string, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		_ = r.file.Close()
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("opening recording file: %w", err)
	}

	r.file = f
	r.filename = filename
	r.startTime = time.Now()
	r.count = 0

	header := RecordHeader{
		XR:      1,
		URL:     url,
		Started: r.startTime.UTC().Format(time.RFC3339),
		Version: "0.1.0",
	}

	data, _ := json.Marshal(header)
	if _, err := r.file.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		r.file = nil
		return fmt.Errorf("writing recording header: %w", err)
	}

	return nil
}

// Stop closes the recording file and returns message count and filename.
func (r *Recorder) Stop() (int, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return 0, "", nil
	}

	count := r.count
	filename := r.filename
	err := r.file.Close()
	r.file = nil
	r.filename = ""
	r.count = 0

	return count, filename, err
}

// IsActive returns true if the recorder is currently active.
func (r *Recorder) IsActive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file != nil
}

// RecordMessage captures a message with relative timing.
func (r *Recorder) RecordMessage(msg *ws.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return nil
	}

	elapsed := time.Since(r.startTime).Milliseconds()
	entry := RecordEntry{
		TS:  elapsed,
		Dir: msg.Metadata.Direction,
	}

	// Normalizing direction
	if entry.Dir == "sent" {
		entry.Dir = "send"
	} else if entry.Dir == "received" {
		entry.Dir = "recv"
	}

	switch msg.Type {
	case ws.TextMessage:
		entry.Msg = string(msg.Data)
	case ws.BinaryMessage:
		entry.MsgBase64 = base64.StdEncoding.EncodeToString(msg.Data)
	case ws.PingMessage:
		entry.Dir = "ping" // Not strictly in story, but useful for replay if we ever want to
		entry.Msg = string(msg.Data)
	case ws.PongMessage:
		entry.Dir = "pong"
		entry.Msg = string(msg.Data)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling record entry: %w", err)
	}

	if _, err := r.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing record entry: %w", err)
	}

	r.count++
	return nil
}
