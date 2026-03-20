package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type RuleLoader interface {
	LoadSignalRules(ctx context.Context) ([]domain.SignalRule, error)
}
