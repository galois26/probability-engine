package engine

import (
	"time"

	"probability-engine/internal/domain"
)

func buildEventAssessment(
	now time.Time,
	ev domain.Event,
	classifierResults []domain.ClassifierAssessmentResult,
	eventSignals []domain.Signal,
) domain.EventAssessment {
	return domain.EventAssessment{
		ID:          buildAssessmentID(now, ev.ID),
		RunID:       buildRunID(now),
		AssessedAt:  now,
		Event:       domain.NewEventSnapshot(ev),
		Decision:    resolveEventDecision(classifierResults, eventSignals),
		Signals:     toSignalSnapshots(eventSignals),
		Classifiers: classifierResults,
	}
}

func fallbackClassifierAssessmentResult(name string, signals []domain.Signal) domain.ClassifierAssessmentResult {
	decision := classifySignalDecision(signals)
	reasons := []string{"classifier does not expose assessment details"}

	if decision.Accepted {
		reasons = append(reasons, "signal emitted")
	} else {
		reasons = append(reasons, "no signal emitted")
	}
	decision.Reasons = reasons

	return domain.ClassifierAssessmentResult{
		Classifier: name,
		Features: domain.FeatureAssessment{
			HasFeatures: false,
			Reason:      "feature assessment not available via current classifier interface",
		},
		Rules: domain.RuleAssessment{
			Evaluated: false,
			Matched:   false,
			Reason:    "rule assessment not available via current classifier interface",
		},
		NaiveBayes: domain.NaiveBayesAssessment{
			Evaluated: false,
			Reason:    "naive bayes assessment not available via current classifier interface",
		},
		Decision: decision,
		Signals:  signals,
	}
}

func resolveEventDecision(
	classifierResults []domain.ClassifierAssessmentResult,
	signals []domain.Signal,
) domain.ClassificationDecision {
	if len(signals) > 0 {
		top := signals[0]
		for _, s := range signals[1:] {
			if s.Probability > top.Probability {
				top = s
			}
		}

		return domain.ClassificationDecision{
			State:        domain.DecisionAcceptedSignal,
			Accepted:     true,
			PrimaryClass: top.Kind,
			Confidence:   top.Probability,
			Reasons:      []string{"at least one classifier emitted a signal"},
		}
	}

	seenBelowThreshold := false
	seenNoFeatures := false

	for _, r := range classifierResults {
		switch r.Decision.State {
		case domain.DecisionRejectedBelowThreshold:
			seenBelowThreshold = true
		case domain.DecisionRejectedNoFeatures:
			seenNoFeatures = true
		}
	}

	switch {
	case seenBelowThreshold:
		return domain.ClassificationDecision{
			State:    domain.DecisionRejectedBelowThreshold,
			Accepted: false,
			Reasons:  []string{"one or more classifiers scored below threshold and no signal was emitted"},
		}
	case seenNoFeatures:
		return domain.ClassificationDecision{
			State:    domain.DecisionRejectedNoFeatures,
			Accepted: false,
			Reasons:  []string{"one or more classifiers found no usable features and no signal was emitted"},
		}
	default:
		return domain.ClassificationDecision{
			State:    domain.DecisionRejectedNoMatch,
			Accepted: false,
			Reasons:  []string{"no classifier emitted a signal"},
		}
	}
}

func classifySignalDecision(signals []domain.Signal) domain.ClassificationDecision {
	if len(signals) == 0 {
		return domain.ClassificationDecision{
			State:    domain.DecisionRejectedNoMatch,
			Accepted: false,
		}
	}

	top := signals[0]
	for _, s := range signals[1:] {
		if s.Probability > top.Probability {
			top = s
		}
	}

	return domain.ClassificationDecision{
		State:        domain.DecisionAcceptedSignal,
		Accepted:     true,
		PrimaryClass: top.Kind,
		Confidence:   top.Probability,
	}
}

func buildRunID(now time.Time) string {
	return now.UTC().Format(time.RFC3339)
}

func buildAssessmentID(now time.Time, eventID string) string {
	return buildRunID(now) + ":" + eventID
}

func toSignalSnapshots(signals []domain.Signal) []domain.SignalSnapshot {
	if len(signals) == 0 {
		return nil
	}

	out := make([]domain.SignalSnapshot, 0, len(signals))
	for _, s := range signals {
		out = append(out, domain.NewSignalSnapshot(s))
	}
	return out
}
