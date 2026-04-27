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

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return &redisManager{client: client}, nil
}

func (m *redisManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("redis manager not initialized (check --redis-url)")
	}
	return m.client.Set(ctx, key, value, ttl).Err()
}

func (m *redisManager) Close() error {
	if m == nil || m.client == nil {
		return nil
	}
	return m.client.Close()
}
