package ports

import (
	"context"

	"github.com/galois/probability-engine/internal/domain"
)

type SignalStore interface {
	SaveSignals(ctx context.Context, signals []domain.Signal) error
}
