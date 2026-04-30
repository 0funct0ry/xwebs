package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
)

// SSEEvent represents a single Server-Sent Event.
type SSEEvent struct {
	Stream  string
	Event   string
	Data    string
	ID      string
	Created time.Time
}

// SSEStream represents a named stream of events.
type SSEStream struct {
	Name           string
	Consumers      map[chan SSEEvent]struct{}
	Buffer         []SSEEvent
	BufferSize     int
	OnNoConsumers  string // "drop" or "buffer"
	mu             sync.RWMutex
	TotalSent      uint64
	TotalBuffered  uint64
}

// SSEManager manages multiple SSE streams.
type SSEManager struct {
	streams map[string]*SSEStream
	mu      sync.RWMutex
	verbose bool
	logf    func(string, ...interface{})
}

// NewSSEManager creates a new SSE manager.
func NewSSEManager(configs []handler.SSEStreamConfig, verbose bool, logf func(string, ...interface{})) *SSEManager {
	m := &SSEManager{
		streams: make(map[string]*SSEStream),
		verbose: verbose,
		logf:    logf,
	}

	for _, cfg := range configs {
		m.streams[cfg.Name] = &SSEStream{
			Name:          cfg.Name,
			Consumers:     make(map[chan SSEEvent]struct{}),
			OnNoConsumers: "drop", // Default
			BufferSize:    100,    // Default
		}
	}

	return m
}

// SendToSSE delivers an event to the named stream.
func (m *SSEManager) SendToSSE(streamName, event, data, id string) error {
	m.mu.RLock()
	s, ok := m.streams[streamName]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown SSE stream: %s", streamName)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sseEvent := SSEEvent{
		Stream:  streamName,
		Event:   event,
		Data:    data,
		ID:      id,
		Created: time.Now(),
	}

	if len(s.Consumers) == 0 {
		if s.OnNoConsumers == "buffer" {
			s.Buffer = append(s.Buffer, sseEvent)
			if len(s.Buffer) > s.BufferSize {
				s.Buffer = s.Buffer[1:] // Evict oldest
			}
			s.TotalBuffered++
		}
		return nil
	}

	for ch := range s.Consumers {
		select {
		case ch <- sseEvent:
			// Success
		default:
			// Consumer too slow, drop or handle? For now, we don't want to block.
		}
	}
	s.TotalSent++

	return nil
}

// HandleSSE returns an http.HandlerFunc for SSE consumers.
func (m *SSEManager) HandleSSE(streamName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		s, ok := m.streams[streamName]
		m.mu.RUnlock()

		if !ok {
			http.Error(w, "Stream not found", http.StatusNotFound)
			return
		}

		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := make(chan SSEEvent, 10)
		s.mu.Lock()
		s.Consumers[ch] = struct{}{}

		// Replay buffer if configured
		if s.OnNoConsumers == "buffer" && len(s.Buffer) > 0 {
			for _, ev := range s.Buffer {
				ch <- ev
			}
			// Should we clear the buffer after replay? The spec says "replayed in order to the next consumer that connects".
			// Usually in SSE, buffering is for any new consumer or for a specific reconnect.
			// Let's keep it for now.
		}
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			delete(s.Consumers, ch)
			s.mu.Unlock()
			close(ch)
		}()

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send initial retry hint
		fmt.Fprintf(w, "retry: 1000\n\n")
		flusher.Flush()

		// Set a long write deadline for the stream if supported
		rc := http.NewResponseController(w)
		_ = rc.SetWriteDeadline(time.Time{}) // Disable write timeout for this stream

		for {
			select {
			case ev := <-ch:
				if ev.ID != "" {
					fmt.Fprintf(w, "id: %s\n", ev.ID)
				}
				if ev.Event != "" && ev.Event != "message" {
					fmt.Fprintf(w, "event: %s\n", ev.Event)
				}
				fmt.Fprintf(w, "data: %s\n\n", ev.Data)
				flusher.Flush()
			case <-r.Context().Done():
				return
			case <-time.After(15 * time.Second):
				// Keep-alive comment
				fmt.Fprintf(w, ": keep-alive\n\n")
				flusher.Flush()
			}
		}
	}
}

// ListStreams returns metadata for all streams.
type SSEStreamInfo struct {
	Name          string
	ConsumerCount int
	BufferSize    int
	BufferDepth   int
	OnNoConsumers string
	TotalSent     uint64
	TotalBuffered uint64
}

func (m *SSEManager) ListStreams() []SSEStreamInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]SSEStreamInfo, 0, len(m.streams))
	for _, s := range m.streams {
		s.mu.RLock()
		infos = append(infos, SSEStreamInfo{
			Name:          s.Name,
			ConsumerCount: len(s.Consumers),
			BufferSize:    s.BufferSize,
			BufferDepth:   len(s.Buffer),
			OnNoConsumers: s.OnNoConsumers,
			TotalSent:     s.TotalSent,
			TotalBuffered: s.TotalBuffered,
		})
		s.mu.RUnlock()
	}
	return infos
}

func (m *SSEManager) GetStreamInfo(name string) (SSEStreamInfo, bool) {
	m.mu.RLock()
	s, ok := m.streams[name]
	m.mu.RUnlock()

	if !ok {
		return SSEStreamInfo{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return SSEStreamInfo{
		Name:          s.Name,
		ConsumerCount: len(s.Consumers),
		BufferSize:    s.BufferSize,
		BufferDepth:   len(s.Buffer),
		OnNoConsumers: s.OnNoConsumers,
		TotalSent:     s.TotalSent,
		TotalBuffered: s.TotalBuffered,
	}, true
}

func (m *SSEManager) ClearBuffer(name string) error {
	m.mu.RLock()
	s, ok := m.streams[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("stream not found: %s", name)
	}

	s.mu.Lock()
	s.Buffer = nil
	s.mu.Unlock()
	return nil
}

func (m *SSEManager) UpdateStreamConfig(name string, onNoConsumers string, bufferSize int) error {
	m.mu.RLock()
	s, ok := m.streams[name]
	m.mu.RUnlock()

	if !ok {
		// Auto-register if it doesn't exist? Spec says streams must be declared.
		// But REPL might want to add them.
		m.mu.Lock()
		s = &SSEStream{
			Name:          name,
			Consumers:     make(map[chan SSEEvent]struct{}),
			OnNoConsumers: onNoConsumers,
			BufferSize:    bufferSize,
		}
		m.streams[name] = s
		m.mu.Unlock()
		return nil
	}

	s.mu.Lock()
	s.OnNoConsumers = onNoConsumers
	s.BufferSize = bufferSize
	if len(s.Buffer) > s.BufferSize {
		s.Buffer = s.Buffer[len(s.Buffer)-s.BufferSize:]
	}
	s.mu.Unlock()
	return nil
}
