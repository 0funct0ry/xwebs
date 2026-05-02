package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// delayMockConn provides a minimal Connection stub for delay tests.
type delayMockConn struct {
	mockConn
}

func newDelayDispatcher(t *testing.T) (*Dispatcher, *delayMockConn) {
	t.Helper()
	conn := &delayMockConn{}
	registry := NewRegistry(ServerMode)
	engine := template.New(false)
	d := NewDispatcher(registry, conn, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, nil, "")
	return d, conn
}

func TestDelayBuiltin_Validate(t *testing.T) {
	b := &DelayBuiltin{}

	t.Run("Missing duration returns error", func(t *testing.T) {
		err := b.Validate(Action{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'duration'")
	})

	t.Run("Invalid static duration returns error", func(t *testing.T) {
		err := b.Validate(Action{Duration: "notaduration"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid duration")
	})

	t.Run("Valid static duration passes", func(t *testing.T) {
		err := b.Validate(Action{Duration: "100ms"})
		assert.NoError(t, err)
	})

	t.Run("Template expression passes without evaluation", func(t *testing.T) {
		err := b.Validate(Action{Duration: `{{"50ms"}}`})
		assert.NoError(t, err)
	})

	t.Run("Valid max passes", func(t *testing.T) {
		err := b.Validate(Action{Duration: "5s", Max: "1s"})
		assert.NoError(t, err)
	})

	t.Run("Invalid static max returns error", func(t *testing.T) {
		err := b.Validate(Action{Duration: "5s", Max: "notaduration"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid max")
	})

	t.Run("Template max passes without evaluation", func(t *testing.T) {
		err := b.Validate(Action{Duration: "5s", Max: `{{"1s"}}`})
		assert.NoError(t, err)
	})
}

func TestDelayBuiltin_Scope(t *testing.T) {
	b := &DelayBuiltin{}
	assert.Equal(t, Shared, b.Scope(), "delay builtin must be Shared so it works in both client and server modes")
}

func TestDelayBuiltin_StaticDuration(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	action := &Action{Duration: "50ms"}

	start := time.Now()
	err := b.Execute(context.Background(), d, action, tmplCtx)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, 45*time.Millisecond, "should sleep at least ~50ms")
}

func TestDelayBuiltin_TemplateDuration(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	// Template expression that resolves to "60ms"
	action := &Action{Duration: `{{"60ms"}}`}

	start := time.Now()
	err := b.Execute(context.Background(), d, action, tmplCtx)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond, "template expression should produce a valid duration")
}

func TestDelayBuiltin_MaxCap(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	// duration is 10s but max caps it at 80ms
	action := &Action{Duration: "10s", Max: "80ms"}

	start := time.Now()
	err := b.Execute(context.Background(), d, action, tmplCtx)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	// Should have been capped — well under 1 second
	assert.Less(t, elapsed, 500*time.Millisecond, "duration should have been capped by max")
	assert.GreaterOrEqual(t, elapsed, 70*time.Millisecond, "should still sleep for at least ~max duration")
}

func TestDelayBuiltin_MaxCapTemplate(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	action := &Action{Duration: "5s", Max: `{{"100ms"}}`}

	start := time.Now()
	err := b.Execute(context.Background(), d, action, tmplCtx)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed, 500*time.Millisecond)
}

func TestDelayBuiltin_InvalidRuntimeDuration(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	// Template resolves to a non-duration string at runtime
	action := &Action{Duration: `{{"not-a-duration"}}`}

	err := b.Execute(context.Background(), d, action, tmplCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration")
}

func TestDelayBuiltin_ContextCancellation(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	// Use a very long duration to ensure context cancellation triggers first
	action := &Action{Duration: "30s"}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a brief moment
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := b.Execute(ctx, d, action, tmplCtx)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, 500*time.Millisecond, "should have returned early when context was cancelled")
}

func TestDelayBuiltin_ZeroDuration(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	action := &Action{Duration: "0s"}
	err := b.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err, "zero duration should be allowed")
}

func TestDelayBuiltin_MaxDoesNotExtend(t *testing.T) {
	d, _ := newDelayDispatcher(t)
	b := &DelayBuiltin{}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, nil)

	// max is larger than duration — duration should be unchanged
	action := &Action{Duration: "50ms", Max: "10s"}

	start := time.Now()
	err := b.Execute(context.Background(), d, action, tmplCtx)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed, 500*time.Millisecond, "max larger than duration should not extend sleep")
	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond)
}

func TestDelayBuiltin_RegisteredAndShared(t *testing.T) {
	h, ok := GetBuiltin("delay")
	require.True(t, ok, "delay builtin must be registered")
	assert.Equal(t, "delay", h.Name())
	assert.Equal(t, Shared, h.Scope())

	// Allowed in both modes
	allowedServer, _, _ := IsBuiltinAllowed("delay", ServerMode)
	allowedClient, _, _ := IsBuiltinAllowed("delay", ClientMode)
	assert.True(t, allowedServer, "delay must be allowed in server mode")
	assert.True(t, allowedClient, "delay must be allowed in client mode")
}
