package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(ctx context.Context, opt *RedisStorageOptions) (*RedisStorage, error) {
	if opt == nil {
		parsed, err := NewDefaultRedisStorageOptions()
		if err != nil {
			return nil, err
		}

		opt = parsed
	}

	client := redis.NewClient(&redis.Options{
		Addr:     opt.Addr,
		Password: opt.Password,
		DB:       opt.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()

		return nil, fmt.Errorf("connecting to redis at %s: %w", opt.Addr, err)
	}

	return &RedisStorage{
		client: client,
	}, nil
}

func (rs *RedisStorage) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	var incr *redis.IntCmd

	_, err := rs.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		incr = pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, window)

		return nil
	})
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

func (rs *RedisStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	count, err := rs.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (rs *RedisStorage) Block(ctx context.Context, key string, duration time.Duration) error {
	return rs.client.Set(ctx, key, 1, duration).Err()
}

func (rs *RedisStorage) Close() error {
	return rs.client.Close()
}
