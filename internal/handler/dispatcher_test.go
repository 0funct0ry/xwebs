package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcher_ExecutePipeline(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

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

	err := d.Execute(context.Background(), h, msg, nil)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"result": "HELLO world"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_ExecutePipelineFailure(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

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

	err := d.Execute(context.Background(), h, msg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline step \"step1\" failed: exit code 2")

	conn.mu.Lock()
	assert.Empty(t, conn.lastWritten) // Respond should not have run
	conn.mu.Unlock()
}

func TestDispatcher_ExecutePipelineIgnoreError(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

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

	err := d.Execute(context.Background(), h, msg, nil)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"code": 2, "result": "rescued"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_GlobalVariables(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	vars := map[string]interface{}{"tmp": "/tmp/test", "version": 1}
	d := NewDispatcher(reg, conn, engine, false, vars, nil, false, nil, nil, nil, nil, nil, "")

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

	err := d.Execute(context.Background(), h, msg, nil)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"status": "ok", "output": "using /tmp/test v1"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_Shorthands(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

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

	err := d.Execute(context.Background(), h, msg, nil)
	require.NoError(t, err)

	conn.mu.Lock()
	assert.JSONEq(t, `{"status": "ok", "output": "processed data"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_RespondContext(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

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

	err := d.Execute(context.Background(), h, msg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit code 2")

	conn.mu.Lock()
	assert.JSONEq(t, `{"code": 2, "err": "some error"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_Debounce(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

	h := &Handler{
		Name:     "debounce-test",
		Match:    Matcher{Pattern: "*"},
		Debounce: "100ms",
		Run:      "echo 'processed {{.Message}}'",
		Respond:  "{{.Stdout | trim}}",
	}
	_ = reg.AddHandlers([]Handler{*h})

	// Send 3 messages in rapid succession, but with tiny delays to ensure ordering
	ctx := context.Background()
	d.handleMessage(ctx, &ws.Message{Data: []byte("msg1"), Metadata: ws.MessageMetadata{Direction: "received"}})
	time.Sleep(10 * time.Millisecond)
	d.handleMessage(ctx, &ws.Message{Data: []byte("msg2"), Metadata: ws.MessageMetadata{Direction: "received"}})
	time.Sleep(10 * time.Millisecond)
	d.handleMessage(ctx, &ws.Message{Data: []byte("msg3"), Metadata: ws.MessageMetadata{Direction: "received"}})

	// Wait for debounce period + buffers
	time.Sleep(250 * time.Millisecond)

	conn.mu.Lock()
	assert.Equal(t, "processed msg3", conn.lastWritten, "Should only process the last message")
	conn.mu.Unlock()
}

func TestDispatcher_ExclusiveShortCircuit(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

	var mu sync.Mutex
	executed := make([]string, 0)

	// Mock execute function to track execution
	h1 := Handler{
		Name:      "h1",
		Priority:  10,
		Exclusive: false,
		Match:     Matcher{Type: "text", Pattern: "ping"},
		Run:       "echo 'h1'",
	}
	h2 := Handler{
		Name:      "h2",
		Priority:  5,
		Exclusive: true,
		Match:     Matcher{Type: "text", Pattern: "ping"},
		Run:       "echo 'h2'",
	}
	h3 := Handler{
		Name:      "h3",
		Priority:  1,
		Exclusive: false,
		Match:     Matcher{Type: "text", Pattern: "ping"},
		Run:       "echo 'h3'",
	}

	_ = reg.AddHandlers([]Handler{h1, h2, h3})

	// Wrap Dispatcher.Log to track executions
	d.Log = func(f string, a ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		msg := fmt.Sprintf(f, a...)
		executed = append(executed, strings.TrimSpace(msg))
	}

	msg := &ws.Message{
		Data: []byte("ping"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
		},
	}

	// This is async by default in Dispatcher.Start,
	// but here we can call handleMessage directly (it's internal to dispatcher.go but visible in package)
	d.handleMessage(context.Background(), msg)

	// Wait for goroutines started in handleMessage
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	assert.Contains(t, executed, "h1")
	assert.Contains(t, executed, "h2")
	assert.NotContains(t, executed, "h3")
}

func TestDispatcher_ExclusivePriority(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

	var mu sync.Mutex
	executed := make([]string, 0)

	// Highest priority is exclusive
	h1 := Handler{
		Name:      "exclusive-priority",
		Priority:  100,
		Exclusive: true,
		Match:     Matcher{Type: "text", Pattern: "ping"},
		Run:       "echo 'exclusive'",
	}
	h2 := Handler{
		Name:      "lower-priority",
		Priority:  50,
		Exclusive: false,
		Match:     Matcher{Type: "text", Pattern: "ping"},
		Run:       "echo 'lower'",
	}

	_ = reg.AddHandlers([]Handler{h1, h2})

	d.Log = func(f string, a ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		msg := fmt.Sprintf(f, a...)
		executed = append(executed, strings.TrimSpace(msg))
	}

	msg := &ws.Message{Data: []byte("ping"), Metadata: ws.MessageMetadata{Direction: "received"}}
	d.handleMessage(context.Background(), msg)
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	assert.Contains(t, executed, "exclusive")
	assert.NotContains(t, executed, "lower")
}
