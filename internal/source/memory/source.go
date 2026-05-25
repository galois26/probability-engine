package memory

import (
	"context"
	"time"

	"github.com/galois26/probability-engine/internal/domain"
)

type Source struct {
	events []domain.Event
}

func New(events []domain.Event) *Source {
	return &Source{events: events}
}

func (s *Source) FetchEvents(ctx context.Context, from time.Time) ([]domain.Event, error) {
	_ = ctx
	out := make([]domain.Event, 0, len(s.events))
	for _, ev := range s.events {
		if ev.Published.After(from) || ev.Published.Equal(from) {
			out = append(out, ev)
		}
	}
	return out, nil
}
