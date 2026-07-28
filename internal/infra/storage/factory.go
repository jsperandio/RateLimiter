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

func New(ctx context.Context, opt *StorageOptions) (entity.LimiterStorage, error) {
	if opt == nil {
		parsed, err := NewDefaultStorageOptions()
		if err != nil {
			return nil, err
		}

		opt = parsed
	}

	switch strings.ToLower(strings.TrimSpace(opt.Strategy)) {
	case StrategyMemory:
		return NewMemoryStorage(nil)

	case StrategyRedis:
		redisOpt, err := NewDefaultRedisStorageOptions()
		if err != nil {
			return nil, err
		}

		return NewRedisStorage(ctx, redisOpt)

	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownStrategy, opt.Strategy)
	}
}
