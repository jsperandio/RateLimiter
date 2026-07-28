package entity

import (
	"fmt"
	"time"
)

type RateLimitKind string

const (
	RateLimitKindIP    RateLimitKind = "ip"
	RateLimitKindToken RateLimitKind = "token"
)

const (
	CounterWindow    = time.Second
	counterKeyPrefix = "ratelimit:count"
	blockKeyPrefix   = "ratelimit:block"
)

type RateLimit struct {
	Kind          RateLimitKind
	Identifier    string
	MaxRequests   int
	BlockDuration time.Duration
}

func (rt RateLimit) CounterKey(now time.Time) string {
	return fmt.Sprintf("%s:%s:%s:%d", counterKeyPrefix, rt.Kind, rt.Identifier, now.Unix())
}

func (rt RateLimit) BlockKey() string {
	return fmt.Sprintf("%s:%s:%s", blockKeyPrefix, rt.Kind, rt.Identifier)
}
