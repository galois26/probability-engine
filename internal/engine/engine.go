package engine

import (
	"context"
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

	eventsByID := make(map[string]domain.Event, len(events))
	allSignals := make([]domain.Signal, 0, len(events))

	for _, ev := range events {
		eventsByID[ev.ID] = ev

		for _, classifier := range e.classifiers {
			signals, err := classifier.Classify(ctx, ev, rules)
			if err != nil {
				return result, err
			}
			allSignals = append(allSignals, signals...)
		}
	}

	if e.signalStore != nil && len(allSignals) > 0 {
		if err := e.signalStore.SaveSignals(ctx, allSignals); err != nil {
			return result, err
		}
	}
	result.Signals = len(allSignals)

	insights, err := e.aggregator.Aggregate(ctx, allSignals, eventsByID)
	if err != nil {
		return result, err
	}

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

	if e.insightStore != nil && len(insights) > 0 {
		if err := e.insightStore.SaveInsights(ctx, insights); err != nil {
			return result, err
		}
	}
	result.Insights = len(insights)

	return result, nil
}
