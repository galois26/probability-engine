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
	source       ports.EventSource
	ruleLoader   ports.RuleLoader
	classifiers  []ports.SignalClassifier
	aggregator   ports.InsightAggregator
	enricher     ports.InsightEnricher
	signalStore  ports.SignalStore
	insightStore ports.InsightStore
	clock        Clock
}

type Options struct {
	Source       ports.EventSource
	RuleLoader   ports.RuleLoader
	Classifiers  []ports.SignalClassifier
	Aggregator   ports.InsightAggregator
	Enricher     ports.InsightEnricher
	SignalStore  ports.SignalStore
	InsightStore ports.InsightStore
	Clock        Clock
}

func New(opts Options) *Engine {
	c := opts.Clock
	if c == nil {
		c = realClock{}
	}

	return &Engine{
		source:       opts.Source,
		ruleLoader:   opts.RuleLoader,
		classifiers:  opts.Classifiers,
		aggregator:   opts.Aggregator,
		enricher:     opts.Enricher,
		signalStore:  opts.SignalStore,
		insightStore: opts.InsightStore,
		clock:        c,
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
	eventsByID := make(map[string]domain.Event, len(events))
	rawSignals := make([]domain.Signal, 0, len(events))

	for _, ev := range events {
		eventsByID[ev.ID] = ev

		for _, classifier := range e.classifiers {
			signals, err := classifier.Classify(ctx, ev, rules)
			if err != nil {
				return result, err
			}
			log.Printf("engine: classifier=%s event=%s produced %d signals", classifier.Name(), ev.ID, len(signals))
			rawSignals = append(rawSignals, signals...)
		}
	}
	log.Printf("engine: raw signals=%d", len(rawSignals))
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
