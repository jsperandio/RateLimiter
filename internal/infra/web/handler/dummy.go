package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/jsperandio/RateLimiter/internal/infra/web/middleware"
)

func Dummy(c *echo.Context) error {
	output, ok := middleware.RateLimitFromContext(c)
	if !ok {
		return c.String(http.StatusOK, "ok")
	}

	return c.JSON(http.StatusOK, DummyResponse{
		Kind:      string(output.Kind),
		Limit:     output.Limit,
		Remaining: output.Remaining,
	})
}
