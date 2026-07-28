package webserver

import "time"

const (
	defaultPort            = "8080"
	defaultGracefulTimeout = 10 * time.Second
)

type WebServerOptions struct {
	Port            string
	GracefulTimeout time.Duration
}

func NewDefaultWebServerOptions() *WebServerOptions {
	return &WebServerOptions{
		Port:            defaultPort,
		GracefulTimeout: defaultGracefulTimeout,
	}
}
