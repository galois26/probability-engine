package ports

import (
	"context"

	"github.com/galois/probability-engine/internal/domain"
)

type InsightEnricher interface {
	Enrich(ctx context.Context, in domain.Insight) (domain.Insight, error)
}
