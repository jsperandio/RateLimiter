package configs

import (
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/jsperandio/RateLimiter/internal/entity"
)

type RateLimitOptions struct {
	IPMaxRequests      int           `env:"RATE_LIMIT_IP_MAX_REQUESTS" envDefault:"10"`
	IPBlockDuration    time.Duration `env:"RATE_LIMIT_IP_BLOCK_DURATION" envDefault:"1m"`
	TokenMaxRequests   int           `env:"RATE_LIMIT_TOKEN_MAX_REQUESTS" envDefault:"100"`
	TokenBlockDuration time.Duration `env:"RATE_LIMIT_TOKEN_BLOCK_DURATION" envDefault:"1m"`
}

func NewDefaultRateLimitOptions() (entity.Limits, error) {
	conf, err := env.ParseAs[RateLimitOptions]()
	if err != nil {
		return entity.Limits{}, err
	}

	return entity.Limits{
		IPMaxRequests:      conf.IPMaxRequests,
		IPBlockDuration:    conf.IPBlockDuration,
		TokenMaxRequests:   conf.TokenMaxRequests,
		TokenBlockDuration: conf.TokenBlockDuration,
	}, nil
}
