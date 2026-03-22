package ports

import (
	"context"
	"time"

	"probability-engine/internal/domain"
)

type EventSource interface {
	FetchEvents(ctx context.Context, from time.Time) ([]domain.Event, error)
}


