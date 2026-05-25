package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type Assessor interface {
	AssessEvents(
		ctx context.Context,
		events []domain.Event,
	) ([]domain.EventAssessment, error)
}
