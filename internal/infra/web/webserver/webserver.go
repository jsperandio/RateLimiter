package webserver

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type WebServer struct {
	options *WebServerOptions
	echo    *echo.Echo
}

func NewWebServer(opt *WebServerOptions) (*WebServer, error) {
	if opt == nil {
		parsed, err := NewDefaultWebServerOptions()
		if err != nil {
			return nil, err
		}

		opt = parsed
	}

	if opt.Port == "" {
		return nil, ErrEmptyPort
	}

	e := echo.New()
	e.Logger = slog.Default()
	e.IPExtractor = echo.ExtractIPDirect()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	return &WebServer{
		options: opt,
		echo:    e,
	}, nil
}

func (ws *WebServer) Use(mw ...echo.MiddlewareFunc) {
	ws.echo.Use(mw...)
}

func (ws *WebServer) RegisterRoute(method, path string, handler echo.HandlerFunc) {
	ws.echo.Add(method, path, handler)
}

func (ws *WebServer) Start(ctx context.Context) error {
	sc := echo.StartConfig{
		Address:         ":" + ws.options.Port,
		HideBanner:      true,
		GracefulTimeout: ws.options.GracefulTimeout,
		OnShutdownError: func(err error) {
			slog.Error("shutdown error", "err", err)
		},
	}

	return sc.Start(ctx, ws.echo)
}
