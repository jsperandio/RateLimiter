package entity

import (
	"context"
	"time"
)

type LimiterStorage interface {
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
	IsBlocked(ctx context.Context, key string) (bool, error)
	Block(ctx context.Context, key string, duration time.Duration) error
}
