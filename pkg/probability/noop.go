package probability

import (
	"context"
	"time"
)

type NoopEngine struct {
	Config Config
}

func NewNoopEngine(cfg Config) *NoopEngine {
	return &NoopEngine{Config: cfg}
}
func (e *NoopEngine) Assess(ctx context.Context, events []Event) ([]Assessment, error) {
	now := time.Now().UTC()
	out := make([]Assessment, 0, len(events))

	for _, ev := range events {
		fp := ev.Fingerprint
		if fp == "" {
			fp = EventFingerprint(ev)
		}

		out = append(out, Assessment{
			ID: AssessmentID(fp, e.Config.EngineVersion, e.Config.RuleVersion),
			Event: EventSnapshot{
				ID:        ev.ID,
				Source:    ev.Source,
				Title:     ev.Title,
				Summary:   ev.Summary,
				URL:       ev.URL,
				Published: ev.PublishedAt,
				Lang:      ev.Lang,
				Country:   ev.Country,
				Labels:    ev.Labels,
				Raw:       ev.Raw,
			},
			AssessedAt: now,
			Decision: ClassificationDecision{
				State:    "noop",
				Accepted: false,
				Reasons:  []string{"noop engine"},
			},
			EngineVersion: e.Config.EngineVersion,
			RuleVersion:   e.Config.RuleVersion,
		})
	}

	return out, nil
}
