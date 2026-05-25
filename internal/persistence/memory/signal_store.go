package memory

import (
	"context"
	"sync"

	"github.com/galois/probability-engine/internal/domain"
)

type SignalStore struct {
	mu      sync.Mutex
	signals []domain.Signal
}

func NewSignalStore() *SignalStore {
	return &SignalStore{
		signals: make([]domain.Signal, 0, 256),
	}
}

func (s *SignalStore) SaveSignals(ctx context.Context, signals []domain.Signal) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	s.signals = append(s.signals, signals...)
	return nil
}

func (s *SignalStore) Snapshot() []domain.Signal {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]domain.Signal, len(s.signals))
	copy(out, s.signals)
	return out
}
