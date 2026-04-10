package s3

import (
	"context"
	"fmt"
	"log"

	"probability-engine/internal/domain"
)

func (s *Store) SaveEventAssessments(ctx context.Context, assessments []domain.EventAssessment) error {
	now := s.clock.Now()

	payload := EventAssessmentSnapshot{
		Version:     "v1",
		GeneratedAt: now,
		Count:       len(assessments),
		Items:       assessments,
	}

	if err := s.putJSON(ctx, s.latestKey("events"), payload); err != nil {
		return fmt.Errorf("save latest event assessments: %w", err)
	}
	if err := s.putJSON(ctx, s.runKey("events", now), payload); err != nil {
		return fmt.Errorf("save event assessments run snapshot: %w", err)
	}

	log.Printf("s3 store: writing event assessments latest key=%s", s.latestKey("events"))
	log.Printf("s3 store: writing event assessments run key=%s", s.runKey("events", now))

	return nil
}
