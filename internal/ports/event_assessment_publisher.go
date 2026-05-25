package ports

import (
	"context"

	"github.com/galois/probability-engine/internal/domain"
)

type EventAssessmentPublisher interface {
	PublishEventAssessments(ctx context.Context, assessments []domain.EventAssessment) error
}
