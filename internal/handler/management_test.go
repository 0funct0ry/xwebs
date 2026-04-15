package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Management(t *testing.T) {
	r := NewRegistry()
	h := Handler{Name: "test", Match: Matcher{Pattern: "ping"}}
	err := r.Add(h)
	require.NoError(t, err)

	// Verify initial stats
	matches, totalLatency, errorsCount, ok := r.GetStats("test")
	assert.True(t, ok)
	assert.Equal(t, uint64(0), matches)
	assert.Equal(t, time.Duration(0), totalLatency)
	assert.Equal(t, uint64(0), errorsCount)

	// Test Match records hit
	msg := &ws.Message{Data: []byte("ping")}
	matchesResult, err := r.Match(msg, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, matchesResult, 1)

	matches, _, _, _ = r.GetStats("test")
	assert.Equal(t, uint64(1), matches)

	// Test Disable
	err = r.DisableHandler("test")
	assert.NoError(t, err)
	assert.True(t, r.IsDisabled("test"))

	matchesResult, err = r.Match(msg, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, matchesResult, 0, "Disabled handler should not match")

	// Test Enable
	err = r.EnableHandler("test")
	assert.NoError(t, err)
	assert.False(t, r.IsDisabled("test"))

	matchesResult, err = r.Match(msg, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, matchesResult, 1)

	// Test RecordExecution
	r.RecordExecution("test", 100*time.Millisecond, nil)
	r.RecordExecution("test", 200*time.Millisecond, errors.New("fail"))

	matches, totalLatency, errorsCount, ok = r.GetStats("test")
	assert.True(t, ok)
	assert.Equal(t, uint64(2), matches) // 2 from Match calls above
	assert.Equal(t, 300*time.Millisecond, totalLatency)
	assert.Equal(t, uint64(1), errorsCount)
}

func TestDispatcher_ExecuteRecordsStats(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Add(Handler{Name: "exec-test", Match: Matcher{Pattern: ".*", Type: "regex"}, Run: "echo hello"}))

	tmplEngine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(r, conn, tmplEngine, false, nil, nil, false, nil, nil, nil)

	msg := &ws.Message{Data: []byte("hi")}
	h, _ := r.GetHandler("exec-test")

	err := d.Execute(context.Background(), &h, msg)
	assert.NoError(t, err)

	matches, totalLatency, errorsCount, ok := r.GetStats("exec-test")
	assert.True(t, ok)
	assert.Equal(t, uint64(0), matches, "Execute doesn't call Match internally, Registry.Match increments it")
	assert.True(t, totalLatency > 0)
	assert.Equal(t, uint64(0), errorsCount)
}
