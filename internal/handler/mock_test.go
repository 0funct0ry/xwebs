package handler

import (
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
)

// mockConn implements ws.Connection interface for testing
type mockConn struct {
	lastWritten string
	messages    []*ws.Message
	mu          sync.Mutex
	done        chan struct{}
	closed      bool
	closeCode   int
	closeReason string
}

func (m *mockConn) Write(msg *ws.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastWritten = string(msg.Data)
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockConn) CloseWithCode(code int, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.closeCode = code
	m.closeReason = reason
	return nil
}

func (m *mockConn) Subscribe() <-chan *ws.Message     { return nil }
func (m *mockConn) Unsubscribe(ch <-chan *ws.Message) {}
func (m *mockConn) Done() <-chan struct{} {
	if m.done != nil {
		return m.done
	}
	return nil
}
func (m *mockConn) IsCompressionEnabled() bool   { return false }
func (m *mockConn) GetID() string                { return "mock-conn-id" }
func (m *mockConn) GetURL() string               { return "ws://localhost:8080" }
func (m *mockConn) GetSubprotocol() string       { return "" }
func (m *mockConn) RemoteAddr() string           { return "127.0.0.1:12345" }
func (m *mockConn) LocalAddr() string            { return "127.0.0.1:8080" }
func (m *mockConn) ConnectedAt() time.Time       { return time.Now().Add(-1 * time.Minute) }
func (m *mockConn) MessageCount() uint64         { return 10 }
func (m *mockConn) MsgsIn() uint64               { return 5 }
func (m *mockConn) MsgsOut() uint64              { return 5 }
func (m *mockConn) LastMsgReceivedAt() time.Time { return time.Now().Add(-10 * time.Second) }
func (m *mockConn) LastMsgSentAt() time.Time     { return time.Now().Add(-5 * time.Second) }
func (m *mockConn) RTT() time.Duration           { return 50 * time.Millisecond }
func (m *mockConn) AvgRTT() time.Duration        { return 45 * time.Millisecond }
