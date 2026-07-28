package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/jsperandio/RateLimiter/configs"
	"github.com/jsperandio/RateLimiter/internal/entity"
	"github.com/jsperandio/RateLimiter/internal/infra/storage"
	"github.com/jsperandio/RateLimiter/internal/infra/web/middleware"
	"github.com/jsperandio/RateLimiter/internal/infra/web/webserver"
	"github.com/jsperandio/RateLimiter/internal/usecase"
)

const gracefulTimeout = 10 * time.Second

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)))

	if err := run(); err != nil {
		slog.Error("application terminated", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}

func run() error {
	cfg, err := configs.LoadConfig(".env")
	if err != nil {
		return err
	}

	globalLimits := entity.Limits{
		IPMaxRequests:      cfg.IPMaxRequests,
		IPBlockDuration:    cfg.IPBlockDuration,
		TokenMaxRequests:   cfg.TokenMaxRequests,
		TokenBlockDuration: cfg.TokenBlockDuration,
	}

	if err := globalLimits.Validate(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	limiterStorage, err := storage.New(ctx, storage.Config{
		Strategy:      cfg.StorageStrategy,
		RedisAddr:     cfg.RedisAddr,
		RedisPassword: cfg.RedisPassword,
		RedisDB:       cfg.RedisDB,
	})
	if err != nil {
		return err
	}

	if closer, ok := limiterStorage.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}

	checker := usecase.NewCheckRateLimitUseCase(limiterStorage, globalLimits, nil)

	ws := webserver.NewWebServer(&webserver.WebServerOptions{
		Port:            cfg.HTTPPort,
		GracefulTimeout: gracefulTimeout,
	})

	ws.Use(middleware.NewRateLimit(checker))

	ws.RegisterRoute(http.MethodGet, "/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	ws.RegisterRoute(http.MethodGet, "/health", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	slog.Info("starting server",
		"port", cfg.HTTPPort,
		"storage", cfg.StorageStrategy,
		"ip_limit", cfg.IPMaxRequests,
		"token_limit", cfg.TokenMaxRequests,
	)

	return ws.Start(ctx)
}
