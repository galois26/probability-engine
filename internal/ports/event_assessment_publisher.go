package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type EventAssessmentPublisher interface {
	PublishEventAssessments(ctx context.Context, assessments []domain.EventAssessment) error
}
