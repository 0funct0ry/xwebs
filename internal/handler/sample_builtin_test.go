package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSampleBuiltin(t *testing.T) {
	r := NewRegistry(ServerMode)
	engine := template.New(false)
	d := &Dispatcher{
		registry:       r,
		templateEngine: engine,
	}

	builtin := &SampleBuiltin{}

	t.Run("basic sampling rate 2", func(t *testing.T) {
		r.ResetSample("h1:sample")
		a := &Action{HandlerName: "h1", Command: "sample", Rate: "2"}
		tmplCtx := &template.TemplateContext{}

		// 1st message -> drop
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Equal(t, ErrDrop, err)

		// 2nd message -> pass
		err = builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		// 3rd message -> drop
		err = builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Equal(t, ErrDrop, err)

		// 4th message -> pass
		err = builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
	})

	t.Run("sampling rate 3", func(t *testing.T) {
		r.ResetSample("h2:sample")
		a := &Action{HandlerName: "h2", Command: "sample", Rate: "3"}
		tmplCtx := &template.TemplateContext{}

		// 1st -> drop
		assert.Equal(t, ErrDrop, builtin.Execute(context.Background(), d, a, tmplCtx))
		// 2nd -> drop
		assert.Equal(t, ErrDrop, builtin.Execute(context.Background(), d, a, tmplCtx))
		// 3rd -> pass
		assert.NoError(t, builtin.Execute(context.Background(), d, a, tmplCtx))
		// 4th -> drop
		assert.Equal(t, ErrDrop, builtin.Execute(context.Background(), d, a, tmplCtx))
	})

	t.Run("template rate", func(t *testing.T) {
		r.ResetSample("h3:sample")
		a := &Action{HandlerName: "h3", Command: "sample", Rate: "{{.Vars.N}}"}
		tmplCtx := &template.TemplateContext{
			Vars: map[string]interface{}{"N": 2},
		}

		// 1st -> drop
		assert.Equal(t, ErrDrop, builtin.Execute(context.Background(), d, a, tmplCtx))
		// 2nd -> pass
		assert.NoError(t, builtin.Execute(context.Background(), d, a, tmplCtx))
	})

	t.Run("invalid rate", func(t *testing.T) {
		a := &Action{HandlerName: "h4", Command: "sample", Rate: "abc"}
		tmplCtx := &template.TemplateContext{}
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid rate")
	})

	t.Run("zero rate", func(t *testing.T) {
		a := &Action{HandlerName: "h5", Command: "sample", Rate: "0"}
		tmplCtx := &template.TemplateContext{}
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be positive")
	})
}

func TestDispatcher_Sample(t *testing.T) {
	r := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(r, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")

	h := &Handler{
		Name:  "sampler",
		Match: Matcher{Pattern: "*"},
		Actions: []Action{
			{Type: "builtin", Command: "sample", Rate: "2"},
			{Type: "send", Message: "passed"},
		},
	}
	require.NoError(t, r.Add(*h))

	ctx := context.Background()

	// 1st message -> sampled out
	err := d.Execute(ctx, h, &ws.Message{Data: []byte("m1")}, nil)
	assert.Equal(t, ErrDrop, err)
	conn.mu.Lock()
	assert.Len(t, conn.messages, 0)
	conn.mu.Unlock()

	// 2nd message -> passed
	err = d.Execute(ctx, h, &ws.Message{Data: []byte("m2")}, nil)
	assert.NoError(t, err)
	conn.mu.Lock()
	assert.Len(t, conn.messages, 1)
	assert.Equal(t, "passed", string(conn.messages[0].Data))
	conn.mu.Unlock()

	// 3rd message -> sampled out
	err = d.Execute(ctx, h, &ws.Message{Data: []byte("m3")}, nil)
	assert.Equal(t, ErrDrop, err)
	conn.mu.Lock()
	assert.Len(t, conn.messages, 1)
	conn.mu.Unlock()

	// 4th message -> passed
	err = d.Execute(ctx, h, &ws.Message{Data: []byte("m4")}, nil)
	assert.NoError(t, err)
	conn.mu.Lock()
	assert.Len(t, conn.messages, 2)
	assert.Equal(t, "passed", string(conn.messages[1].Data))
	conn.mu.Unlock()
}
