package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type InsightAggregator interface {
	Aggregate(ctx context.Context, signals []domain.Signal, eventsByID map[string]domain.Event) ([]domain.Insight, error)
}
