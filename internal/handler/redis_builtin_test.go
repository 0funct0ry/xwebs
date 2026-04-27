package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRedisManager struct {
	mock.Mock
}

func (m *MockRedisManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockRedisManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestRedisSetBuiltin(t *testing.T) {
	engine := template.New(false)
	builtin := &RedisSetBuiltin{}

	t.Run("basic set", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:   "test-key",
			Value: "test-value",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Set", mock.Anything, "test-key", "test-value", time.Duration(0)).Return(nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("template expressions", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:   "key-{{.MessageIndex}}",
			Value: "val-{{.Message}}",
		}
		tmplCtx := template.NewContext()
		tmplCtx.MessageIndex = 5
		tmplCtx.Message = "hello"

		mockRedis.On("Set", mock.Anything, "key-5", "val-hello", time.Duration(0)).Return(nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("with ttl", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:   "test-key",
			Value: "test-value",
			TTL:   "1m",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Set", mock.Anything, "test-key", "test-value", time.Minute).Return(nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("template ttl", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:   "test-key",
			Value: "test-value",
			TTL:   "{{index .Session \"Timeout\"}}s",
		}
		tmplCtx := template.NewContext()
		tmplCtx.Session = map[string]interface{}{
			"Timeout": "30",
		}

		mockRedis.On("Set", mock.Anything, "test-key", "test-value", 30*time.Second).Return(nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("redis error", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:   "test-key",
			Value: "test-value",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Set", mock.Anything, "test-key", "test-value", time.Duration(0)).Return(assert.AnError).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis set error")
		mockRedis.AssertExpectations(t)
	})

	t.Run("missing redis manager", func(t *testing.T) {
		dNoRedis := &Dispatcher{
			redisManager:   nil,
			templateEngine: engine,
		}
		a := &Action{
			Key:   "test-key",
			Value: "test-value",
		}
		tmplCtx := template.NewContext()

		err := builtin.Execute(context.Background(), dNoRedis, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis manager not initialized")
	})
}
