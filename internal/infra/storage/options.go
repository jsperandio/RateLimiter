package storage

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type StorageOptions struct {
	Strategy string `env:"STORAGE_STRATEGY" envDefault:"memory"`
}

func NewDefaultStorageOptions() (*StorageOptions, error) {
	opt, err := env.ParseAs[StorageOptions]()
	if err != nil {
		return nil, err
	}

	return &opt, nil
}

type MemoryStorageOptions struct {
	CleanupInterval time.Duration `env:"MEMORY_CLEANUP_INTERVAL" envDefault:"1m"`
	Now             func() time.Time
}

func NewDefaultMemoryStorageOptions() (*MemoryStorageOptions, error) {
	opt, err := env.ParseAs[MemoryStorageOptions]()
	if err != nil {
		return nil, err
	}

	opt.Now = time.Now

	return &opt, nil
}

type RedisStorageOptions struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

func NewDefaultRedisStorageOptions() (*RedisStorageOptions, error) {
	opt, err := env.ParseAs[RedisStorageOptions]()
	if err != nil {
		return nil, err
	}

	return &opt, nil
}
