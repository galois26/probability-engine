package s3

import (
	"context"
	"fmt"
	"log"

	"github.com/galois26/probability-engine/internal/domain"
)

func (s *Store) SaveSignals(ctx context.Context, signals []domain.Signal) error {
	now := s.clock.Now()

	payload := SignalSnapshot{
		Version:     "v1",
		GeneratedAt: now,
		Count:       len(signals),
		Items:       signals,
	}

	if err := s.putJSON(ctx, s.latestKey("signals"), payload); err != nil {
		return fmt.Errorf("save latest signals: %w", err)
	}
	if err := s.putJSON(ctx, s.runKey("signals", now), payload); err != nil {
		return fmt.Errorf("save signals run snapshot: %w", err)
	}
	log.Printf("s3 store: writing signals latest key=%s", s.latestKey("signals"))
	log.Printf("s3 store: writing signals run key=%s", s.runKey("signals", now))
	return nil
}
