package storage

import "time"

const defaultCleanupInterval = time.Minute

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
