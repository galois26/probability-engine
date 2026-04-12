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
	source                   ports.EventSource
	ruleLoader               ports.RuleLoader
	classifiers              []ports.SignalClassifier
	aggregator               ports.InsightAggregator
	enricher                 ports.InsightEnricher
	signalStore              ports.SignalStore
	eventAssessmentStore     ports.EventAssessmentStore
	eventAssessmentPublisher ports.EventAssessmentPublisher
	insightStore             ports.InsightStore
	clock                    Clock
}

type Options struct {
	Source                   ports.EventSource
	RuleLoader               ports.RuleLoader
	Classifiers              []ports.SignalClassifier
	Aggregator               ports.InsightAggregator
	Enricher                 ports.InsightEnricher
	SignalStore              ports.SignalStore
	EventAssessmentStore     ports.EventAssessmentStore
	EventAssessmentPublisher ports.EventAssessmentPublisher
	InsightStore             ports.InsightStore
	Clock                    Clock
}

func New(opts Options) *Engine {
	c := opts.Clock
	if c == nil {
		c = realClock{}
	}

	return &Engine{
		source:                   opts.Source,
		ruleLoader:               opts.RuleLoader,
		classifiers:              opts.Classifiers,
		aggregator:               opts.Aggregator,
		enricher:                 opts.Enricher,
		signalStore:              opts.SignalStore,
		eventAssessmentStore:     opts.EventAssessmentStore,
		eventAssessmentPublisher: opts.EventAssessmentPublisher,
		insightStore:             opts.InsightStore,
		clock:                    c,
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
		classifierResults := make([]domain.ClassifierAssessmentResult, 0, len(e.classifiers))

		for _, classifier := range e.classifiers {
			if assessing, ok := classifier.(ports.AssessingSignalClassifier); ok {
				assessmentResult, err := assessing.Assess(ctx, ev, rules)
				if err != nil {
					return result, err
				}

				log.Printf(
					"engine: classifier=%s event=%s assessed accepted=%t signals=%d",
					classifier.Name(),
					ev.ID,
					assessmentResult.Decision.Accepted,
					len(assessmentResult.Signals),
				)

				classifierResults = append(classifierResults, assessmentResult)
				eventSignals = append(eventSignals, assessmentResult.Signals...)
				rawSignals = append(rawSignals, assessmentResult.Signals...)
				continue
			}

			signals, err := classifier.Classify(ctx, ev, rules)
			if err != nil {
				return result, err
			}

			log.Printf(
				"engine: classifier=%s event=%s produced %d signals (no assessment)",
				classifier.Name(),
				ev.ID,
				len(signals),
			)

			classifierResults = append(classifierResults, fallbackClassifierAssessmentResult(classifier.Name(), signals))
			eventSignals = append(eventSignals, signals...)
			rawSignals = append(rawSignals, signals...)
		}

		assessment := buildEventAssessment(now, ev, classifierResults, eventSignals)
		assessments = append(assessments, assessment)

		log.Printf(
			"engine: assessment event=%s state=%s accepted=%t signals=%d classifiers=%d",
			ev.ID,
			assessment.Decision.State,
			assessment.Decision.Accepted,
			len(assessment.Signals),
			len(assessment.Classifiers),
		)
	}

	log.Printf("engine: raw signals=%d", len(rawSignals))
	log.Printf("engine: event assessments=%d", len(assessments))

	if e.eventAssessmentStore != nil {
		if err := e.eventAssessmentStore.SaveEventAssessments(ctx, assessments); err != nil {
			return result, err
		}
	}

	if e.eventAssessmentPublisher != nil {
		if err := e.eventAssessmentPublisher.PublishEventAssessments(ctx, assessments); err != nil {
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

	acceptedCount := 0
	rejectedCount := 0
	decisionCounts := make(map[domain.DecisionState]int)

	for _, a := range assessments {
		if a.Decision.Accepted {
			acceptedCount++
		} else {
			rejectedCount++
		}
		decisionCounts[a.Decision.State]++
	}

	log.Printf(
		"engine metrics: events=%d assessments=%d accepted=%d rejected=%d raw_signals=%d resolved_signals=%d insights=%d states=%v",
		len(events),
		len(assessments),
		acceptedCount,
		rejectedCount,
		len(rawSignals),
		len(resolvedSignals),
		len(insights),
		decisionCounts,
	)

	result.Insights = len(insights)

	return result, nil
}
