package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/redis/go-redis/v9"
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

func (m *MockRedisManager) Get(ctx context.Context, key string) (interface{}, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func (m *MockRedisManager) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockRedisManager) Publish(ctx context.Context, channel string, message interface{}) error {
	args := m.Called(ctx, channel, message)
	return args.Error(0)
}

func (m *MockRedisManager) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	args := m.Called(ctx, channels)
	return args.Get(0).(*redis.PubSub)
}

func (m *MockRedisManager) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	args := m.Called(ctx, patterns)
	return args.Get(0).(*redis.PubSub)
}

func (m *MockRedisManager) LPush(ctx context.Context, key string, values ...interface{}) error {
	args := m.Called(ctx, key, values)
	return args.Error(0)
}

func (m *MockRedisManager) RPop(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
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

func TestRedisGetBuiltin(t *testing.T) {
	engine := template.New(false)
	builtin := &RedisGetBuiltin{}

	t.Run("basic get", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key: "test-key",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Get", mock.Anything, "test-key").Return("test-value", nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		assert.Equal(t, "test-value", tmplCtx.RedisValue)
		mockRedis.AssertExpectations(t)
	})

	t.Run("get missing with default", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:     "test-key",
			Default: "my-default",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Get", mock.Anything, "test-key").Return(nil, nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		assert.Equal(t, "my-default", tmplCtx.RedisValue)
		mockRedis.AssertExpectations(t)
	})

	t.Run("get missing without default", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key: "test-key",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Get", mock.Anything, "test-key").Return(nil, nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		assert.Equal(t, "", tmplCtx.RedisValue)
		mockRedis.AssertExpectations(t)
	})

	t.Run("template expressions in key and default", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:     "key-{{.MessageIndex}}",
			Default: "default-for-{{.Message}}",
		}
		tmplCtx := template.NewContext()
		tmplCtx.MessageIndex = 10
		tmplCtx.Message = "foo"

		mockRedis.On("Get", mock.Anything, "key-10").Return(nil, nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		assert.Equal(t, "default-for-foo", tmplCtx.RedisValue)
		mockRedis.AssertExpectations(t)
	})

	t.Run("redis error", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key: "test-key",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Get", mock.Anything, "test-key").Return(nil, assert.AnError).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis get error")
		mockRedis.AssertExpectations(t)
	})

	t.Run("missing redis manager", func(t *testing.T) {
		dNoRedis := &Dispatcher{
			redisManager:   nil,
			templateEngine: engine,
		}
		a := &Action{
			Key: "test-key",
		}
		tmplCtx := template.NewContext()

		err := builtin.Execute(context.Background(), dNoRedis, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis manager not initialized")
	})
}

func TestRedisDelBuiltin(t *testing.T) {
	engine := template.New(false)
	builtin := &RedisDelBuiltin{}

	t.Run("basic del", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key: "test-key",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Del", mock.Anything, "test-key").Return(nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("template expressions in key", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key: "key-{{.MessageIndex}}",
		}
		tmplCtx := template.NewContext()
		tmplCtx.MessageIndex = 10

		mockRedis.On("Del", mock.Anything, "key-10").Return(nil).Once()

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
			Key: "test-key",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Del", mock.Anything, "test-key").Return(assert.AnError).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis del error")
		mockRedis.AssertExpectations(t)
	})

	t.Run("missing redis manager", func(t *testing.T) {
		dNoRedis := &Dispatcher{
			redisManager:   nil,
			templateEngine: engine,
		}
		a := &Action{
			Key: "test-key",
		}
		tmplCtx := template.NewContext()

		err := builtin.Execute(context.Background(), dNoRedis, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis manager not initialized")
	})
}

func TestRedisPublishBuiltin(t *testing.T) {
	engine := template.New(false)
	builtin := &RedisPublishBuiltin{}

	t.Run("basic publish", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Channel: "test-channel",
			Message: "test-message",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Publish", mock.Anything, "test-channel", "test-message").Return(nil).Once()

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
			Channel: "chan-{{.MessageIndex}}",
			Message: "msg-{{.Message}}",
		}
		tmplCtx := template.NewContext()
		tmplCtx.MessageIndex = 1
		tmplCtx.Message = "hello"

		mockRedis.On("Publish", mock.Anything, "chan-1", "msg-hello").Return(nil).Once()

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
			Channel: "test-channel",
			Message: "test-message",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("Publish", mock.Anything, "test-channel", "test-message").Return(assert.AnError).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis publish error")
		mockRedis.AssertExpectations(t)
	})

	t.Run("missing redis manager", func(t *testing.T) {
		dNoRedis := &Dispatcher{
			redisManager:   nil,
			templateEngine: engine,
		}
		a := &Action{
			Channel: "test-channel",
			Message: "test-message",
		}
		tmplCtx := template.NewContext()

		err := builtin.Execute(context.Background(), dNoRedis, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis manager not initialized")
	})
}

func TestRedisLPushBuiltin(t *testing.T) {
	engine := template.New(false)
	builtin := &RedisLPushBuiltin{}

	t.Run("basic lpush", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:   "test-list",
			Value: "item1",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("LPush", mock.Anything, "test-list", []interface{}{"item1"}).Return(nil).Once()

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
			Key:   "list-{{.MessageIndex}}",
			Value: "val-{{.Message}}",
		}
		tmplCtx := template.NewContext()
		tmplCtx.MessageIndex = 1
		tmplCtx.Message = "hello"

		mockRedis.On("LPush", mock.Anything, "list-1", []interface{}{"val-hello"}).Return(nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})
}

func TestRedisRPopBuiltin(t *testing.T) {
	engine := template.New(false)
	builtin := &RedisRPopBuiltin{}

	t.Run("basic rpop", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key: "test-list",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("RPop", mock.Anything, "test-list").Return("item1", nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		assert.Equal(t, "item1", tmplCtx.RedisValue)
		mockRedis.AssertExpectations(t)
	})

	t.Run("rpop empty with default", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:     "test-list",
			Default: "empty-default",
		}
		tmplCtx := template.NewContext()

		mockRedis.On("RPop", mock.Anything, "test-list").Return("", nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		assert.Equal(t, "empty-default", tmplCtx.RedisValue)
		mockRedis.AssertExpectations(t)
	})

	t.Run("template expressions in key and default", func(t *testing.T) {
		mockRedis := new(MockRedisManager)
		d := &Dispatcher{
			redisManager:   mockRedis,
			templateEngine: engine,
		}
		a := &Action{
			Key:     "list-{{.MessageIndex}}",
			Default: "none-for-{{.Message}}",
		}
		tmplCtx := template.NewContext()
		tmplCtx.MessageIndex = 10
		tmplCtx.Message = "foo"

		mockRedis.On("RPop", mock.Anything, "list-10").Return("", nil).Once()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		assert.Equal(t, "none-for-foo", tmplCtx.RedisValue)
		mockRedis.AssertExpectations(t)
	})
}
