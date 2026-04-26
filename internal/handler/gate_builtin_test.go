package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateBuiltin(t *testing.T) {
	kv := &mockKVManager{
		store: map[string]interface{}{
			"enabled": "true",
			"count":   "5",
		},
	}

	engine := template.New(false)
	conn := &mockConn{}

	d := &Dispatcher{
		kvManager:      kv,
		templateEngine: engine,
		conn:           conn,
	}

	builtin := &GateBuiltin{}

	t.Run("GateOpen", func(t *testing.T) {
		a := &Action{
			Key:    "enabled",
			Expect: "true",
		}
		tmplCtx := template.NewContext()
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
	})

	t.Run("GateClosed", func(t *testing.T) {
		a := &Action{
			Key:    "enabled",
			Expect: "false",
		}
		tmplCtx := template.NewContext()
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Equal(t, ErrDrop, err)
	})

	t.Run("GateClosedWithResponse", func(t *testing.T) {
		mConn := &mockConn{}
		d.conn = mConn

		a := &Action{
			Key:      "enabled",
			Expect:   "false",
			OnClosed: "Access Denied",
		}
		tmplCtx := template.NewContext()
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Equal(t, ErrDrop, err)

		mConn.mu.Lock()
		require.Len(t, mConn.messages, 1)
		assert.Equal(t, "Access Denied", string(mConn.messages[0].Data))
		mConn.mu.Unlock()
	})

	t.Run("TemplateExpressions", func(t *testing.T) {
		a := &Action{
			Key:    "{{ \"enabled\" }}",
			Expect: "{{ \"true\" }}",
		}
		tmplCtx := template.NewContext()
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		a := &Action{
			Key:    "missing",
			Expect: "something",
		}
		tmplCtx := template.NewContext()
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Equal(t, ErrDrop, err)
	})
}
