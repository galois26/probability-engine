package memory

import (
	"context"
	"sync"

	"github.com/galois/probability-engine/internal/domain"
)

type EventAssessmentStore struct {
	mu          sync.RWMutex
	assessments []domain.EventAssessment
}

func NewEventAssessmentStore() *EventAssessmentStore {
	return &EventAssessmentStore{}
}

func (s *EventAssessmentStore) SaveEventAssessments(_ context.Context, assessments []domain.EventAssessment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.assessments = append([]domain.EventAssessment(nil), assessments...)
	return nil
}

func (s *EventAssessmentStore) Snapshot() []domain.EventAssessment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.EventAssessment, len(s.assessments))
	copy(out, s.assessments)
	return out
}
