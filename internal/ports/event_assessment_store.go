package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type EventAssessmentStore interface {
	SaveEventAssessments(ctx context.Context, assessments []domain.EventAssessment) error
}
