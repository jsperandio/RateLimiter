package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jsperandio/RateLimiter/internal/entity"
	"github.com/jsperandio/RateLimiter/internal/infra/web/middleware"
	"github.com/jsperandio/RateLimiter/internal/usecase"
)

func newTestContext(t *testing.T) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/dummy", nil)
	rec := httptest.NewRecorder()

	return e.NewContext(req, rec), rec
}

func Test_Dummy(t *testing.T) {
	t.Run("when the context carries the rate limit result, should return it as json", func(t *testing.T) {
		c, rec := newTestContext(t)
		c.Set(middleware.ContextKeyRateLimit, usecase.CheckRateLimitOutputDTO{
			Allowed:   true,
			Kind:      entity.RateLimitKindIP,
			Limit:     10,
			Remaining: 7,
		})

		require.NoError(t, Dummy(c))

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"kind":"ip","limit":10,"remaining":7}`, rec.Body.String())
	})

	t.Run("when the limit came from a token, should report the token kind", func(t *testing.T) {
		c, rec := newTestContext(t)
		c.Set(middleware.ContextKeyRateLimit, usecase.CheckRateLimitOutputDTO{
			Allowed:   true,
			Kind:      entity.RateLimitKindToken,
			Limit:     100,
			Remaining: 99,
		})

		require.NoError(t, Dummy(c))

		assert.JSONEq(t, `{"kind":"token","limit":100,"remaining":99}`, rec.Body.String())
	})

	t.Run("when the response is built, should not echo the identifier", func(t *testing.T) {
		c, rec := newTestContext(t)
		c.Set(middleware.ContextKeyRateLimit, usecase.CheckRateLimitOutputDTO{
			Allowed:   true,
			Kind:      entity.RateLimitKindToken,
			Limit:     100,
			Remaining: 99,
		})

		require.NoError(t, Dummy(c))

		assert.NotContains(t, rec.Body.String(), "identifier",
			"o token nao deve vazar no corpo da resposta")
	})

	t.Run("when the context has no rate limit result, should return a plain ok", func(t *testing.T) {
		c, rec := newTestContext(t)

		require.NoError(t, Dummy(c))

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})
}

func Test_Health(t *testing.T) {
	t.Run("when called, should return ok without touching the rate limit", func(t *testing.T) {
		c, rec := newTestContext(t)

		require.NoError(t, Health(c))

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})
}
