package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/galois/probability-engine/internal/domain"
)

type InsightStore struct {
	mu       sync.Mutex
	insights map[string]domain.Insight
}

func NewInsightStore() *InsightStore {
	return &InsightStore{
		insights: make(map[string]domain.Insight),
	}
}

func (s *InsightStore) SaveInsights(ctx context.Context, insights []domain.Insight) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, in := range insights {
		s.insights[in.ID] = in
	}
	return nil
}

func (s *InsightStore) UpdateInsightFlags(ctx context.Context, insightID string, flags domain.InsightFlags) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	in, ok := s.insights[insightID]
	if !ok {
		return fmt.Errorf("insight not found: %s", insightID)
	}

	in.Flags = flags
	s.insights[insightID] = in
	return nil
}

func (s *InsightStore) Snapshot() []domain.Insight {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]domain.Insight, 0, len(s.insights))
	for _, in := range s.insights {
		out = append(out, in)
	}
	return out
}
