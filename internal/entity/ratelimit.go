package entity

import (
	"strconv"
	"strings"
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
	return strings.Join([]string{
		counterKeyPrefix, string(rt.Kind),
		rt.Identifier, strconv.FormatInt(now.Unix(), 10),
	}, ":")
}

func (rt RateLimit) BlockKey() string {
	return strings.Join([]string{
		blockKeyPrefix, string(rt.Kind),
		rt.Identifier,
	}, ":")
}
