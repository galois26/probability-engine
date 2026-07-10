package probability

import (
	"context"
	"fmt"

	"github.com/galois26/probability-engine/internal/domain"
	"github.com/galois26/probability-engine/internal/ports"
)

type EngineAdapter struct {
	engine        ports.Assessor
	engineVersion string
	ruleVersion   string
}

func NewEngineAdapter(
	eng ports.Assessor,
	engineVersion string,
	ruleVersion string,
) *EngineAdapter {
	return &EngineAdapter{
		engine:        eng,
		engineVersion: engineVersion,
		ruleVersion:   ruleVersion,
	}
}

// Assess adapts public probability events into domain events, runs the internal
// assessor, and maps domain assessments back to the public API shape.
func (a *EngineAdapter) Assess(ctx context.Context, events []Event) ([]Assessment, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("probability: nil internal engine")
	}

	if len(events) == 0 {
		return []Assessment{}, nil
	}

	domainEvents := make([]domain.Event, 0, len(events))
	for _, ev := range events {
		domainEvents = append(domainEvents, toDomainEvent(withFingerprint(ev)))
	}

	domainAssessments, err := a.engine.AssessEvents(ctx, domainEvents)
	if err != nil {
		return nil, err
	}

	assessments := make([]Assessment, 0, len(domainAssessments))
	for _, assessment := range domainAssessments {
		assessments = append(assessments, fromDomainAssessment(
			assessment,
			a.engineVersion,
			a.ruleVersion,
		))
	}

	return assessments, nil
}

// withFingerprint guarantees every event entering the domain layer has a stable
// fingerprint. Caller-provided fingerprints are preserved.
func withFingerprint(ev Event) Event {
	if ev.Fingerprint != "" {
		return ev
	}

	ev.Fingerprint = EventFingerprint(ev)
	return ev
}

// toDomainEvent converts the public event type into the internal domain type.
// It intentionally does not set default timestamps or add infrastructure data.
func toDomainEvent(ev Event) domain.Event {
	return domain.Event{
		ID:          ev.ID,
		Fingerprint: ev.Fingerprint,
		Source:      ev.Source,
		Title:       ev.Title,
		Summary:     ev.Summary,
		URL:         ev.URL,
		Published:   ev.PublishedAt,
		Lang:        ev.Lang,
		Country:     ev.Country,
		Labels:      ev.Labels,
		Raw:         ev.Raw,
	}
}

// fromDomainAssessment converts a complete domain assessment into the public
// assessment contract consumed by probability-engine callers such as multi-ingester.
func fromDomainAssessment(a domain.EventAssessment, engineVersion, ruleVersion string) Assessment {
	return Assessment{
		ID:         a.ID,
		EventID:    a.Event.ID,
		RunID:      a.RunID,
		AssessedAt: a.AssessedAt,

		Event: fromDomainEventSnapshot(a.Event),

		Decision: ClassificationDecision{
			State:        string(a.Decision.State),
			Accepted:     a.Decision.Accepted,
			PrimaryClass: a.Decision.PrimaryClass,
			Confidence:   a.Decision.Confidence,
			Threshold:    a.Decision.Threshold,
			Reasons:      a.Decision.Reasons,
		},
		Signals:       mapSignalSnapshots(a.Signals),
		Classifiers:   mapClassifierResults(a.Classifiers),
		EngineVersion: engineVersion,
		RuleVersion:   ruleVersion,
	}
}

// fromDomainEventSnapshot maps the domain event snapshot directly to the public
// event snapshot. Fingerprint is copied from the domain snapshot without recomputing.
func fromDomainEventSnapshot(ev domain.EventSnapshot) EventSnapshot {
	return EventSnapshot{
		ID:          ev.ID,
		Fingerprint: ev.Fingerprint,
		Source:      ev.Source,
		Title:       ev.Title,
		Summary:     ev.Summary,
		URL:         ev.URL,
		Published:   ev.Published,
		Lang:        ev.Lang,
		Country:     ev.Country,
		Labels:      ev.Labels,
		Raw:         ev.Raw,
	}
}

func mapSignalSnapshots(signals []domain.SignalSnapshot) []SignalSnapshot {
	out := make([]SignalSnapshot, 0, len(signals))
	for _, s := range signals {
		out = append(out, SignalSnapshot{
			ID:          s.ID,
			EventID:     s.EventID,
			Kind:        s.Kind,
			MarketScope: s.MarketScope,
			Direction:   string(s.Direction),
			Probability: s.Probability,
			Classifier:  s.Classifier,
			Features:    s.Features,
			Explanation: s.Explanation,
			Labels:      s.Labels,
			CreatedAt:   s.CreatedAt,
			Trace:       s.Trace,
		})
	}
	return out
}

func mapClassifierResults(classifiers []domain.ClassifierAssessmentResult) []ClassifierAssessmentResult {
	out := make([]ClassifierAssessmentResult, 0, len(classifiers))
	for _, c := range classifiers {
		out = append(out, ClassifierAssessmentResult{
			Classifier: c.Classifier,
			Features: FeatureAssessment{
				HasFeatures: c.Features.HasFeatures,
				Tokens:      c.Features.Tokens,
				Keywords:    c.Features.Keywords,
				Reason:      c.Features.Reason,
			},
			Rules: RuleAssessment{
				Evaluated: c.Rules.Evaluated,
				Matched:   c.Rules.Matched,
				Matches:   mapRuleMatches(c.Rules.Matches),
				Reason:    c.Rules.Reason,
			},
			NaiveBayes: NaiveBayesAssessment{
				Evaluated:      c.NaiveBayes.Evaluated,
				PredictedClass: c.NaiveBayes.PredictedClass,
				Scores:         mapClassScores(c.NaiveBayes.Scores),
				Reason:         c.NaiveBayes.Reason,
			},
			Decision: ClassificationDecision{
				State:        string(c.Decision.State),
				Accepted:     c.Decision.Accepted,
				PrimaryClass: c.Decision.PrimaryClass,
				Confidence:   c.Decision.Confidence,
				Threshold:    c.Decision.Threshold,
				Reasons:      c.Decision.Reasons,
			},
			Signals: mapSignals(c.Signals),
		})
	}
	return out
}

func mapRuleMatches(matches []domain.RuleMatchResult) []RuleMatchResult {
	out := make([]RuleMatchResult, 0, len(matches))
	for _, m := range matches {
		out = append(out, RuleMatchResult{
			RuleID:       m.RuleID,
			RuleName:     m.RuleName,
			Matched:      m.Matched,
			Score:        m.Score,
			MatchedTerms: m.MatchedTerms,
			Reason:       m.Reason,
		})
	}
	return out
}

func mapClassScores(scores []domain.ClassScore) []ClassScore {
	out := make([]ClassScore, 0, len(scores))
	for _, s := range scores {
		out = append(out, ClassScore{
			Class:       s.Class,
			Score:       s.Score,
			Probability: s.Probability,
		})
	}
	return out
}

func mapSignals(signals []domain.Signal) []Signal {
	out := make([]Signal, 0, len(signals))
	for _, s := range signals {
		out = append(out, Signal{
			ID:          s.ID,
			EventID:     s.EventID,
			Kind:        s.Kind,
			MarketScope: s.MarketScope,
			Direction:   string(s.Direction),
			Probability: s.Probability,
			Classifier:  s.Classifier,
			Features:    s.Features,
			Explanation: s.Explanation,
			Labels:      s.Labels,
			CreatedAt:   s.CreatedAt,
			Trace:       s.Trace,
		})
	}
	return out
}
