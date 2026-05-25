package ports

import (
	"context"

	"github.com/galois/probability-engine/internal/domain"
)

type EventAssessmentStore interface {
	SaveEventAssessments(ctx context.Context, assessments []domain.EventAssessment) error
}
