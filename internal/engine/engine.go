package engine

import (
	"context"
	"log"
	"time"

	"probability-engine/internal/domain"
	"probability-engine/internal/ports"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type Engine struct {
	source               ports.EventSource
	ruleLoader           ports.RuleLoader
	classifiers          []ports.SignalClassifier
	aggregator           ports.InsightAggregator
	enricher             ports.InsightEnricher
	signalStore          ports.SignalStore
	eventAssessmentStore ports.EventAssessmentStore
	insightStore         ports.InsightStore
	clock                Clock
}

type Options struct {
	Source               ports.EventSource
	RuleLoader           ports.RuleLoader
	Classifiers          []ports.SignalClassifier
	Aggregator           ports.InsightAggregator
	Enricher             ports.InsightEnricher
	SignalStore          ports.SignalStore
	EventAssessmentStore ports.EventAssessmentStore
	InsightStore         ports.InsightStore
	Clock                Clock
}

func New(opts Options) *Engine {
	c := opts.Clock
	if c == nil {
		c = realClock{}
	}

	return &Engine{
		source:               opts.Source,
		ruleLoader:           opts.RuleLoader,
		classifiers:          opts.Classifiers,
		aggregator:           opts.Aggregator,
		enricher:             opts.Enricher,
		signalStore:          opts.SignalStore,
		eventAssessmentStore: opts.EventAssessmentStore,
		insightStore:         opts.InsightStore,
		clock:                c,
	}
}

type RunResult struct {
	Events   int `json:"events"`
	Signals  int `json:"signals"`
	Insights int `json:"insights"`
}

func (e *Engine) Run(ctx context.Context, from time.Time) (RunResult, error) {
	var result RunResult

	events, err := e.source.FetchEvents(ctx, from)
	if err != nil {
		return result, err
	}
	log.Printf("engine: fetched events=%d from=%s", len(events), from.Format(time.RFC3339))
	result.Events = len(events)

	rules, err := e.ruleLoader.LoadSignalRules(ctx)
	if err != nil {
		return result, err
	}
	for _, r := range rules {
		log.Printf(
			"engine: rule name=%s threshold=%.2f pos=%v neg=%v scope=%v direction=%s",
			r.Name, r.Threshold, r.PositiveFeatures, r.NegativeFeatures, r.MarketScope, r.DirectionDefault,
		)
	}

	now := e.clock.Now()
	eventsByID := make(map[string]domain.Event, len(events))
	rawSignals := make([]domain.Signal, 0, len(events))
	assessments := make([]domain.EventAssessment, 0, len(events))

	for _, ev := range events {
		eventsByID[ev.ID] = ev

		eventSignals := make([]domain.Signal, 0, len(e.classifiers))

		for _, classifier := range e.classifiers {
			signals, err := classifier.Classify(ctx, ev, rules)
			if err != nil {
				return result, err
			}
			log.Printf("engine: classifier=%s event=%s produced %d signals", classifier.Name(), ev.ID, len(signals))

			eventSignals = append(eventSignals, signals...)
			rawSignals = append(rawSignals, signals...)
		}

		assessment := buildEventAssessment(now, ev, eventSignals)
		assessments = append(assessments, assessment)

		log.Printf(
			"engine: assessment event=%s state=%s accepted=%t signals=%d",
			ev.ID,
			assessment.Decision.State,
			assessment.Decision.Accepted,
			len(assessment.Signals),
		)
	}

	log.Printf("engine: raw signals=%d", len(rawSignals))
	log.Printf("engine: event assessments=%d", len(assessments))

	if e.eventAssessmentStore != nil {
		if err := e.eventAssessmentStore.SaveEventAssessments(ctx, assessments); err != nil {
			return result, err
		}
	}

	resolvedSignals := resolveSignals(rawSignals)
	log.Printf("engine: resolved signals=%d", len(resolvedSignals))

	if e.signalStore != nil && len(resolvedSignals) > 0 {
		if err := e.signalStore.SaveSignals(ctx, resolvedSignals); err != nil {
			return result, err
		}
	}
	result.Signals = len(resolvedSignals)

	insights, err := e.aggregator.Aggregate(ctx, resolvedSignals, eventsByID)
	if err != nil {
		return result, err
	}
	log.Printf("engine: aggregated insights=%d", len(insights))

	if e.enricher != nil {
		enriched := make([]domain.Insight, 0, len(insights))
		for _, in := range insights {
			out, err := e.enricher.Enrich(ctx, in)
			if err != nil {
				return result, err
			}
			enriched = append(enriched, out)
		}
		insights = enriched
	}

	for _, s := range resolvedSignals {
		log.Printf("engine: resolved signal kind=%s event=%s classifier=%s prob=%.2f scope=%v",
			s.Kind, s.EventID, s.Classifier, s.Probability, s.MarketScope)
	}

	if e.insightStore != nil && len(insights) > 0 {
		if err := e.insightStore.SaveInsights(ctx, insights); err != nil {
			return result, err
		}
	}
	result.Insights = len(insights)

	return result, nil
}

func buildEventAssessment(now time.Time, ev domain.Event, signals []domain.Signal) domain.EventAssessment {
	decision := classifyEventDecision(signals)

	reasons := []string(nil)
	if decision.Accepted {
		reasons = []string{"at least one classifier emitted a signal"}
	} else {
		reasons = []string{"no classifier emitted a signal"}
	}
	decision.Reasons = reasons

	return domain.EventAssessment{
		ID:         buildAssessmentID(now, ev.ID),
		RunID:      buildRunID(now),
		AssessedAt: now,
		Event:      newEventSnapshot(ev),
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
		Signals:  toSignalSnapshots(signals),
	}
}

func classifyEventDecision(signals []domain.Signal) domain.ClassificationDecision {
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
		Reasons:      nil,
	}
}

func newEventSnapshot(ev domain.Event) domain.EventSnapshot {
	return domain.EventSnapshot{
		ID:        ev.ID,
		Source:    ev.Source,
		Title:     ev.Title,
		Summary:   ev.Summary,
		URL:       ev.URL,
		Published: ev.Published,
		Lang:      ev.Lang,
		Country:   ev.Country,
		Labels:    ev.Labels,
		Raw:       ev.Raw,
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
		out = append(out, domain.SignalSnapshot{
			ID:          s.ID,
			EventID:     s.EventID,
			Kind:        s.Kind,
			MarketScope: s.MarketScope,
			Direction:   s.Direction,
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
