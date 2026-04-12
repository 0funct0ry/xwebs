package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		input     string
		wantPerS  float64
		wantBurst int
		wantErr   bool
	}{
		{"10/s", 10, 10, false},
		{"10/sec", 10, 10, false},
		{"10/second", 10, 10, false},
		{"60/m", 1, 60, false},
		{"1/m", 1.0 / 60.0, 1, false},
		{"3600/h", 1, 3600, false},
		{"1.5/s", 1.5, 1, false},
		{"invalid", 0, 0, true},
		{"10", 0, 0, true},
		{"10/day", 0, 0, true},
		{"0/s", 0, 0, true},
		{"-1/s", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			perS, burst, err := ParseRateLimit(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.wantPerS, perS, 0.0001)
				assert.Equal(t, tt.wantBurst, burst)
			}
		})
	}
}

type rateLimitMockConn struct {
	Connection
	writeCount int
}

func (m *rateLimitMockConn) Write(msg *ws.Message) error {
	m.writeCount++
	return nil
}
func (m *rateLimitMockConn) Subscribe() <-chan *ws.Message     { return nil }
func (m *rateLimitMockConn) Unsubscribe(ch <-chan *ws.Message) {}
func (m *rateLimitMockConn) Done() <-chan struct{}             { return make(chan struct{}) }
func (m *rateLimitMockConn) GetURL() string                    { return "ws://localhost" }
func (m *rateLimitMockConn) GetSubprotocol() string            { return "" }
func (m *rateLimitMockConn) IsCompressionEnabled() bool        { return false }
func (m *rateLimitMockConn) RemoteAddr() string                { return "127.0.0.1:12345" }
func (m *rateLimitMockConn) LocalAddr() string                 { return "127.0.0.1:8080" }
func (m *rateLimitMockConn) ConnectedAt() time.Time            { return time.Now().Add(-1 * time.Minute) }
func (m *rateLimitMockConn) MessageCount() uint64              { return 0 }
func (m *rateLimitMockConn) MsgsIn() uint64                    { return 0 }
func (m *rateLimitMockConn) MsgsOut() uint64                   { return 0 }
func (m *rateLimitMockConn) LastMsgReceivedAt() time.Time      { return time.Time{} }
func (m *rateLimitMockConn) LastMsgSentAt() time.Time          { return time.Time{} }
func (m *rateLimitMockConn) RTT() time.Duration                { return 0 }
func (m *rateLimitMockConn) AvgRTT() time.Duration             { return 0 }

func TestDispatcherRateLimit(t *testing.T) {
	registry := NewRegistry()
	handler := Handler{
		Name:      "limited",
		RateLimit: "2/s", // 2 per second, burst 2
		Match:     Matcher{Pattern: "hello"},
		Respond:   "limited response",
	}
	registry.AddHandlers([]Handler{handler})

	conn := &rateLimitMockConn{}
	engine := template.New(false)
	dispatcher := NewDispatcher(registry, conn, engine, true, nil, nil, false, nil, nil)

	ctx := context.Background()
	msg := &ws.Message{
		Data: []byte("hello"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
		},
	}

	// First two should pass (burst = 2)
	dispatcher.handleMessage(ctx, msg)
	dispatcher.handleMessage(ctx, msg)

	// Wait a bit for async execution to complete since handleMessage spawns goroutines
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, conn.writeCount, "Should have executed 2 times")

	// Third should be dropped
	dispatcher.handleMessage(ctx, msg)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, conn.writeCount, "Should still be 2 due to rate limit")

	// Wait 1.1s to allow 2 more tokens
	time.Sleep(1100 * time.Millisecond)
	dispatcher.handleMessage(ctx, msg)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 3, conn.writeCount, "Should have executed 3 times after waiting")
}
