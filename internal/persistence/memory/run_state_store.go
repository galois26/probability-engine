package memory

import (
	"context"
	"sync"
	"time"
)

type RunStateStore struct {
	mu   sync.RWMutex
	data map[string]time.Time
}

func NewRunStateStore() *RunStateStore {
	return &RunStateStore{
		data: make(map[string]time.Time),
	}
}

func (s *RunStateStore) LoadLastRun(ctx context.Context, job string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[job].UTC(), nil
}

func (s *RunStateStore) SaveLastRun(ctx context.Context, job string, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[job] = t.UTC()
	return nil
}
