package storage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{
		now: time.Date(2026, 7, 28, 10, 30, 15, 0, time.UTC),
	}
}

func (tc *testClock) Now() time.Time {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	return tc.now
}

func (tc *testClock) Advance(d time.Duration) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.now = tc.now.Add(d)
}

func newTestMemoryStorage(t *testing.T, clock *testClock, cleanup time.Duration) *MemoryStorage {
	t.Helper()

	ms, err := NewMemoryStorage(&MemoryStorageOptions{
		Now:             clock.Now,
		CleanupInterval: cleanup,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ms.Close() })

	return ms
}

func Test_MemoryStorage_Increment(t *testing.T) {
	ctx := context.Background()

	t.Run("when the key is new, should start at one", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		got, err := ms.Increment(ctx, "counter", time.Second)

		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})

	t.Run("when called repeatedly, should accumulate", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		for i := 1; i <= 5; i++ {
			got, err := ms.Increment(ctx, "counter", time.Second)

			require.NoError(t, err)
			assert.Equal(t, int64(i), got)
		}
	})

	t.Run("when the window expires, should start over", func(t *testing.T) {
		clock := newTestClock()
		ms := newTestMemoryStorage(t, clock, time.Minute)

		_, err := ms.Increment(ctx, "counter", time.Second)
		require.NoError(t, err)

		clock.Advance(2 * time.Second)

		got, err := ms.Increment(ctx, "counter", time.Second)

		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})

	t.Run("when keys differ, should count independently", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		_, err := ms.Increment(ctx, "first", time.Second)
		require.NoError(t, err)
		_, err = ms.Increment(ctx, "first", time.Second)
		require.NoError(t, err)

		got, err := ms.Increment(ctx, "second", time.Second)

		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})

	t.Run("when called concurrently, should not lose increments", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		const (
			goroutines = 50
			perRoutine = 20
		)

		var wg sync.WaitGroup
		for range goroutines {
			wg.Go(func() {
				for range perRoutine {
					_, err := ms.Increment(ctx, "counter", time.Hour)
					assert.NoError(t, err)
				}
			})
		}
		wg.Wait()

		got, err := ms.Increment(ctx, "counter", time.Hour)

		require.NoError(t, err)
		assert.Equal(t, int64(goroutines*perRoutine+1), got)
	})
}

func Test_MemoryStorage_IsBlocked(t *testing.T) {
	ctx := context.Background()

	t.Run("when the key was never blocked, should return false", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		got, err := ms.IsBlocked(ctx, "block")

		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("when the key is blocked, should return true", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		require.NoError(t, ms.Block(ctx, "block", time.Minute))

		got, err := ms.IsBlocked(ctx, "block")

		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("when the block duration has passed, should return false", func(t *testing.T) {
		clock := newTestClock()
		ms := newTestMemoryStorage(t, clock, time.Hour)

		require.NoError(t, ms.Block(ctx, "block", time.Minute))

		clock.Advance(time.Minute + time.Second)

		got, err := ms.IsBlocked(ctx, "block")

		require.NoError(t, err)
		assert.False(t, got)
	})
}

func Test_MemoryStorage_Block(t *testing.T) {
	ctx := context.Background()

	t.Run("when blocking twice, should extend the deadline", func(t *testing.T) {
		clock := newTestClock()
		ms := newTestMemoryStorage(t, clock, time.Hour)

		require.NoError(t, ms.Block(ctx, "block", time.Minute))

		clock.Advance(30 * time.Second)
		require.NoError(t, ms.Block(ctx, "block", time.Minute))

		clock.Advance(45 * time.Second)

		got, err := ms.IsBlocked(ctx, "block")

		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("when a key is blocked, should not affect other keys", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		require.NoError(t, ms.Block(ctx, "blocked", time.Minute))

		got, err := ms.IsBlocked(ctx, "other")

		require.NoError(t, err)
		assert.False(t, got)
	})
}

func Test_MemoryStorage_Close(t *testing.T) {
	ctx := context.Background()

	t.Run("when the cleaner runs, should evict expired entries", func(t *testing.T) {
		clock := newTestClock()
		ms := newTestMemoryStorage(t, clock, 5*time.Millisecond)

		_, err := ms.Increment(ctx, "counter", time.Second)
		require.NoError(t, err)
		require.Equal(t, 1, ms.len())

		clock.Advance(2 * time.Second)

		assert.Eventually(
			t,
			func() bool {
				return ms.len() == 0
			},
			time.Second,
			5*time.Millisecond,
			"o cleaner deveria remover a entrada vencida")
	})

	t.Run("when closed, should stop the cleaner", func(t *testing.T) {
		clock := newTestClock()
		ms, err := NewMemoryStorage(&MemoryStorageOptions{
			Now:             clock.Now,
			CleanupInterval: 5 * time.Millisecond,
		})
		require.NoError(t, err)

		require.NoError(t, ms.Close())

		_, err = ms.Increment(ctx, "counter", time.Second)
		require.NoError(t, err)

		clock.Advance(2 * time.Second)

		assert.Never(
			t,
			func() bool {
				return ms.len() == 0
			},
			50*time.Millisecond,
			5*time.Millisecond,
			"o cleaner parado nao deveria mais varrer o mapa")
	})

	t.Run("when closed twice, should not panic", func(t *testing.T) {
		ms := newTestMemoryStorage(t, newTestClock(), time.Minute)

		require.NoError(t, ms.Close())
		assert.NotPanics(t, func() { _ = ms.Close() })
	})
}
