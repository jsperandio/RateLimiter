package configs

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Conf struct {
	HTTPPort        string `env:"HTTP_PORT" envDefault:"8080"`
	StorageStrategy string `env:"STORAGE_STRATEGY" envDefault:"redis"`

	IPMaxRequests      int           `env:"RATE_LIMIT_IP_MAX_REQUESTS" envDefault:"10"`
	IPBlockDuration    time.Duration `env:"RATE_LIMIT_IP_BLOCK_DURATION" envDefault:"1m"`
	TokenMaxRequests   int           `env:"RATE_LIMIT_TOKEN_MAX_REQUESTS" envDefault:"100"`
	TokenBlockDuration time.Duration `env:"RATE_LIMIT_TOKEN_BLOCK_DURATION" envDefault:"1m"`

	RedisAddr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	MemcachedAddr string `env:"MEMCACHED_ADDR" envDefault:"localhost:11211"`
}

func LoadConfig(path string) (*Conf, error) {
	if path != "" {
		_ = godotenv.Load(path)
	}

	cfg, err := env.ParseAs[Conf]()
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
