package handler

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcher_ExecuteRetryLinear(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, "")

	// Create a temp file to track attempts
	tmpFile := "/tmp/xwebs_retry_test_linear"
	os.Remove(tmpFile)
	defer os.Remove(tmpFile)

	// This command will fail the first 2 times, and succeed on the 3rd.
	// We use the file to store the attempt count.
	cmd := fmt.Sprintf(`count=$(cat %s 2>/dev/null || echo 0); count=$((count+1)); echo $count > %s; if [ $count -lt 3 ]; then exit 1; else echo "success"; fi`, tmpFile, tmpFile)

	h := &Handler{
		Name: "retry-linear",
		Run:  cmd,
		Retry: &RetryConfig{
			Count:    3,
			Backoff:  "linear",
			Interval: "100ms",
		},
		Respond: `{"result": "{{.Stdout | trim}}"}`,
	}

	msg := &ws.Message{
		Data: []byte("test"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	start := time.Now()
	err := d.Execute(context.Background(), h, msg, nil)
	duration := time.Since(start)

	require.NoError(t, err)
	// Attempts 1 and 2 fail.
	// Wait after 1: 100ms * 1 = 100ms
	// Wait after 2: 100ms * 2 = 200ms
	// Total wait: 300ms
	assert.GreaterOrEqual(t, duration, 300*time.Millisecond)

	conn.mu.Lock()
	assert.JSONEq(t, `{"result": "success"}`, conn.lastWritten)
	conn.mu.Unlock()
}

func TestDispatcher_ExecuteRetryExponential(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, "")

	h := &Handler{
		Name: "retry-exponential",
		Run:  "exit 1",
		Retry: &RetryConfig{
			Count:       2,
			Backoff:     "exponential",
			Interval:    "100ms",
			MaxInterval: "150ms",
		},
	}

	msg := &ws.Message{
		Data: []byte("test"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	start := time.Now()
	err := d.Execute(context.Background(), h, msg, nil)
	duration := time.Since(start)

	require.Error(t, err)
	// attempt 1 fails: backoff 100ms * 2^0 = 100ms
	// attempt 2 fails: backoff 100ms * 2^1 = 200ms -> capped at 150ms
	// Total wait approx 250ms (plus execution time)
	assert.GreaterOrEqual(t, duration, 250*time.Millisecond)
}

func TestDispatcher_ExecuteRetryPipeline(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, "")

	tmpFile := "/tmp/xwebs_retry_test_pipeline"
	os.Remove(tmpFile)
	defer os.Remove(tmpFile)

	cmd := fmt.Sprintf(`count=$(cat %s 2>/dev/null || echo 0); count=$((count+1)); echo $count > %s; if [ $count -lt 2 ]; then exit 1; else echo "recovered"; fi`, tmpFile, tmpFile)

	h := &Handler{
		Name: "retry-pipeline",
		Pipeline: []PipelineStep{
			{Run: cmd, As: "step1"},
		},
		Retry: &RetryConfig{
			Count:    2,
			Backoff:  "linear",
			Interval: "50ms",
		},
		Respond: `{"status": "{{.Steps.step1.Stdout | trim}}"}`,
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
	assert.JSONEq(t, `{"status": "recovered"}`, conn.lastWritten)
	conn.mu.Unlock()
}
