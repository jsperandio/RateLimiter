package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jsperandio/RateLimiter/configs"
	"github.com/jsperandio/RateLimiter/internal/infra/storage"
	"github.com/jsperandio/RateLimiter/internal/infra/web/handler"
	"github.com/jsperandio/RateLimiter/internal/infra/web/middleware"
	"github.com/jsperandio/RateLimiter/internal/infra/web/webserver"
	"github.com/jsperandio/RateLimiter/internal/usecase"
)

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
	configs.LoadEnv(".env")

	globalLimits, err := configs.NewDefaultRateLimitOptions()
	if err != nil {
		return err
	}

	if err := globalLimits.Validate(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	storageOptions, err := storage.NewDefaultStorageOptions()
	if err != nil {
		return err
	}

	limiterStorage, err := storage.New(ctx, storageOptions)
	if err != nil {
		return err
	}

	if closer, ok := limiterStorage.(io.Closer); ok {
		defer func() {
			_ = closer.Close()
		}()
	}

	checker := usecase.NewCheckRateLimitUseCase(limiterStorage, globalLimits, nil)

	ws, err := webserver.NewWebServer(nil)
	if err != nil {
		return err
	}

	ws.RegisterRoute(http.MethodGet, "/health", handler.Health)

	rateLimit := middleware.NewRateLimit(checker)
	ws.RegisterRoute(http.MethodGet, "/dummy", handler.Dummy, rateLimit)
	ws.RegisterRoute(http.MethodGet, "/", handler.Dummy, rateLimit)

	slog.Info("starting application",
		"storage", storageOptions.Strategy,
		"ip_limit", globalLimits.IPMaxRequests,
		"ip_block_duration", globalLimits.IPBlockDuration,
		"token_limit", globalLimits.TokenMaxRequests,
		"token_block_duration", globalLimits.TokenBlockDuration,
	)

	return ws.Start(ctx)
}
