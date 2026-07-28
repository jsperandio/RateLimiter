package storage

import (
	"context"
	"sync"
	"time"
)

type msRow struct {
	value     int64
	expiresAt time.Time
}

type MemoryStorage struct {
	options *MemoryStorageOptions

	mu    sync.RWMutex
	table map[string]msRow

	stop     chan struct{}
	stopOnce sync.Once
}

func NewMemoryStorage(opt *MemoryStorageOptions) (*MemoryStorage, error) {
	if opt == nil {
		parsed, err := NewDefaultMemoryStorageOptions()
		if err != nil {
			return nil, err
		}

		opt = parsed
	}

	if opt.Now == nil {
		opt.Now = time.Now
	}

	if opt.CleanupInterval <= 0 {
		return nil, ErrInvalidCleanupInterval
	}

	ms := &MemoryStorage{
		options: opt,
		table:   make(map[string]msRow),
		stop:    make(chan struct{}),
	}

	go ms.cleaner()

	return ms, nil
}

func (ms *MemoryStorage) Increment(_ context.Context, key string, window time.Duration) (int64, error) {
	now := ms.options.Now()

	ms.mu.Lock()
	defer ms.mu.Unlock()

	current, ok := ms.table[key]
	if !ok || now.After(current.expiresAt) {
		ms.table[key] = msRow{
			value:     1,
			expiresAt: now.Add(window),
		}

		return 1, nil
	}

	current.value++
	ms.table[key] = current

	return current.value, nil
}

func (ms *MemoryStorage) IsBlocked(_ context.Context, key string) (bool, error) {
	now := ms.options.Now()

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entry, ok := ms.table[key]
	if !ok || now.After(entry.expiresAt) {
		return false, nil
	}

	return true, nil
}

func (ms *MemoryStorage) Block(_ context.Context, key string, duration time.Duration) error {
	now := ms.options.Now()

	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.table[key] = msRow{
		value:     1,
		expiresAt: now.Add(duration),
	}

	return nil
}

func (ms *MemoryStorage) Close() error {
	ms.stopOnce.Do(func() {
		close(ms.stop)
	})

	return nil
}

func (ms *MemoryStorage) cleaner() {
	ticker := time.NewTicker(ms.options.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stop:
			return
		case <-ticker.C:
			ms.deleteExpired()
		}
	}
}

func (ms *MemoryStorage) deleteExpired() {
	now := ms.options.Now()

	ms.mu.Lock()
	defer ms.mu.Unlock()

	for k, v := range ms.table {
		if now.After(v.expiresAt) {
			delete(ms.table, k)
		}
	}
}

func (ms *MemoryStorage) len() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return len(ms.table)
}
