package probability

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"probability-engine/internal/domain"
	"probability-engine/internal/ports"
	"probability-engine/pkg/probability"
	"time"
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

func (a *EngineAdapter) Assess(ctx context.Context, events []Event) ([]Assessment, error) {
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
	log.Printf("probability adapter: domain_events=%d domain_assessments=%d", len(domainEvents), len(assessments))
	out := make([]Assessment, 0, len(assessments))
	for _, aev := range assessments {
		out = append(out, fromDomainAssessment(aev, a.engineVersion, a.ruleVersion))
	}
	log.Printf("prob adapter input event id=%q title=%q", domainEvents[0].ID, domainEvents[0].Title)
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

func fromDomainAssessment(a probability.EventAssessment, engineVersion, ruleVersion string) Assessment {
	return Assessment{
		ID:         a.ID,
		EventID:    a.Event.ID,
		RunID:      a.RunID,
		AssessedAt: a.AssessedAt,

		Event: EventSnapshot{
			ID:        a.Event.ID,
			Source:    a.Event.Source,
			Title:     a.Event.Title,
			Summary:   a.Event.Summary,
			URL:       a.Event.URL,
			Published: a.Event.Published,
			Lang:      a.Event.Lang,
			Country:   a.Event.Country,
			Labels:    a.Event.Labels,
			Raw:       a.Event.Raw,
		},

		Decision: ClassificationDecision{
			State:        string(a.Decision.State),
			Accepted:     a.Decision.Accepted,
			PrimaryClass: a.Decision.PrimaryClass,
			Confidence:   a.Decision.Confidence,
			Threshold:    a.Decision.Threshold,
			Reasons:      a.Decision.Reasons,
		},
		Signals:       mapSignalSnapshots(a.Signals),
		Classifiers:   mapClassifiers(a.Classifiers),
		EngineVersion: engineVersion,
		RuleVersion:   ruleVersion,
	}
}

func rawMapFromJSON(b []byte) map[string]any {
	if len(b) == 0 {
		return nil
	}
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
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

func mapClassifiers(classifiers []domain.ClassifierAssessmentResult) []ClassifierAssessmentResult {
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
