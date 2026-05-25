package ports

import (
	"context"

	"github.com/galois26/probability-engine/internal/domain"
)

type RuleLoader interface {
	LoadSignalRules(ctx context.Context) ([]domain.SignalRule, error)
}
