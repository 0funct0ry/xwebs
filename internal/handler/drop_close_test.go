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

func TestDropBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil)

	h := &Handler{
		Name:    "drop-test",
		Builtin: "drop",
		Respond: "should not be sent",
	}

	msg := &ws.Message{
		Data: []byte("test"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg, nil)
	assert.Equal(t, ErrDrop, err)

	conn.mu.Lock()
	assert.Empty(t, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcherSequentialExecutionWithDrop(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil)

	var executed []string
	d.Log = func(f string, a ...interface{}) {
		executed = append(executed, a[0].(string))
	}
	// Note: We need a way to track execution without relying on Log which might be verbose.
	// But for this test, we'll just check if the second handler's response was sent.

	h1 := Handler{
		Name:     "h1-drop",
		Priority: 100,
		Match:    Matcher{Pattern: "drop-me"},
		Builtin:  "drop",
	}
	h2 := Handler{
		Name:     "h2-respond",
		Priority: 50,
		Match:    Matcher{Pattern: "drop-me"},
		Respond:  "i should not be here",
	}

	_ = reg.AddHandlers([]Handler{h1, h2})

	msg := &ws.Message{
		Data: []byte("drop-me"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	// handleMessage starts a goroutine. We need to wait or use a synchronous version for testing.
	// Since handleMessage is internal, we can't easily mock the 'go func'.
	// But we can check results after a short sleep.

	d.handleMessage(context.Background(), msg)
	time.Sleep(100 * time.Millisecond)

	conn.mu.Lock()
	assert.Empty(t, conn.lastWritten, "Subsequent handler should not have executed")
	conn.mu.Unlock()
}

func TestCloseBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil)

	h := &Handler{
		Name:    "close-test",
		Builtin: "close",
		Code:    "4000",
		Reason:  "test-reason-{{.Message}}",
	}

	msg := &ws.Message{
		Data: []byte("bye"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg, nil)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.True(t, conn.closed)
	assert.Equal(t, 4000, conn.closeCode)
	assert.Equal(t, "test-reason-bye", conn.closeReason)
	conn.mu.Unlock()
}
