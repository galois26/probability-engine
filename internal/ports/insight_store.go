package ports

import (
	"context"

	"github.com/galois/probability-engine/internal/domain"
)

type InsightStore interface {
	SaveInsights(ctx context.Context, insights []domain.Insight) error
	UpdateInsightFlags(ctx context.Context, insightID string, flags domain.InsightFlags) error
}
