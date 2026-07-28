package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/jsperandio/RateLimiter/internal/entity"
)

const (
	StrategyMemory string = "memory"
	StrategyRedis  string = "redis"
)

type Config struct {
	Strategy string

	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func New(ctx context.Context, cfg Config) (entity.LimiterStorage, error) {
	stg := strings.ToLower(strings.TrimSpace(cfg.Strategy))

	switch stg {
	case StrategyMemory:
		return NewMemoryStorage(nil), nil

	case StrategyRedis:
		return NewRedisStorage(ctx, &RedisStorageOptions{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownStrategy, cfg.Strategy)
	}
}
