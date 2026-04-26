package handler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type debounceMockConn struct {
	id      string
	sent    chan *ws.Message
	done    chan struct{}
	msgsIn  uint64
	msgsOut uint64
}

func (m *debounceMockConn) Write(msg *ws.Message) error {
	m.sent <- msg
	m.msgsOut++
	return nil
}
func (m *debounceMockConn) CloseWithCode(code int, reason string) error { return nil }
func (m *debounceMockConn) Subscribe() <-chan *ws.Message               { return nil }
func (m *debounceMockConn) Unsubscribe(ch <-chan *ws.Message)           {}
func (m *debounceMockConn) Done() <-chan struct{}                       { return m.done }
func (m *debounceMockConn) IsCompressionEnabled() bool                  { return false }
func (m *debounceMockConn) GetID() string                               { return m.id }
func (m *debounceMockConn) GetURL() string                              { return "ws://mock" }
func (m *debounceMockConn) GetSubprotocol() string                      { return "" }
func (m *debounceMockConn) RemoteAddr() string                          { return "127.0.0.1" }
func (m *debounceMockConn) LocalAddr() string                           { return "127.0.0.1" }
func (m *debounceMockConn) ConnectedAt() time.Time                      { return time.Now() }
func (m *debounceMockConn) MessageCount() uint64                        { return m.msgsIn + m.msgsOut }
func (m *debounceMockConn) MsgsIn() uint64                              { return m.msgsIn }
func (m *debounceMockConn) MsgsOut() uint64                             { return m.msgsOut }
func (m *debounceMockConn) LastMsgReceivedAt() time.Time                { return time.Now() }
func (m *debounceMockConn) LastMsgSentAt() time.Time                    { return time.Now() }
func (m *debounceMockConn) RTT() time.Duration                          { return 0 }
func (m *debounceMockConn) AvgRTT() time.Duration                       { return 0 }

func TestDebounceBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)

	t.Run("Basic debounce with respond", func(t *testing.T) {
		conn := &debounceMockConn{id: "c1", sent: make(chan *ws.Message, 10), done: make(chan struct{})}
		d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, nil, nil, nil)

		a := &Action{
			Type:        "builtin",
			Command:     "debounce",
			Window:      "50ms",
			Respond:     "Debounced: {{ .Message }}",
			HandlerName: "h1",
		}

		msg1 := &ws.Message{Data: []byte("first"), Metadata: ws.MessageMetadata{Timestamp: time.Now()}}
		tmplCtx1 := template.NewContext()
		d.populateTemplateContext(tmplCtx1, msg1)

		err := d.executeBuiltin(context.Background(), a, tmplCtx1)
		require.Equal(t, ErrDrop, err)

		time.Sleep(20 * time.Millisecond)

		msg2 := &ws.Message{Data: []byte("second"), Metadata: ws.MessageMetadata{Timestamp: time.Now()}}
		tmplCtx2 := template.NewContext()
		d.populateTemplateContext(tmplCtx2, msg2)

		err = d.executeBuiltin(context.Background(), a, tmplCtx2)
		require.Equal(t, ErrDrop, err)

		// Wait for window to expire (total > 50ms from msg2)
		select {
		case m := <-conn.sent:
			assert.Equal(t, "Debounced: second", string(m.Data))
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Timed out waiting for debounced response")
		}

		assert.Equal(t, 0, len(conn.sent), "Should only send one message")
	})

	t.Run("Scoped debouncing - client vs global", func(t *testing.T) {
		// Test client scope
		conn1 := &debounceMockConn{id: "client1", sent: make(chan *ws.Message, 10), done: make(chan struct{})}
		conn2 := &debounceMockConn{id: "client2", sent: make(chan *ws.Message, 10), done: make(chan struct{})}

		d1 := NewDispatcher(reg, conn1, engine, true, nil, nil, false, nil, nil, nil, nil)
		d2 := NewDispatcher(reg, conn2, engine, true, nil, nil, false, nil, nil, nil, nil)

		aClient := &Action{
			Type:        "builtin",
			Command:     "debounce",
			Window:      "50ms",
			Respond:     "Client: {{ .Message }}",
			Scope:       "client",
			HandlerName: "scoped-h",
		}

		msg1 := &ws.Message{Data: []byte("m1"), Metadata: ws.MessageMetadata{Timestamp: time.Now()}}
		tmplCtx1 := template.NewContext()
		d1.populateTemplateContext(tmplCtx1, msg1)

		msg2 := &ws.Message{Data: []byte("m2"), Metadata: ws.MessageMetadata{Timestamp: time.Now()}}
		tmplCtx2 := template.NewContext()
		d2.populateTemplateContext(tmplCtx2, msg2)

		_ = d1.executeBuiltin(context.Background(), aClient, tmplCtx1)
		_ = d2.executeBuiltin(context.Background(), aClient, tmplCtx2)

		// Both should trigger after 50ms independently
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			m := <-conn1.sent
			assert.Equal(t, "Client: m1", string(m.Data))
			wg.Done()
		}()
		go func() {
			m := <-conn2.sent
			assert.Equal(t, "Client: m2", string(m.Data))
			wg.Done()
		}()
		wg.Wait()

		// Test global scope
		aGlobal := &Action{
			Type:        "builtin",
			Command:     "debounce",
			Window:      "50ms",
			Respond:     "Global: {{ .Message }}",
			Scope:       "global",
			HandlerName: "global-h",
		}

		_ = d1.executeBuiltin(context.Background(), aGlobal, tmplCtx1)
		time.Sleep(20 * time.Millisecond)
		_ = d2.executeBuiltin(context.Background(), aGlobal, tmplCtx2) // Should reset timer

		select {
		case m := <-conn2.sent:
			assert.Equal(t, "Global: m2", string(m.Data))
		case <-conn1.sent:
			t.Fatal("conn1 should not receive global response if conn2 reset the timer")
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Timed out waiting for global response")
		}
	})
}
