package storage

import (
	"context"
	"io"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewDefaultStorageOptions(t *testing.T) {
	t.Run("when STORAGE_STRATEGY is unset, should default to memory", func(t *testing.T) {
		t.Setenv("STORAGE_STRATEGY", "")

		got, err := NewDefaultStorageOptions()

		require.NoError(t, err)
		assert.Equal(t, StrategyMemory, got.Strategy)
	})

	t.Run("when STORAGE_STRATEGY is set, should use it", func(t *testing.T) {
		t.Setenv("STORAGE_STRATEGY", StrategyRedis)

		got, err := NewDefaultStorageOptions()

		require.NoError(t, err)
		assert.Equal(t, StrategyRedis, got.Strategy)
	})
}

func Test_New(t *testing.T) {
	ctx := context.Background()

	t.Run("when the strategy is memory, should return the memory storage", func(t *testing.T) {
		t.Setenv("MEMORY_CLEANUP_INTERVAL", "1m")

		got, err := New(ctx, &StorageOptions{
			Strategy: StrategyMemory,
		})

		require.NoError(t, err)
		assert.IsType(t, &MemoryStorage{}, got)

		require.NoError(t, got.(io.Closer).Close())
	})

	t.Run("when the strategy is redis, should return the redis storage", func(t *testing.T) {
		mr := miniredis.RunT(t)
		t.Setenv("REDIS_ADDR", mr.Addr())

		got, err := New(ctx, &StorageOptions{
			Strategy: StrategyRedis,
		})

		require.NoError(t, err)
		assert.IsType(t, &RedisStorage{}, got)

		require.NoError(t, got.(io.Closer).Close())
	})

	t.Run("when the strategy has spaces or uppercase, should still resolve", func(t *testing.T) {
		t.Setenv("MEMORY_CLEANUP_INTERVAL", "1m")

		got, err := New(ctx, &StorageOptions{
			Strategy: "  MEMORY ",
		})

		require.NoError(t, err)
		assert.IsType(t, &MemoryStorage{}, got)

		require.NoError(t, got.(io.Closer).Close())
	})

	t.Run("when the config is nil, should read the strategy from the environment", func(t *testing.T) {
		t.Setenv("STORAGE_STRATEGY", StrategyMemory)
		t.Setenv("MEMORY_CLEANUP_INTERVAL", "1m")

		got, err := New(ctx, nil)

		require.NoError(t, err)
		assert.IsType(t, &MemoryStorage{}, got)

		require.NoError(t, got.(io.Closer).Close())
	})

	t.Run("when the strategy is unknown, should return ErrUnknownStrategy", func(t *testing.T) {
		got, err := New(ctx, &StorageOptions{
			Strategy: "postgres",
		})

		assert.ErrorIs(t, err, ErrUnknownStrategy)
		assert.Nil(t, got)
	})

	t.Run("when the strategy is empty, should return ErrUnknownStrategy", func(t *testing.T) {
		got, err := New(ctx, &StorageOptions{})

		assert.ErrorIs(t, err, ErrUnknownStrategy)
		assert.Nil(t, got)
	})

	t.Run("when redis is unreachable, should fail instead of returning a broken storage", func(t *testing.T) {
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()

		t.Setenv("REDIS_ADDR", addr)

		got, err := New(ctx, &StorageOptions{
			Strategy: StrategyRedis,
		})

		assert.Error(t, err)
		assert.Nil(t, got)
	})
}
