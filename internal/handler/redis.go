package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisManager struct {
	client *redis.Client
}

// NewRedisManager creates a new Redis manager from a connection URL.
func NewRedisManager(url string) (RedisManager, error) {
	if url == "" {
		return nil, nil
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parsing redis url: %w", err)
	}

	client := redis.NewClient(opts)

	return &redisManager{client: client}, nil
}

func (m *redisManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	return m.client.Set(ctx, key, value, ttl).Err()
}

func (m *redisManager) Get(ctx context.Context, key string) (interface{}, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	val, err := m.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (m *redisManager) Del(ctx context.Context, key string) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	return m.client.Del(ctx, key).Err()
}
 
func (m *redisManager) Publish(ctx context.Context, channel string, message interface{}) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	return m.client.Publish(ctx, channel, message).Err()
}

func (m *redisManager) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	if m == nil || m.client == nil {
		return nil
	}
	return m.client.Subscribe(ctx, channels...)
}

func (m *redisManager) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	if m == nil || m.client == nil {
		return nil
	}
	return m.client.PSubscribe(ctx, patterns...)
}

func (m *redisManager) LPush(ctx context.Context, key string, values ...interface{}) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	return m.client.LPush(ctx, key, values...).Err()
}

func (m *redisManager) RPop(ctx context.Context, key string) (string, error) {
	if m == nil || m.client == nil {
		return "", fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	val, err := m.client.RPop(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (m *redisManager) Incr(ctx context.Context, key string, by int64) (int64, error) {
	if m == nil || m.client == nil {
		return 0, fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	if by == 0 || by == 1 {
		return m.client.Incr(ctx, key).Result()
	}
	return m.client.IncrBy(ctx, key, by).Result()
}

func (m *redisManager) Close() error {
	if m == nil || m.client == nil {
		return nil
	}
	return m.client.Close()
}
