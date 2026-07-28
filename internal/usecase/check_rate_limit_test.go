package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/RateLimiter/internal/entity"
	"github.com/jsperandio/RateLimiter/internal/infra/storage"
)

type testClock struct {
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{
		now: time.Date(2026, 7, 28, 10, 30, 15, 0, time.UTC),
	}
}

func (tc *testClock) Now() time.Time {
	return tc.now
}

func (tc *testClock) Advance(d time.Duration) {
	tc.now = tc.now.Add(d)
}

type spyStorage struct {
	entity.LimiterStorage
	increments int
}

func (ss *spyStorage) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	ss.increments++

	return ss.LimiterStorage.Increment(ctx, key, window)
}

var errStorage = errors.New("storage is down")

type failingStorage struct {
	failIsBlocked bool
	failIncrement bool
	failBlock     bool
}

func (fs failingStorage) Increment(context.Context, string, time.Duration) (int64, error) {
	if fs.failIncrement {
		return 0, errStorage
	}

	return 1, nil
}

func (fs failingStorage) IsBlocked(context.Context, string) (bool, error) {
	if fs.failIsBlocked {
		return false, errStorage
	}

	return false, nil
}

func (fs failingStorage) Block(context.Context, string, time.Duration) error {
	if fs.failBlock {
		return errStorage
	}

	return nil
}

func testLimits() entity.Limits {
	return entity.Limits{
		IPMaxRequests:      10,
		IPBlockDuration:    5 * time.Minute,
		TokenMaxRequests:   100,
		TokenBlockDuration: 5 * time.Minute,
	}
}

func newTestUseCase(t *testing.T, clock *testClock) (*CheckRateLimitUseCase, *spyStorage) {
	t.Helper()

	memory, err := storage.NewMemoryStorage(&storage.MemoryStorageOptions{
		Now:             clock.Now,
		CleanupInterval: time.Hour,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = memory.Close() })

	spy := &spyStorage{
		LimiterStorage: memory,
	}

	opt := &CheckRateLimitOptions{
		Now: clock.Now,
	}

	return NewCheckRateLimitUseCase(spy, testLimits(), opt), spy
}

