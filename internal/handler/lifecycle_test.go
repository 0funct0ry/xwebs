package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
)

type mockLifecycleConn struct {
	ws.Connection // embed for interface compliance
	messages      []*ws.Message
	done          chan struct{}
}

func (m *mockLifecycleConn) Write(msg *ws.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockLifecycleConn) Subscribe() <-chan *ws.Message {
	return make(chan *ws.Message)
}

func (m *mockLifecycleConn) Unsubscribe(ch <-chan *ws.Message) {}

func (m *mockLifecycleConn) Done() <-chan struct{} {
	return m.done
}

func (m *mockLifecycleConn) GetURL() string {
	return "ws://test"
}

func (m *mockLifecycleConn) GetSubprotocol() string {
	return ""
}

func (m *mockLifecycleConn) IsCompressionEnabled() bool {
	return false
}

func TestLifecycleHooks(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockLifecycleConn{done: make(chan struct{})}

	// 1. Test on_connect
	reg.AddHandlers([]Handler{
		{
			Name: "conn_handler",
			OnConnect: []Action{
				{Type: "send", Message: "hello from connect"},
			},
		},
	})

	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil)
	d.HandleConnect()

	assert.Len(t, conn.messages, 1)
	assert.Equal(t, "hello from connect", string(conn.messages[0].Data))

	// 2. Test on_disconnect
	reg.AddHandlers([]Handler{
		{
			Name: "disc_handler",
			OnDisconnect: []Action{
				{Type: "send", Message: "bye bye"},
			},
		},
	})
	// Re-fetch sorted handlers via HandleDisconnect internally
	d.HandleDisconnect()

	assert.Len(t, conn.messages, 2)
	assert.Equal(t, "bye bye", string(conn.messages[1].Data))

	// 3. Test on_error with .Error context
	reg.AddHandlers([]Handler{
		{
			Name: "err_handler",
			OnError: []Action{
				{Type: "send", Message: "Error occurred: {{.Error}}"},
			},
		},
	})
	
	testErr := errors.New("something went wrong")
	d.HandleError(testErr)

	assert.Len(t, conn.messages, 3)
	assert.Contains(t, string(conn.messages[2].Data), "Error occurred: something went wrong")
}

func TestHandlerErrorTrigger(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockLifecycleConn{done: make(chan struct{})}

	// Handler that fails
	reg.AddHandlers([]Handler{
		{
			Name: "failing_handler",
			Match: Matcher{Type: "glob", Pattern: "*"},
			Run: "exit 1",
		},
		{
			Name: "error_watcher",
			OnError: []Action{
				{Type: "send", Message: "detected handler failure: {{.Error}}"},
			},
		},
	})

	d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil)
	
	msg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte("trigger"),
		Metadata: ws.MessageMetadata{Direction: "received"},
	}

	// Execute synchronously for testing
	matches, _ := reg.Match(msg, engine, template.NewContext())
	for _, h := range matches {
		err := d.Execute(context.Background(), h, msg)
		if h.Name == "failing_handler" {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

	// Wait a bit for async-ish things if any (though Execute is sync here)
	time.Sleep(100 * time.Millisecond)

	assert.True(t, len(conn.messages) >= 1, "Should have received an error notification")
	found := false
	for _, m := range conn.messages {
		if string(m.Data) == "detected handler failure: command failed with exit code 1" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected error notification message not found")
}
