package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLimits() Limits {
	return Limits{
		IPMaxRequests:      10,
		IPBlockDuration:    5 * time.Minute,
		TokenMaxRequests:   100,
		TokenBlockDuration: 10 * time.Minute,
	}
}

func Test_Limits_Validate(t *testing.T) {
	t.Run("when all values are positive, should return nil", func(t *testing.T) {
		assert.NoError(t, testLimits().Validate())
	})

	t.Run("when ip max requests is zero, should return ErrInvalidMaxRequests", func(t *testing.T) {
		limits := testLimits()
		limits.IPMaxRequests = 0

		assert.ErrorIs(t, limits.Validate(), ErrInvalidMaxRequests)
	})

	t.Run("when token max requests is negative, should return ErrInvalidMaxRequests", func(t *testing.T) {
		limits := testLimits()
		limits.TokenMaxRequests = -1

		assert.ErrorIs(t, limits.Validate(), ErrInvalidMaxRequests)
	})

	t.Run("when ip block duration is zero, should return ErrInvalidBlockDuration", func(t *testing.T) {
		limits := testLimits()
		limits.IPBlockDuration = 0

		assert.ErrorIs(t, limits.Validate(), ErrInvalidBlockDuration)
	})

	t.Run("when token block duration is negative, should return ErrInvalidBlockDuration", func(t *testing.T) {
		limits := testLimits()
		limits.TokenBlockDuration = -time.Second

		assert.ErrorIs(t, limits.Validate(), ErrInvalidBlockDuration)
	})
}

func Test_ResolveLimit(t *testing.T) {
	t.Run("when token is present, should use the token limits", func(t *testing.T) {
		got, err := ResolveLimit("1.2.3.4", "abc123", testLimits())

		require.NoError(t, err)
		assert.Equal(t, RateLimitKindToken, got.Kind)
		assert.Equal(t, "abc123", got.Identifier)
		assert.Equal(t, 100, got.MaxRequests)
		assert.Equal(t, 10*time.Minute, got.BlockDuration)
	})

	t.Run("when token limit is higher than ip limit, should keep the token limit", func(t *testing.T) {
		limits := Limits{
			IPMaxRequests:      10,
			IPBlockDuration:    time.Minute,
			TokenMaxRequests:   100,
			TokenBlockDuration: time.Minute,
		}

		got, err := ResolveLimit("1.2.3.4", "abc123", limits)

		require.NoError(t, err)
		assert.Equal(t, 100, got.MaxRequests, "o limite do token deve substituir o do ip")
	})

	t.Run("when token limit is lower than ip limit, should still keep the token limit", func(t *testing.T) {
		limits := Limits{
			IPMaxRequests:      100,
			IPBlockDuration:    time.Minute,
			TokenMaxRequests:   5,
			TokenBlockDuration: time.Minute,
		}

		got, err := ResolveLimit("1.2.3.4", "abc123", limits)

		require.NoError(t, err)
		assert.Equal(t, 5, got.MaxRequests, "substitui, nao pega o maior")
	})

	t.Run("when only ip is present, should use the ip limits", func(t *testing.T) {
		got, err := ResolveLimit("1.2.3.4", "", testLimits())

		require.NoError(t, err)
		assert.Equal(t, RateLimitKindIP, got.Kind)
		assert.Equal(t, "1.2.3.4", got.Identifier)
		assert.Equal(t, 10, got.MaxRequests)
		assert.Equal(t, 5*time.Minute, got.BlockDuration)
	})

	t.Run("when token is only whitespace, should fall back to the ip limits", func(t *testing.T) {
		got, err := ResolveLimit("1.2.3.4", "   ", testLimits())

		require.NoError(t, err)
		assert.Equal(t, RateLimitKindIP, got.Kind)
	})

	t.Run("when token has surrounding spaces, should not create a separate bucket", func(t *testing.T) {
		spaced, err := ResolveLimit("1.2.3.4", "  abc123  ", testLimits())
		require.NoError(t, err)

		clean, err := ResolveLimit("1.2.3.4", "abc123", testLimits())
		require.NoError(t, err)

		assert.Equal(t, clean.BlockKey(), spaced.BlockKey())
	})

	t.Run("when ip and token are both empty, should return ErrNoIdentifier", func(t *testing.T) {
		got, err := ResolveLimit("", "", testLimits())

		assert.ErrorIs(t, err, ErrNoIdentifier)
		assert.Empty(t, got.Identifier)
	})
}
