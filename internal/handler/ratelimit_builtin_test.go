package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
)

type ratelimitMockConn struct {
	id       string
	mockConn // Embed to avoid implementing everything
}

func (m *ratelimitMockConn) GetID() string { return m.id }

func TestRateLimitBuiltin(t *testing.T) {
	engine := template.New(false)

	conn1 := &ratelimitMockConn{id: "client-1"}
	conn2 := &ratelimitMockConn{id: "client-2"}

	builtin := &RateLimitBuiltin{}

	t.Run("Client Scope (Default)", func(t *testing.T) {
		registry := NewRegistry(ServerMode)
		d1 := NewDispatcher(registry, conn1, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, "")
		d2 := NewDispatcher(registry, conn2, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, "")

		action := &Action{
			HandlerName: "test-handler",
			Rate:        "1/s",
			Scope:       "client",
		}
		tmplCtx1 := template.NewContext()
		d1.populateTemplateContext(tmplCtx1, nil)

		tmplCtx2 := template.NewContext()
		d2.populateTemplateContext(tmplCtx2, nil)

		// Client 1: First request allowed
		err := builtin.Execute(context.Background(), d1, action, tmplCtx1)
		assert.NoError(t, err)

		// Client 1: Second request rejected
		err = builtin.Execute(context.Background(), d1, action, tmplCtx1)
		assert.ErrorIs(t, err, ErrLimitExceeded)

		// Client 2: First request allowed (independent bucket)
		err = builtin.Execute(context.Background(), d2, action, tmplCtx2)
		assert.NoError(t, err)
	})

	t.Run("Global Scope", func(t *testing.T) {
		registry := NewRegistry(ServerMode)
		d1 := NewDispatcher(registry, conn1, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, "")
		d2 := NewDispatcher(registry, conn2, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, "")

		action := &Action{
			HandlerName: "test-handler",
			Rate:        "1/s",
			Scope:       "global",
		}
		tmplCtx1 := template.NewContext()
		d1.populateTemplateContext(tmplCtx1, nil)

		tmplCtx2 := template.NewContext()
		d2.populateTemplateContext(tmplCtx2, nil)

		// Client 1: First request allowed
		err := builtin.Execute(context.Background(), d1, action, tmplCtx1)
		assert.NoError(t, err)

		// Client 2: Second request rejected (shared bucket)
		err = builtin.Execute(context.Background(), d2, action, tmplCtx2)
		assert.ErrorIs(t, err, ErrLimitExceeded)
	})

	t.Run("Handler Scope", func(t *testing.T) {
		registry := NewRegistry(ServerMode)
		d1 := NewDispatcher(registry, conn1, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, "")

		actionA := &Action{
			HandlerName: "handler-A",
			Rate:        "1/s",
			Scope:       "handler",
		}
		actionB := &Action{
			HandlerName: "handler-B",
			Rate:        "1/s",
			Scope:       "handler",
		}
		tmplCtx := template.NewContext()
		d1.populateTemplateContext(tmplCtx, nil)

		// Handler A: Allowed
		err := builtin.Execute(context.Background(), d1, actionA, tmplCtx)
		assert.NoError(t, err)

		// Handler A: Rejected
		err = builtin.Execute(context.Background(), d1, actionA, tmplCtx)
		assert.ErrorIs(t, err, ErrLimitExceeded)

		// Handler B: Allowed (independent bucket)
		err = builtin.Execute(context.Background(), d1, actionB, tmplCtx)
		assert.NoError(t, err)
	})

	t.Run("Dynamic Rate Template", func(t *testing.T) {
		registry := NewRegistry(ServerMode)
		kv := &mockKV{data: make(map[string]interface{})}
		kv.data["my_rate"] = "10/s"

		d := NewDispatcher(registry, conn1, engine, true, nil, nil, false, nil, nil, nil, kv, nil, nil, "")
		action := &Action{
			HandlerName: "dyn-handler",
			Rate:        `{{kv "my_rate"}}`,
			Scope:       "handler",
		}
		tmplCtx := template.NewContext()
		d.populateTemplateContext(tmplCtx, nil)

		// Execute 10 times (should all pass)
		for i := 0; i < 10; i++ {
			err := builtin.Execute(context.Background(), d, action, tmplCtx)
			assert.NoError(t, err, "failed at iteration %d", i)
		}

		// 11th should fail
		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		assert.ErrorIs(t, err, ErrLimitExceeded)

		// Change rate in KV
		kv.data["my_rate"] = "100/s"
		// Refresh context
		d.populateTemplateContext(tmplCtx, nil)
		// Wait a bit for refill
		time.Sleep(200 * time.Millisecond)
		// Now it should allow again because GetScopedLimiter detects rate change and updates burst/rate
		err = builtin.Execute(context.Background(), d, action, tmplCtx)
		assert.NoError(t, err)
	})
}

type mockKV struct {
	data map[string]interface{}
}

func (m *mockKV) ListKV() map[string]interface{} { return m.data }
func (m *mockKV) GetKV(key string) (interface{}, bool) {
	v, ok := m.data[key]
	return v, ok
}
func (m *mockKV) SetKV(key string, val interface{}, ttl time.Duration) { m.data[key] = val }
func (m *mockKV) DeleteKV(key string)                                  { delete(m.data, key) }
