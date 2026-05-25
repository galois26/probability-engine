package ports

import (
	"context"

	"github.com/galois26/probability-engine/internal/domain"
)

type EventAssessmentPublisher interface {
	PublishEventAssessments(ctx context.Context, assessments []domain.EventAssessment) error
}
