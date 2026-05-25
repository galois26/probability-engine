package ports

import (
	"context"
	"time"

	"github.com/galois/probability-engine/internal/domain"
)

type EventSource interface {
	FetchEvents(ctx context.Context, from time.Time) ([]domain.Event, error)
}
