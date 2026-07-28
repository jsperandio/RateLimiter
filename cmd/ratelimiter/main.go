package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/jsperandio/RateLimiter/configs"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)))

	cfg, err := configs.LoadConfig(".env")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	e := echo.New()
	e.Logger = slog.Default()
	e.IPExtractor = echo.ExtractIPDirect()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address:         ":" + cfg.HTTPPort,
		GracefulTimeout: 10 * time.Second,
		OnShutdownError: func(err error) {
			slog.Error("graceful shutdown timed out", "error", err)
		},
	}

	slog.Info("starting server", "port", cfg.HTTPPort, "storage", cfg.StorageStrategy)

	if err := sc.Start(ctx, e); err != nil {
		slog.Error("server terminated", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}
