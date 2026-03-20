package ports

import (
	"context"

	"probability-engine/internal/domain"
)

type SignalStore interface {
	SaveSignals(ctx context.Context, signals []domain.Signal) error
}
