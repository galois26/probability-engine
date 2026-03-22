package main

import (
	"context"
	"log"
	"time"

	agg "probability-engine/internal/aggregation/rolling"
	bayes "probability-engine/internal/classification/bayes"
	feat "probability-engine/internal/classification/features"
	rulecls "probability-engine/internal/classification/rules"
	"probability-engine/internal/domain"
	"probability-engine/internal/engine"
	noop "probability-engine/internal/enrichment/noop"
	memstore "probability-engine/internal/persistence/memory"
	"probability-engine/internal/ports"
	"probability-engine/internal/rules"
	memsrc "probability-engine/internal/source/memory"
)

func main() {
	log.Println("probability-engine starting")

	events := []domain.Event{
		{
			ID:        "ev1",
			Source:    "newsdata",
			Title:     "Sanctions imposed on country A after energy infrastructure attack",
			Summary:   "New export restrictions and sanctions may affect commodities and FX markets.",
			URL:       "https://example.com/1",
			Published: time.Now().UTC().Add(-1 * time.Hour),
			Country:   "AA",
			Labels:    map[string]string{"category": "geopolitics"},
		},
		{
			ID:        "ev2",
			Source:    "gta",
			Title:     "Minerals export restrictions announced",
			Summary:   "Supply chain disruption risk increases for industrial metals.",
			URL:       "https://example.com/2",
			Published: time.Now().UTC().Add(-30 * time.Minute),
			Country:   "BB",
			Labels:    map[string]string{"category": "trade"},
		},
		{
			ID:        "ev3",
			Source:    "coindesk",
			Title:     "Pipeline outage raises concern over regional gas supply",
			Summary:   "Conflict near energy infrastructure may disrupt gas markets.",
			URL:       "https://example.com/3",
			Published: time.Now().UTC().Add(-20 * time.Minute),
			Country:   "CC",
			Labels:    map[string]string{"category": "energy"},
		},
		{
			ID:        "ev4",
			Source:    "newsdata",
			Title:     "Export ban expands after sanctions package",
			Summary:   "New restrictions may deepen commodity market pressure.",
			URL:       "https://example.com/4",
			Published: time.Now().UTC().Add(-10 * time.Minute),
			Country:   "AA",
			Labels:    map[string]string{"category": "geopolitics"},
		},
	}

	source := memsrc.New(events)
	ruleLoader := rules.NewLoader("./rules")
	ruleClassifier := rulecls.NewClassifier()

	bayesModel := bayes.Model{
		Version: "v0",
		Classes: map[string]bayes.ClassModel{
			"sanctions": {
				Prior: 0.4,
				Likelihoods: map[string]float64{
					"sanctions":                  0.8,
					"export":                     0.6,
					"restrictions":               0.7,
					"energy":                     0.5,
					"infrastructure":             0.4,
					"label:category=geopolitics": 0.7,
				},
				MarketScope: []string{"fx", "commodities"},
				Direction:   "negative",
			},
			"supply_chain": {
				Prior: 0.3,
				Likelihoods: map[string]float64{
					"minerals":             0.8,
					"supply":               0.6,
					"chain":                0.5,
					"disruption":           0.7,
					"label:category=trade": 0.6,
				},
				MarketScope: []string{"metals", "commodities"},
				Direction:   "negative",
			},
			"conflict_energy": {
				Prior: 0.3,
				Likelihoods: map[string]float64{
					"conflict":              0.7,
					"energy":                0.7,
					"infrastructure":        0.6,
					"pipeline":              0.7,
					"gas":                   0.6,
					"label:category=energy": 0.7,
				},
				MarketScope: []string{"oil", "gas", "energy"},
				Direction:   "negative",
			},
		},
	}

	bayesClassifier := bayes.NewClassifier(bayesModel, feat.NewDefaultExtractor())

	signalStore := memstore.NewSignalStore()
	insightStore := memstore.NewInsightStore()

	eng := engine.New(engine.Options{
		Source:       source,
		RuleLoader:   ruleLoader,
		Classifiers:  []ports.SignalClassifier{ruleClassifier, bayesClassifier},
		Aggregator:   agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:     noop.New(),
		SignalStore:  signalStore,
		InsightStore: insightStore,
	})

	from := time.Now().UTC().Add(-24 * time.Hour)
	res, err := eng.Run(context.Background(), from)
	if err != nil {
		log.Fatalf("engine run failed: %v", err)
	}

	log.Printf("engine run complete: events=%d signals=%d insights=%d", res.Events, res.Signals, res.Insights)

	for _, s := range signalStore.Snapshot() {
		log.Printf("signal: id=%s kind=%s event=%s classifier=%s prob=%.2f scope=%v",
			s.ID, s.Kind, s.EventID, s.Classifier, s.Probability, s.MarketScope)
	}

	for _, in := range insightStore.Snapshot() {
		log.Printf("insight: id=%s title=%q prob=%.2f severity=%s signals=%d",
			in.ID, in.Title, in.Probability, in.Severity, len(in.Signals))
	}
}
