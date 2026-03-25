package s3

import (
	"context"
	"fmt"

	"probability-engine/internal/domain"
)

func (s *Store) SaveInsights(ctx context.Context, insights []domain.Insight) error {
	now := s.clock.Now()

	payload := InsightSnapshot{
		Version:     "v1",
		GeneratedAt: now,
		Count:       len(insights),
		Items:       insights,
	}

	if err := s.putJSON(ctx, s.latestKey("insights"), payload); err != nil {
		return fmt.Errorf("save latest insights: %w", err)
	}
	if err := s.putJSON(ctx, s.runKey("insights", now), payload); err != nil {
		return fmt.Errorf("save insights run snapshot: %w", err)
	}
	return nil
}

func (s *Store) UpdateInsightFlags(ctx context.Context, insightID string, flags domain.InsightFlags) error {
	var snap InsightSnapshot
	if err := s.getJSON(ctx, s.latestKey("insights"), &snap); err != nil {
		return fmt.Errorf("load latest insights: %w", err)
	}

	found := false
	now := s.clock.Now()

	for i := range snap.Items {
		if snap.Items[i].ID == insightID {
			snap.Items[i].Flags = flags
			snap.Items[i].UpdatedAt = now
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("insight %s not found", insightID)
	}

	snap.GeneratedAt = now
	snap.Count = len(snap.Items)

	if err := s.putJSON(ctx, s.latestKey("insights"), snap); err != nil {
		return fmt.Errorf("update latest insights: %w", err)
	}
	if err := s.putJSON(ctx, s.runKey("insights", now), snap); err != nil {
		return fmt.Errorf("write updated insights snapshot: %w", err)
	}
	return nil
}
