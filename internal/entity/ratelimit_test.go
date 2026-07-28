package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_RateLimit_CounterKey(t *testing.T) {
	rt := RateLimit{Kind: RateLimitKindIP, Identifier: "1.2.3.4"}
	now := time.Date(2026, 7, 28, 10, 30, 15, 0, time.UTC)

	t.Run("when called twice within the same second, should return the same key", func(t *testing.T) {
		later := now.Add(999 * time.Millisecond)

		assert.Equal(t, rt.CounterKey(now), rt.CounterKey(later))
	})

	t.Run("when the second changes, should return a different key", func(t *testing.T) {
		next := now.Add(time.Second)

		assert.NotEqual(t, rt.CounterKey(now), rt.CounterKey(next))
	})

	t.Run("when kind is token, should not collide with the same identifier as ip", func(t *testing.T) {
		asToken := RateLimit{Kind: RateLimitKindToken, Identifier: "1.2.3.4"}

		assert.NotEqual(t, rt.CounterKey(now), asToken.CounterKey(now))
	})
}

func Test_RateLimit_BlockKey(t *testing.T) {
	t.Run("when kind differs, should not collide with the same identifier", func(t *testing.T) {
		asIP := RateLimit{Kind: RateLimitKindIP, Identifier: "abc123"}
		asToken := RateLimit{Kind: RateLimitKindToken, Identifier: "abc123"}

		assert.NotEqual(t, asIP.BlockKey(), asToken.BlockKey())
	})

	t.Run("when compared to the counter key, should use a separate namespace", func(t *testing.T) {
		rt := RateLimit{Kind: RateLimitKindIP, Identifier: "1.2.3.4"}
		now := time.Date(2026, 7, 28, 10, 30, 15, 0, time.UTC)

		assert.NotEqual(t, rt.CounterKey(now), rt.BlockKey())
	})
}