func Test_CheckRateLimitUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	byIP := CheckRateLimitInputDTO{
		IP: "1.2.3.4",
	}

	t.Run("when under the limit, should allow and report the remaining requests", func(t *testing.T) {
		uc, _ := newTestUseCase(t, newTestClock())

		got, err := uc.Execute(ctx, byIP)

		require.NoError(t, err)
		assert.True(t, got.Allowed)
		assert.Equal(t, entity.RateLimitKindIP, got.Kind)
		assert.Equal(t, 10, got.Limit)
		assert.Equal(t, 9, got.Remaining)
	})

	t.Run("when exactly at the limit, should still allow", func(t *testing.T) {
		uc, _ := newTestUseCase(t, newTestClock())

		var got CheckRateLimitOutputDTO
		for range 10 {
			var err error
			got, err = uc.Execute(ctx, byIP)
			require.NoError(t, err)
		}

		assert.True(t, got.Allowed)
		assert.Equal(t, 0, got.Remaining)
	})

	t.Run("when the limit is exceeded, should deny", func(t *testing.T) {
		uc, _ := newTestUseCase(t, newTestClock())

		for range 10 {
			_, err := uc.Execute(ctx, byIP)
			require.NoError(t, err)
		}

		got, err := uc.Execute(ctx, byIP)

		require.NoError(t, err)
		assert.False(t, got.Allowed)
	})

	t.Run("when the limit is exceeded, should block for the configured duration", func(t *testing.T) {
		clock := newTestClock()
		uc, _ := newTestUseCase(t, clock)

		for range 11 {
			_, err := uc.Execute(ctx, byIP)
			require.NoError(t, err)
		}

		clock.Advance(2 * time.Second)

		got, err := uc.Execute(ctx, byIP)

		require.NoError(t, err)
		assert.False(t, got.Allowed, "a janela virou mas o bloqueio de 5min continua valendo")
	})

	t.Run("when already blocked, should keep denying without incrementing", func(t *testing.T) {
		clock := newTestClock()
		uc, spy := newTestUseCase(t, clock)

		for range 11 {
			_, err := uc.Execute(ctx, byIP)
			require.NoError(t, err)
		}

		incrementsWhenBlocked := spy.increments

		clock.Advance(2 * time.Second)
		got, err := uc.Execute(ctx, byIP)

		require.NoError(t, err)
		assert.False(t, got.Allowed)
		assert.Equal(t, incrementsWhenBlocked, spy.increments, "requisicao bloqueada nao deve contar")
	})

	t.Run("when the block expires, should allow again", func(t *testing.T) {
		clock := newTestClock()
		uc, _ := newTestUseCase(t, clock)

		for range 11 {
			_, err := uc.Execute(ctx, byIP)
			require.NoError(t, err)
		}

		clock.Advance(5*time.Minute + time.Second)

		got, err := uc.Execute(ctx, byIP)

		require.NoError(t, err)
		assert.True(t, got.Allowed)
		assert.Equal(t, 9, got.Remaining, "o contador deve recomecar do zero")
	})

	t.Run("when the second rolls over, should reset the counter", func(t *testing.T) {
		clock := newTestClock()
		uc, _ := newTestUseCase(t, clock)

		for range 10 {
			_, err := uc.Execute(ctx, byIP)
			require.NoError(t, err)
		}

		clock.Advance(time.Second)

		got, err := uc.Execute(ctx, byIP)

		require.NoError(t, err)
		assert.True(t, got.Allowed)
		assert.Equal(t, 9, got.Remaining)
	})

	t.Run("when a token is present, should apply the token limit instead of the ip limit", func(t *testing.T) {
		uc, _ := newTestUseCase(t, newTestClock())
		byToken := CheckRateLimitInputDTO{
			IP:    "1.2.3.4",
			Token: "abc123",
		}

		for i := range 11 {
			got, err := uc.Execute(ctx, byToken)

			require.NoError(t, err)
			assert.True(t, got.Allowed, "requisicao %d deveria passar: o limite do token e 100", i+1)
			assert.Equal(t, entity.RateLimitKindToken, got.Kind)
			assert.Equal(t, 100, got.Limit)
		}
	})

	t.Run("when the token limit is exceeded, should deny", func(t *testing.T) {
		uc, _ := newTestUseCase(t, newTestClock())
		byToken := CheckRateLimitInputDTO{
			IP:    "1.2.3.4",
			Token: "abc123",
		}

		for range 100 {
			_, err := uc.Execute(ctx, byToken)
			require.NoError(t, err)
		}

		got, err := uc.Execute(ctx, byToken)

		require.NoError(t, err)
		assert.False(t, got.Allowed)
	})

	t.Run("when the ip is blocked, should still allow the same client using a token", func(t *testing.T) {
		uc, _ := newTestUseCase(t, newTestClock())

		for range 11 {
			_, err := uc.Execute(ctx, byIP)
			require.NoError(t, err)
		}

		got, err := uc.Execute(ctx, CheckRateLimitInputDTO{
			IP:    "1.2.3.4",
			Token: "abc123",
		})

		require.NoError(t, err)
		assert.True(t, got.Allowed, "o balde do token e independente do balde do ip")
	})

	t.Run("when neither ip nor token is given, should return ErrNoIdentifier", func(t *testing.T) {
		uc, _ := newTestUseCase(t, newTestClock())

		got, err := uc.Execute(ctx, CheckRateLimitInputDTO{})

		assert.ErrorIs(t, err, entity.ErrNoIdentifier)
		assert.False(t, got.Allowed)
	})

	t.Run("when the storage fails to report the block, should return the error", func(t *testing.T) {
		stg := failingStorage{
			failIsBlocked: true,
		}
		uc := NewCheckRateLimitUseCase(stg, testLimits(), nil)

		got, err := uc.Execute(ctx, byIP)

		assert.ErrorIs(t, err, errStorage)
		assert.False(t, got.Allowed)
	})

	t.Run("when the storage fails to increment, should return the error", func(t *testing.T) {
		stg := failingStorage{
			failIncrement: true,
		}
		uc := NewCheckRateLimitUseCase(stg, testLimits(), nil)

		got, err := uc.Execute(ctx, byIP)

		assert.ErrorIs(t, err, errStorage)
		assert.False(t, got.Allowed)
	})
}
