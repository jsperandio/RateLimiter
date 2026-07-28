package entity

import (
	"strings"
	"time"
)

type Limits struct {
	IPMaxRequests      int
	IPBlockDuration    time.Duration
	TokenMaxRequests   int
	TokenBlockDuration time.Duration
}

func (l Limits) Validate() error {
	if l.IPMaxRequests <= 0 || l.TokenMaxRequests <= 0 {
		return ErrInvalidMaxRequests
	}

	if l.IPBlockDuration <= 0 || l.TokenBlockDuration <= 0 {
		return ErrInvalidBlockDuration
	}

	return nil
}

func ResolveLimit(ip, token string, limits Limits) (RateLimit, error) {
	if token = strings.TrimSpace(token); token != "" {
		return RateLimit{
			Kind:          RateLimitKindToken,
			Identifier:    token,
			MaxRequests:   limits.TokenMaxRequests,
			BlockDuration: limits.TokenBlockDuration,
		}, nil
	}

	if ip = strings.TrimSpace(ip); ip == "" {
		return RateLimit{}, ErrNoIdentifier
	}

	return RateLimit{
		Kind:          RateLimitKindIP,
		Identifier:    ip,
		MaxRequests:   limits.IPMaxRequests,
		BlockDuration: limits.IPBlockDuration,
	}, nil
}
