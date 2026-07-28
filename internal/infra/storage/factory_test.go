package storage

import (
	"context"
	"io"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	ctx := context.Background()

	t.Run("when the strategy is memory, should return the memory storage", func(t *testing.T) {
		got, err := New(ctx, Config{
			Strategy: StrategyMemory,
		})

		require.NoError(t, err)
		assert.IsType(t, &MemoryStorage{}, got)

		require.NoError(t, got.(io.Closer).Close())
	})

	t.Run("when the strategy is redis, should return the redis storage", func(t *testing.T) {
		mr := miniredis.RunT(t)

		got, err := New(ctx, Config{
			Strategy:  StrategyRedis,
			RedisAddr: mr.Addr(),
		})

		require.NoError(t, err)
		assert.IsType(t, &RedisStorage{}, got)

		require.NoError(t, got.(io.Closer).Close())
	})

	t.Run("when the strategy has spaces or uppercase, should still resolve", func(t *testing.T) {
		got, err := New(ctx, Config{
			Strategy: "  MEMORY ",
		})

		require.NoError(t, err)
		assert.IsType(t, &MemoryStorage{}, got)

		require.NoError(t, got.(io.Closer).Close())
	})

	t.Run("when the strategy is unknown, should return ErrUnknownStrategy", func(t *testing.T) {
		got, err := New(ctx, Config{
			Strategy: "postgres",
		})

		assert.ErrorIs(t, err, ErrUnknownStrategy)
		assert.Nil(t, got)
	})

	t.Run("when the strategy is empty, should return ErrUnknownStrategy", func(t *testing.T) {
		got, err := New(ctx, Config{})

		assert.ErrorIs(t, err, ErrUnknownStrategy)
		assert.Nil(t, got)
	})

	t.Run("when redis is unreachable, should fail instead of returning a broken storage", func(t *testing.T) {
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()

		got, err := New(ctx, Config{
			Strategy:  StrategyRedis,
			RedisAddr: addr,
		})

		assert.Error(t, err)
		assert.Nil(t, got)
	})
}
