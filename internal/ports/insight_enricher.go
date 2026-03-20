package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type InsightEnricher interface {
	Enrich(ctx context.Context, in domain.Insight) (domain.Insight, error)
}
