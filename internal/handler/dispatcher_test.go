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

// mockConn implements ws.Connection interface for testing
type mockConn struct {
	lastWritten string
	mu          sync.Mutex
}

func (m *mockConn) Write(msg *ws.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastWritten = string(msg.Data)
	return nil
}

func (m *mockConn) Subscribe() <-chan *ws.Message { return nil }
func (m *mockConn) Unsubscribe(ch <-chan *ws.Message) {}
func (m *mockConn) Done() <-chan struct{} { return nil }
func (m *mockConn) IsCompressionEnabled() bool { return false }
func (m *mockConn) GetURL() string { return "ws://localhost:8080" }
func (m *mockConn) GetSubprotocol() string { return "" }

func TestDispatcher_ExecutePipeline(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil)

	h := &Handler{
		Name: "pipeline-test",
		Pipeline: []PipelineStep{
			{Run: "echo 'hello' | tr '[:lower:]' '[:upper:]'", As: "step1"},
			{Run: "echo '{{.Steps.step1.Stdout | trim}} world'", As: "step2"},
		},
		Respond: `{"result": "{{.Steps.step2.Stdout | trim}}"}`,
	}

	msg := &ws.Message{
		Data: []byte("test"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"result": "HELLO world"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_ExecutePipelineFailure(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil)

	h := &Handler{
		Name: "pipeline-failure",
		Pipeline: []PipelineStep{
			{Run: "exit 2", As: "step1"},
			{Run: "echo 'should not reach here'", As: "step2"},
		},
		Respond: `{"result": "unreachable"}`,
	}

	msg := &ws.Message{
		Data: []byte("test"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline step \"step1\" failed: exit code 2")

	conn.mu.Lock()
	assert.Empty(t, conn.lastWritten) // Respond should not have run
	conn.mu.Unlock()
}

func TestDispatcher_ExecutePipelineIgnoreError(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil)

	h := &Handler{
		Name: "pipeline-ignore",
		Pipeline: []PipelineStep{
			{Run: "exit 2", As: "step1", IgnoreError: true},
			{Run: "echo 'rescued'", As: "step2"},
		},
		Respond: `{"code": {{.Steps.step1.ExitCode}}, "result": "{{.Steps.step2.Stdout | trim}}"}`,
	}

	msg := &ws.Message{
		Data: []byte("test"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"code": 2, "result": "rescued"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_GlobalVariables(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}
	vars := map[string]interface{}{"tmp": "/tmp/test", "version": 1}
	d := NewDispatcher(reg, conn, engine, false, vars)

	h := &Handler{
		Name:    "vars-test",
		Run:     "echo 'using {{.Vars.tmp}} v{{.Vars.version}}'",
		Respond: `{"status": "ok", "output": "{{.Stdout | trim}}"}`,
	}

	msg := &ws.Message{
		Data: []byte("data"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"status": "ok", "output": "using /tmp/test v1"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_Shorthands(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil)

	h := &Handler{
		Name:    "shorthand-test",
		Run:     "echo 'processed {{.Message}}'",
		Respond: `{"status": "ok", "output": "{{.Stdout | trim}}"}`,
	}

	msg := &ws.Message{
		Data: []byte("data"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"status": "ok", "output": "processed data"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_RespondContext(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil)

	h := &Handler{
		Name:    "respond-context-test",
		Run:     "echo 'some error' >&2; exit 2",
		Respond: `{"code": {{.ExitCode}}, "err": "{{.Stderr | trim}}"}`,
	}

	msg := &ws.Message{
		Data: []byte("test"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	err := d.Execute(context.Background(), h, msg)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"code": 2, "err": "some error"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_Debounce(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil)

	h := &Handler{
		Name:     "debounce-test",
		Debounce: "100ms",
		Run:      "echo 'processed {{.Message}}'",
		Respond:  "{{.Stdout | trim}}",
	}
	reg.AddHandlers([]Handler{*h})

	// Send 3 messages in rapid succession
	ctx := context.Background()
	d.handleMessage(ctx, &ws.Message{Data: []byte("msg1"), Metadata: ws.MessageMetadata{Direction: "received"}})
	d.handleMessage(ctx, &ws.Message{Data: []byte("msg2"), Metadata: ws.MessageMetadata{Direction: "received"}})
	d.handleMessage(ctx, &ws.Message{Data: []byte("msg3"), Metadata: ws.MessageMetadata{Direction: "received"}})

	// Wait for debounce period + buffers
	time.Sleep(250 * time.Millisecond)

	conn.mu.Lock()
	assert.Equal(t, "processed msg3", conn.lastWritten, "Should only process the last message")
	conn.mu.Unlock()
}

