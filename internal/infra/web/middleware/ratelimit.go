package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/jsperandio/RateLimiter/internal/usecase"
)

type RateLimitChecker interface {
	Execute(ctx context.Context, input usecase.CheckRateLimitInputDTO) (usecase.CheckRateLimitOutputDTO, error)
}

func NewRateLimit(checker RateLimitChecker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			input := usecase.CheckRateLimitInputDTO{
				IP:    c.RealIP(),
				Token: c.Request().Header.Get(HeaderAPIKey),
			}

			output, err := checker.Execute(c.Request().Context(), input)
			if err != nil {
				slog.Error("rate limit check failed, letting the request through",
					"error", err,
					"ip", input.IP,
				)

				return next(c)
			}

			if !output.Allowed {
				slog.Warn("request blocked by rate limit",
					"kind", output.Kind,
					"limit", output.Limit,
				)

				return c.String(http.StatusTooManyRequests, TooManyRequestsMessage)
			}

			c.Set(ContextKeyRateLimit, output)

			return next(c)
		}
	}
}

func RateLimitFromContext(c *echo.Context) (usecase.CheckRateLimitOutputDTO, bool) {
	output, ok := c.Get(ContextKeyRateLimit).(usecase.CheckRateLimitOutputDTO)

	return output, ok
}
