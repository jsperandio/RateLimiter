package storage

import "time"

const (
	defaultCleanupInterval = time.Minute
	defaultRedisAddr       = "localhost:6379"
)

type MemoryStorageOptions struct {
	CleanupInterval time.Duration
	Now             func() time.Time
}

func NewDefaultMemoryStorageOptions() *MemoryStorageOptions {
	return &MemoryStorageOptions{
		CleanupInterval: defaultCleanupInterval,
		Now:             time.Now,
	}
}

type RedisStorageOptions struct {
	Addr     string
	Password string
	DB       int
}

func NewDefaultRedisStorageOptions() *RedisStorageOptions {
	return &RedisStorageOptions{
		Addr: defaultRedisAddr,
	}
}
