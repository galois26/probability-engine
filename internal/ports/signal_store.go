package ports

import (
	"context"

	"github.com/galois26/probability-engine/internal/domain"
)

type SignalStore interface {
	SaveSignals(ctx context.Context, signals []domain.Signal) error
}
