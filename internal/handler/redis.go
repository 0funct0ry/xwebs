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

func (m *redisManager) Close() error {
	if m == nil || m.client == nil {
		return nil
	}
	return m.client.Close()
}
