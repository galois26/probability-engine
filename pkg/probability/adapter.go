package probability

import (
	"context"
	"fmt"
	"time"

	"probability-engine/internal/domain"
	"probability-engine/internal/engine"
)

type InternalEngineAdapter struct {
	engine        *engine.Engine
	engineVersion string
	ruleVersion   string
}

func NewInternalEngineAdapter(
	eng *engine.Engine,
	engineVersion string,
	ruleVersion string,
) *InternalEngineAdapter {
	return &InternalEngineAdapter{
		engine:        eng,
		engineVersion: engineVersion,
		ruleVersion:   ruleVersion,
	}
}

func (a *InternalEngineAdapter) Assess(ctx context.Context, events []Event) ([]Assessment, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("probability: nil internal engine")
	}

	domainEvents := make([]domain.Event, 0, len(events))
	for _, ev := range events {
		domainEvents = append(domainEvents, toDomainEvent(ev))
	}

	assessments, err := a.engine.AssessEvents(ctx, domainEvents)
	if err != nil {
		return nil, err
	}

	out := make([]Assessment, 0, len(assessments))
	for _, aev := range assessments {
		out = append(out, fromDomainAssessment(aev, a.engineVersion, a.ruleVersion))
	}

	return out, nil
}

func toDomainEvent(ev Event) domain.Event {
	published := ev.PublishedAt
	if published.IsZero() {
		published = time.Now().UTC()
	}

	return domain.Event{
		ID:        ev.ID,
		Source:    ev.Source,
		Title:     ev.Title,
		Summary:   ev.Summary,
		URL:       ev.URL,
		Published: published,
		Lang:      ev.Lang,
		Country:   ev.Country,
		Labels:    ev.Labels,
		Raw:       ev.Raw,
	}
}

func fromDomainAssessment(a domain.EventAssessment, engineVersion, ruleVersion string) Assessment {
	signals := make([]Signal, 0, len(a.Signals))
	for _, s := range a.Signals {
		signals = append(signals, Signal{
			ID:          s.ID,
			Kind:        s.Kind,
			Probability: s.Probability,
			Classifier:  s.Classifier,
			Features:    s.Features,
			Explanation: s.Explanation,
			Labels:      s.Labels,
		})
	}

	classifiers := make([]ClassifierResult, 0, len(a.Classifiers))
	for _, c := range a.Classifiers {
		classifiers = append(classifiers, ClassifierResult{
			Classifier: c.Classifier,
			Accepted:   c.Decision.Accepted,
			State:      string(c.Decision.State),
			Confidence: c.Decision.Confidence,
			Reasons:    c.Decision.Reasons,
		})
	}

	fp := ""
	if a.Event.Labels != nil {
		fp = a.Event.Labels["fingerprint"]
	}

	return Assessment{
		ID:          a.ID,
		EventID:     a.Event.ID,
		Fingerprint: fp,
		AssessedAt:  a.AssessedAt,
		Decision: Decision{
			State:        string(a.Decision.State),
			Accepted:     a.Decision.Accepted,
			PrimaryClass: a.Decision.PrimaryClass,
			Confidence:   a.Decision.Confidence,
			Threshold:    a.Decision.Threshold,
			Reasons:      a.Decision.Reasons,
		},
		Signals:       signals,
		Classifiers:   classifiers,
		EngineVersion: engineVersion,
		RuleVersion:   ruleVersion,
	}
}
