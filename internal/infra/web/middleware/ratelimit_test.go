package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/RateLimiter/internal/entity"
	"github.com/jsperandio/RateLimiter/internal/usecase"
)

type mockRateLimitChecker struct {
	allowed       bool
	shouldFail    bool
	err           error
	receivedInput usecase.CheckRateLimitInputDTO
	wasCalled     bool
}

func (mc *mockRateLimitChecker) Execute(
	ctx context.Context,
	input usecase.CheckRateLimitInputDTO,
) (usecase.CheckRateLimitOutputDTO, error) {
	mc.wasCalled = true
	mc.receivedInput = input

	if mc.shouldFail {
		return usecase.CheckRateLimitOutputDTO{}, mc.err
	}

	output := usecase.CheckRateLimitOutputDTO{
		Allowed:   mc.allowed,
		Kind:      entity.RateLimitKindIP,
		Limit:     10,
		Remaining: 9,
	}

	return output, nil
}

func Test_NewRateLimit(t *testing.T) {
	t.Run("when the request is allowed, should call the next handler", func(t *testing.T) {
		checker := &mockRateLimitChecker{
			allowed: true,
		}
		middleware := NewRateLimit(checker)

		handlerWasCalled := false
		nextHandler := func(c *echo.Context) error {
			handlerWasCalled = true
			return nil
		}

		wrappedHandler := middleware(nextHandler)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := wrappedHandler(c)

		require.NoError(t, err)
		assert.True(t, handlerWasCalled)
	})

	t.Run("when the request is denied, should answer 429", func(t *testing.T) {
		checker := &mockRateLimitChecker{
			allowed: false,
		}
		middleware := NewRateLimit(checker)

		nextHandler := func(c *echo.Context) error {
			return nil
		}

		wrappedHandler := middleware(nextHandler)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = wrappedHandler(c)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	t.Run("when the request is denied, should answer the exact challenge message", func(t *testing.T) {
		checker := &mockRateLimitChecker{
			allowed: false,
		}
		middleware := NewRateLimit(checker)

		nextHandler := func(c *echo.Context) error {
			return nil
		}

		wrappedHandler := middleware(nextHandler)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = wrappedHandler(c)

		assert.Equal(t, TooManyRequestsMessage, rec.Body.String())
	})

	t.Run("when the request is denied, should not call the next handler", func(t *testing.T) {
		checker := &mockRateLimitChecker{
			allowed: false,
		}
		middleware := NewRateLimit(checker)

		handlerWasCalled := false
		nextHandler := func(c *echo.Context) error {
			handlerWasCalled = true
			return nil
		}

		wrappedHandler := middleware(nextHandler)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = wrappedHandler(c)

		assert.False(t, handlerWasCalled)
	})

	t.Run("when the API_KEY header is present, should forward the token to the usecase", func(t *testing.T) {
		checker := &mockRateLimitChecker{
			allowed: true,
		}
		middleware := NewRateLimit(checker)

		nextHandler := func(c *echo.Context) error {
			return nil
		}

		wrappedHandler := middleware(nextHandler)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)
		req.Header.Set(HeaderAPIKey, "test-token-123")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = wrappedHandler(c)

		assert.True(t, checker.wasCalled)
		assert.Equal(t, "test-token-123", checker.receivedInput.Token)
	})

	t.Run("when there is no API_KEY header, should forward only the ip", func(t *testing.T) {
		checker := &mockRateLimitChecker{
			allowed: true,
		}
		middleware := NewRateLimit(checker)

		nextHandler := func(c *echo.Context) error {
			return nil
		}

		wrappedHandler := middleware(nextHandler)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = wrappedHandler(c)

		assert.True(t, checker.wasCalled)
		assert.Empty(t, checker.receivedInput.Token)
		assert.NotEmpty(t, checker.receivedInput.IP)
	})

	t.Run("when the usecase fails, should fail open and call the next handler", func(t *testing.T) {
		checker := &mockRateLimitChecker{
			shouldFail: true,
			err:        context.DeadlineExceeded,
		}
		middleware := NewRateLimit(checker)

		handlerWasCalled := false
		nextHandler := func(c *echo.Context) error {
			handlerWasCalled = true
			return nil
		}

		wrappedHandler := middleware(nextHandler)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := wrappedHandler(c)

		require.NoError(t, err)
		assert.True(t, handlerWasCalled)
	})
}
