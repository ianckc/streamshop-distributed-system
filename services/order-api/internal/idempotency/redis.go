package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisBackend struct {
	client *redis.Client
}

func NewRedisBackend(client *redis.Client) *RedisBackend {
	return &RedisBackend{client: client}
}

func (b *RedisBackend) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return b.client.SetNX(ctx, key, value, ttl).Result()
}

func (b *RedisBackend) Get(ctx context.Context, key string) (string, error) {
	val, err := b.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	return val, err
}

func (b *RedisBackend) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return b.client.Set(ctx, key, value, ttl).Err()
}
