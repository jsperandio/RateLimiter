package integration

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/RateLimiter/internal/entity"
	"github.com/jsperandio/RateLimiter/internal/infra/storage"
	"github.com/jsperandio/RateLimiter/internal/infra/web/handler"
	"github.com/jsperandio/RateLimiter/internal/infra/web/middleware"
	"github.com/jsperandio/RateLimiter/internal/usecase"
)

const (
	concurrentRequests = 50
	ipMaxRequests      = 10
	tokenMaxRequests   = 25
)

func frozenClock() func() time.Time {
	now := time.Date(2026, 7, 28, 10, 30, 15, 0, time.UTC)

	return func() time.Time {
		return now
	}
}

func testLimits() entity.Limits {
	return entity.Limits{
		IPMaxRequests:      ipMaxRequests,
		IPBlockDuration:    5 * time.Minute,
		TokenMaxRequests:   tokenMaxRequests,
		TokenBlockDuration: 5 * time.Minute,
	}
}

func newMemoryStorage(t *testing.T, now func() time.Time) entity.LimiterStorage {
	t.Helper()

	stg, err := storage.NewMemoryStorage(&storage.MemoryStorageOptions{
		Now:             now,
		CleanupInterval: time.Hour,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = stg.Close() })

	return stg
}

func newRedisStorage(t *testing.T) entity.LimiterStorage {
	t.Helper()

	mr := miniredis.RunT(t)

	stg, err := storage.NewRedisStorage(t.Context(), &storage.RedisStorageOptions{
		Addr: mr.Addr(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = stg.Close() })

	return stg
}

func newTestServer(t *testing.T, stg entity.LimiterStorage, now func() time.Time) *httptest.Server {
	t.Helper()

	checker := usecase.NewCheckRateLimitUseCase(stg, testLimits(), &usecase.CheckRateLimitOptions{
		Now: now,
	})

	rateLimit := middleware.NewRateLimit(checker)

	e := echo.New()
	e.IPExtractor = echo.ExtractIPDirect()
	e.GET("/dummy", handler.Dummy, rateLimit)
	e.GET("/", handler.Dummy, rateLimit)
	e.GET("/health", handler.Health)

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	return srv
}

func fireConcurrent(t *testing.T, url, token string, total int) map[int]int {
	t.Helper()

	codes := make([]int, total)

	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := range total {
		wg.Go(func() {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			assert.NoError(t, err)

			if token != "" {
				req.Header.Set(middleware.HeaderAPIKey, token)
			}

			<-start

			resp, err := http.DefaultClient.Do(req)
			if !assert.NoError(t, err) {
				return
			}
			defer resp.Body.Close()

			codes[i] = resp.StatusCode
		})
	}

	close(start)
	wg.Wait()

	histogram := make(map[int]int)
	for _, code := range codes {
		histogram[code]++
	}

	return histogram
}

func Test_RateLimit_Concurrency(t *testing.T) {
	t.Run("when concurrent requests hit the memory storage within one window, should allow exactly the limit", func(t *testing.T) {
		now := frozenClock()
		srv := newTestServer(t, newMemoryStorage(t, now), now)

		got := fireConcurrent(t, srv.URL+"/dummy", "", concurrentRequests)

		assert.Equal(t, ipMaxRequests, got[http.StatusOK])
		assert.Equal(t, concurrentRequests-ipMaxRequests, got[http.StatusTooManyRequests])
	})

	t.Run("when concurrent requests hit redis within one window, should allow exactly the limit", func(t *testing.T) {
		now := frozenClock()
		srv := newTestServer(t, newRedisStorage(t), now)

		got := fireConcurrent(t, srv.URL+"/dummy", "", concurrentRequests)

		assert.Equal(t, ipMaxRequests, got[http.StatusOK])
		assert.Equal(t, concurrentRequests-ipMaxRequests, got[http.StatusTooManyRequests])
	})

	t.Run("when concurrent requests carry a token, should apply the token limit instead of the ip limit", func(t *testing.T) {
		now := frozenClock()
		srv := newTestServer(t, newRedisStorage(t), now)

		got := fireConcurrent(t, srv.URL+"/dummy", "abc123", concurrentRequests)

		assert.Equal(t, tokenMaxRequests, got[http.StatusOK],
			"o limite do token deve substituir o do ip mesmo sob concorrencia")
		assert.Equal(t, concurrentRequests-tokenMaxRequests, got[http.StatusTooManyRequests])
	})

	t.Run("when the same client alternates ip and token, should keep the groups keys independent", func(t *testing.T) {
		now := frozenClock()
		srv := newTestServer(t, newRedisStorage(t), now)

		byIP := fireConcurrent(t, srv.URL+"/dummy", "", concurrentRequests)
		byToken := fireConcurrent(t, srv.URL+"/dummy", "abc123", concurrentRequests)

		assert.Equal(t, ipMaxRequests, byIP[http.StatusOK])
		assert.Equal(t, tokenMaxRequests, byToken[http.StatusOK],
			"o grupo do token nao pode ser afetado pelo bloqueio do ip")
	})
}

func Test_RouteExemption(t *testing.T) {
	t.Run("when health is hammered, should never be rate limited", func(t *testing.T) {
		now := frozenClock()
		srv := newTestServer(t, newRedisStorage(t), now)

		got := fireConcurrent(t, srv.URL+"/health", "", concurrentRequests)

		assert.Equal(t, concurrentRequests, got[http.StatusOK],
			"o health check nao pode consumir cota do limiter")
		assert.Zero(t, got[http.StatusTooManyRequests])
	})

	t.Run("when the client is already blocked, should still answer health", func(t *testing.T) {
		now := frozenClock()
		srv := newTestServer(t, newRedisStorage(t), now)

		blocked := fireConcurrent(t, srv.URL+"/dummy", "", concurrentRequests)
		require.NotZero(t, blocked[http.StatusTooManyRequests], "o ip precisa estar bloqueado")

		got := fireConcurrent(t, srv.URL+"/health", "", concurrentRequests)

		assert.Equal(t, concurrentRequests, got[http.StatusOK],
			"health precisa responder mesmo com o ip bloqueado")
	})

	t.Run("when the root alias is hammered, should be rate limited like dummy", func(t *testing.T) {
		now := frozenClock()
		srv := newTestServer(t, newRedisStorage(t), now)

		got := fireConcurrent(t, srv.URL+"/", "", concurrentRequests)

		assert.Equal(t, ipMaxRequests, got[http.StatusOK])
		assert.Equal(t, concurrentRequests-ipMaxRequests, got[http.StatusTooManyRequests])
	})
}
