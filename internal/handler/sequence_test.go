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

type mockConnection struct {
	CapturedMessages []*ws.Message
	ID               string
}

func (m *mockConnection) Write(msg *ws.Message) error {
	m.CapturedMessages = append(m.CapturedMessages, msg)
	return nil
}
func (m *mockConnection) CloseWithCode(code int, reason string) error { return nil }
func (m *mockConnection) Subscribe() <-chan *ws.Message               { return nil }
func (m *mockConnection) Unsubscribe(ch <-chan *ws.Message)           {}
func (m *mockConnection) Done() <-chan struct{}                       { return nil }
func (m *mockConnection) IsCompressionEnabled() bool                  { return false }
func (m *mockConnection) GetID() string                               { return m.ID }
func (m *mockConnection) GetURL() string                              { return "ws://localhost" }
func (m *mockConnection) GetSubprotocol() string                      { return "" }
func (m *mockConnection) RemoteAddr() string                          { return "127.0.0.1" }
func (m *mockConnection) LocalAddr() string                           { return "127.0.0.1" }
func (m *mockConnection) ConnectedAt() time.Time                      { return time.Now() }
func (m *mockConnection) MessageCount() uint64                        { return 0 }
func (m *mockConnection) MsgsIn() uint64                              { return 0 }
func (m *mockConnection) MsgsOut() uint64                             { return 0 }
func (m *mockConnection) LastMsgReceivedAt() time.Time                { return time.Now() }
func (m *mockConnection) LastMsgSentAt() time.Time                    { return time.Now() }
func (m *mockConnection) RTT() time.Duration                          { return 0 }
func (m *mockConnection) AvgRTT() time.Duration                       { return 0 }

func TestSequenceBuiltin(t *testing.T) {
	registry := NewRegistry(ServerMode)
	engine := template.New(false)

	conn := &mockConnection{ID: "client-1"}
	dispatcher := NewDispatcher(registry, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

	action := &Action{
		Command:     "sequence",
		Responses:   []string{"one", "two", "three"},
		HandlerName: "test-handler",
	}

	tmplCtx := template.NewContext()
	bh, ok := GetBuiltin("sequence")
	require.True(t, ok)

	// 1. First execution
	err := bh.Execute(context.Background(), dispatcher, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "one", string(conn.CapturedMessages[0].Data))

	// 2. Second execution
	err = bh.Execute(context.Background(), dispatcher, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "two", string(conn.CapturedMessages[1].Data))

	// 3. Third execution
	err = bh.Execute(context.Background(), dispatcher, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "three", string(conn.CapturedMessages[2].Data))

	// 4. Fourth execution (loop=false by default, stay on last)
	err = bh.Execute(context.Background(), dispatcher, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "three", string(conn.CapturedMessages[3].Data))

	// 5. Test Loop = true
	action.Loop = true
	// Current is 2. loop is true.
	// GetNextSequenceIndex(2, loop=true) returns 2, next = (2+1)%3 = 0.
	err = bh.Execute(context.Background(), dispatcher, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "three", string(conn.CapturedMessages[4].Data))

	// Next execution should be "one"
	err = bh.Execute(context.Background(), dispatcher, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "one", string(conn.CapturedMessages[5].Data))

	// 6. Test Reset
	registry.ResetSequence("test-handler")
	err = bh.Execute(context.Background(), dispatcher, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "one", string(conn.CapturedMessages[6].Data))
}

func TestSequenceBuiltinPerClient(t *testing.T) {
	registry := NewRegistry(ServerMode)
	engine := template.New(false)

	conn1 := &mockConnection{ID: "client-1"}
	conn2 := &mockConnection{ID: "client-2"}

	d1 := NewDispatcher(registry, conn1, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")
	d2 := NewDispatcher(registry, conn2, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

	action := &Action{
		Command:     "sequence",
		Responses:   []string{"one", "two"},
		HandlerName: "multi-client",
		PerClient:   true,
		Loop:        true,
	}

	tmplCtx := template.NewContext()
	bh, ok := GetBuiltin("sequence")
	require.True(t, ok)

	// Client 1: step 1
	err := bh.Execute(context.Background(), d1, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "one", string(conn1.CapturedMessages[0].Data))

	// Client 2: step 1 (should be 'one' because independent)
	err = bh.Execute(context.Background(), d2, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "one", string(conn2.CapturedMessages[0].Data))

	// Client 1: step 2
	err = bh.Execute(context.Background(), d1, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "two", string(conn1.CapturedMessages[1].Data))

	// Client 2: step 2
	err = bh.Execute(context.Background(), d2, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, "two", string(conn2.CapturedMessages[1].Data))
}
