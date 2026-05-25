package noop

import (
	"context"

	"github.com/galois26/probability-engine/internal/domain"
)

type Enricher struct{}

func New() *Enricher {
	return &Enricher{}
}

func (e *Enricher) Enrich(ctx context.Context, in domain.Insight) (domain.Insight, error) {
	_ = ctx
	return in, nil
}
