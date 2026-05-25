package ports

import (
	"context"

	"github.com/galois/probability-engine/internal/domain"
)

type RuleLoader interface {
	LoadSignalRules(ctx context.Context) ([]domain.SignalRule, error)
}
