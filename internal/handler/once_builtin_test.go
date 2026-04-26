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

func TestOnceBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	
	// Create a mock connection
	conn := &mockConn{}

	engine := template.New(false)
	dispatcher := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, nil, nil, nil)

	// 1. Add a handler with 'once' builtin
	h := Handler{
		Name: "init-handler",
		Match: Matcher{
			Pattern: "hello",
		},
		Builtin: "once",
		Respond: "Initialised",
	}
	err := reg.Add(h)
	require.NoError(t, err)

	// 2. First match should execute and disable the handler
	msg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte("hello"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	// Manually trigger handleMessage (as Start loop is async)
	dispatcher.handleMessage(context.Background(), msg)

	// Wait for response (async)
	time.Sleep(100 * time.Millisecond)

	conn.mu.Lock()
	assert.Equal(t, "Initialised", conn.lastWritten)
	conn.mu.Unlock()

	// Verify handler is disabled with reason "once"
	assert.True(t, reg.IsDisabled("init-handler"))
	assert.Equal(t, "once", reg.GetDisabledReason("init-handler"))

	// 3. Subsequent match should NOT execute
	conn.mu.Lock()
	conn.lastWritten = ""
	conn.mu.Unlock()

	dispatcher.handleMessage(context.Background(), msg)
	time.Sleep(100 * time.Millisecond)

	conn.mu.Lock()
	assert.Equal(t, "", conn.lastWritten)
	conn.mu.Unlock()
}

func TestOnceBuiltin_Scope(t *testing.T) {
	reg := NewRegistry(ClientMode)
	h := Handler{
		Name: "client-once",
		Match: Matcher{Pattern: "*"},
		Builtin: "once",
	}
	
	// 'once' is ServerOnly, so adding it to a ClientMode registry should fail validation
	err := reg.Add(h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only available in server mode")
}
