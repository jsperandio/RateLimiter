package storage

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedisStorage(t *testing.T) (*RedisStorage, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	rs, err := NewRedisStorage(t.Context(), &RedisStorageOptions{
		Addr: mr.Addr(),
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = rs.Close() })

	return rs, mr
}

func Test_NewRedisStorage(t *testing.T) {
	t.Run("when redis answers the ping, should return the storage", func(t *testing.T) {
		mr := miniredis.RunT(t)

		got, err := NewRedisStorage(t.Context(), &RedisStorageOptions{
			Addr: mr.Addr(),
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.NoError(t, got.Close())
	})

	t.Run("when redis is unreachable, should fail instead of returning a broken storage", func(t *testing.T) {
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()

		got, err := NewRedisStorage(t.Context(), &RedisStorageOptions{
			Addr: addr,
		})

		assert.Error(t, err)
		assert.Nil(t, got)
	})
}

func Test_RedisStorage_Increment(t *testing.T) {
	ctx := context.Background()

	t.Run("when the key is new, should start at one", func(t *testing.T) {
		rs, _ := newTestRedisStorage(t)

		got, err := rs.Increment(ctx, "counter", time.Second)

		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})

	t.Run("when called repeatedly, should accumulate", func(t *testing.T) {
		rs, _ := newTestRedisStorage(t)

		for i := 1; i <= 5; i++ {
			got, err := rs.Increment(ctx, "counter", time.Second)

			require.NoError(t, err)
			assert.Equal(t, int64(i), got)
		}
	})

	t.Run("when the key is new, should set the expiration", func(t *testing.T) {
		rs, mr := newTestRedisStorage(t)

		_, err := rs.Increment(ctx, "counter", time.Second)
		require.NoError(t, err)

		assert.Equal(t, time.Second, mr.TTL("counter"), "chave sem ttl vazaria memoria no redis")
	})

	t.Run("when the window expires, should start over", func(t *testing.T) {
		rs, mr := newTestRedisStorage(t)

		_, err := rs.Increment(ctx, "counter", time.Second)
		require.NoError(t, err)

		mr.FastForward(2 * time.Second)

		got, err := rs.Increment(ctx, "counter", time.Second)

		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})

	t.Run("when keys differ, should count independently", func(t *testing.T) {
		rs, _ := newTestRedisStorage(t)

		_, err := rs.Increment(ctx, "first", time.Second)
		require.NoError(t, err)

		got, err := rs.Increment(ctx, "second", time.Second)

		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})
}

func Test_RedisStorage_IsBlocked(t *testing.T) {
	ctx := context.Background()

	t.Run("when the key was never blocked, should return false", func(t *testing.T) {
		rs, _ := newTestRedisStorage(t)

		got, err := rs.IsBlocked(ctx, "block")

		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("when the key is blocked, should return true", func(t *testing.T) {
		rs, _ := newTestRedisStorage(t)

		require.NoError(t, rs.Block(ctx, "block", time.Minute))

		got, err := rs.IsBlocked(ctx, "block")

		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("when the block duration has passed, should return false", func(t *testing.T) {
		rs, mr := newTestRedisStorage(t)

		require.NoError(t, rs.Block(ctx, "block", time.Minute))

		mr.FastForward(time.Minute + time.Second)

		got, err := rs.IsBlocked(ctx, "block")

		require.NoError(t, err)
		assert.False(t, got)
	})
}

func Test_RedisStorage_Block(t *testing.T) {
	ctx := context.Background()

	t.Run("when blocking, should set the key with the given ttl", func(t *testing.T) {
		rs, mr := newTestRedisStorage(t)

		require.NoError(t, rs.Block(ctx, "block", 5*time.Minute))

		assert.True(t, mr.Exists("block"))
		assert.Equal(t, 5*time.Minute, mr.TTL("block"))
	})

	t.Run("when a key is blocked, should not affect other keys", func(t *testing.T) {
		rs, _ := newTestRedisStorage(t)

		require.NoError(t, rs.Block(ctx, "blocked", time.Minute))

		got, err := rs.IsBlocked(ctx, "other")

		require.NoError(t, err)
		assert.False(t, got)
	})
}
