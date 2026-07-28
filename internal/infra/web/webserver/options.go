package webserver

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type WebServerOptions struct {
	Port            string        `env:"HTTP_PORT" envDefault:"8080"`
	GracefulTimeout time.Duration `env:"HTTP_GRACEFUL_TIMEOUT" envDefault:"10s"`
}

func NewDefaultWebServerOptions() (*WebServerOptions, error) {
	opt, err := env.ParseAs[WebServerOptions]()
	if err != nil {
		return nil, err
	}

	return &opt, nil
}
