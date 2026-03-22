package engine

import (
	"context"
	"testing"
	"time"

	agg "probability-engine/internal/aggregation/rolling"
	bayes "probability-engine/internal/classification/bayes"
	feat "probability-engine/internal/classification/features"
	rulecls "probability-engine/internal/classification/rules"
	"probability-engine/internal/domain"
	noop "probability-engine/internal/enrichment/noop"
	memstore "probability-engine/internal/persistence/memory"
	"probability-engine/internal/ports"
	memsrc "probability-engine/internal/source/memory"
	"probability-engine/internal/testutil"
)

func TestEngine_Run_EndToEnd(t *testing.T) {
	now := time.Now().UTC()

	events := []domain.Event{
		{
			ID:        "ev1",
			Source:    "newsdata",
			Title:     "Sanctions imposed on country A after energy infrastructure attack",
			Summary:   "New export restrictions and sanctions may affect commodities and FX markets.",
			URL:       "https://example.com/1",
			Published: now.Add(-1 * time.Hour),
			Country:   "AA",
			Labels:    map[string]string{"category": "geopolitics"},
		},
		{
			ID:        "ev2",
			Source:    "gta",
			Title:     "Minerals export restrictions announced",
			Summary:   "Supply chain disruption risk increases for industrial metals.",
			URL:       "https://example.com/2",
			Published: now.Add(-30 * time.Minute),
			Country:   "BB",
			Labels:    map[string]string{"category": "trade"},
		},
		{
			ID:        "ev3",
			Source:    "coindesk",
			Title:     "Pipeline outage raises concern over regional gas supply",
			Summary:   "Conflict near energy infrastructure may disrupt gas markets.",
			URL:       "https://example.com/3",
			Published: now.Add(-20 * time.Minute),
			Country:   "CC",
			Labels:    map[string]string{"category": "energy"},
		},
		{
			ID:        "ev4",
			Source:    "newsdata",
			Title:     "Export ban expands after sanctions package",
			Summary:   "New restrictions may deepen commodity market pressure.",
			URL:       "https://example.com/4",
			Published: now.Add(-10 * time.Minute),
			Country:   "AA",
			Labels:    map[string]string{"category": "geopolitics"},
		},
	}

	source := memsrc.New(events)
	ruleLoader := stubRuleLoader{
		rules: []domain.SignalRule{
			{
				Name:             "conflict_energy",
				MarketScope:      []string{"oil", "gas", "energy"},
				DirectionDefault: domain.DirectionNegative,
				Threshold:        0.60,
				PositiveFeatures: []string{"conflict", "energy infrastructure", "attacked", "pipeline", "refinery"},
				NegativeFeatures: []string{"ceasefire", "operations restored"},
				Labels:           map[string]string{"category": "energy"},
			},
			{
				Name:             "sanctions",
				MarketScope:      []string{"fx", "commodities", "equities"},
				DirectionDefault: domain.DirectionNegative,
				Threshold:        0.65,
				PositiveFeatures: []string{"sanctions", "sanctions package", "financial sanctions", "secondary sanctions", "export ban", "embargo", "sanctioned"},
				NegativeFeatures: []string{"sanctions lifted", "exemptions granted"},
				Labels:           map[string]string{"category": "geopolitics"},
			},
			{
				Name:             "supply_chain",
				MarketScope:      []string{"metals", "commodities", "industrials"},
				DirectionDefault: domain.DirectionNegative,
				Threshold:        0.60,
				PositiveFeatures: []string{"minerals", "supply chain", "disruption", "shortages", "bottleneck", "industrial metals"},
				NegativeFeatures: []string{"supply restored", "bottleneck eased"},
				Labels:           map[string]string{"category": "trade"},
			},
		},
	}

	ruleClassifier := rulecls.NewClassifier()

	bayesModel := bayes.Model{
		Version: "test-v1",
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
					"industrial":           0.6,
					"metals":               0.7,
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

	eng := New(Options{
		Source:       source,
		RuleLoader:   ruleLoader,
		Classifiers:  []ports.SignalClassifier{ruleClassifier, bayesClassifier},
		Aggregator:   agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:     noop.New(),
		SignalStore:  signalStore,
		InsightStore: insightStore,
	})

	got, err := eng.Run(context.Background(), now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got.Events != 4 {
		t.Fatalf("Events = %d, want 4", got.Events)
	}
	if got.Signals != 5 {
		t.Fatalf("Signals = %d, want 5", got.Signals)
	}
	if got.Insights != 3 {
		t.Fatalf("Insights = %d, want 3", got.Insights)
	}

	signals := signalStore.Snapshot()
	if len(signals) != 5 {
		t.Fatalf("stored signals = %d, want 5", len(signals))
	}

	insights := insightStore.Snapshot()
	if len(insights) != 3 {
		t.Fatalf("stored insights = %d, want 3", len(insights))
	}

	assertResolvedSignals(t, signals)
	assertInsights(t, insights)
}

type stubRuleLoader struct {
	rules []domain.SignalRule
}

func (s stubRuleLoader) LoadSignalRules(ctx context.Context) ([]domain.SignalRule, error) {
	return append([]domain.SignalRule(nil), s.rules...), nil
}

func assertResolvedSignals(t *testing.T, signals []domain.Signal) {
	t.Helper()

	gotKindsByEvent := map[string][]string{}
	for _, s := range signals {
		gotKindsByEvent[s.EventID] = append(gotKindsByEvent[s.EventID], s.Kind)
	}

	testutil.SameStrings(t, gotKindsByEvent["ev1"], []string{"conflict_energy", "sanctions"})
	testutil.SameStrings(t, gotKindsByEvent["ev2"], []string{"supply_chain"})
	testutil.SameStrings(t, gotKindsByEvent["ev3"], []string{"conflict_energy"})
	testutil.SameStrings(t, gotKindsByEvent["ev4"], []string{"sanctions"})

	for _, s := range signals {
		switch {
		case s.EventID == "ev1" && s.Kind == "sanctions":
			if s.Classifier != "naive_bayes+rules" && s.Classifier != "rules+naive_bayes" {
				t.Fatalf("ev1 sanctions classifier = %q, want merged classifier", s.Classifier)
			}
			testutil.SameStrings(t, s.MarketScope, []string{"commodities", "equities", "fx"})
		case s.EventID == "ev2" && s.Kind == "supply_chain":
			if s.Classifier != "naive_bayes+rules" && s.Classifier != "rules+naive_bayes" {
				t.Fatalf("ev2 supply_chain classifier = %q, want merged classifier", s.Classifier)
			}
		case s.EventID == "ev3" && s.Kind == "conflict_energy":
			if s.Probability <= 0 || s.Probability >= 1.0 {
				t.Fatalf("ev3 conflict_energy probability = %.4f, want > 0 and < 1.0", s.Probability)
			}
		}
	}
}

func assertInsights(t *testing.T, insights []domain.Insight) {
	t.Helper()

	byPrefix := map[string]domain.Insight{}
	for _, in := range insights {
		switch {
		case len(in.Title) >= len("sanctions") && in.Title[:len("sanctions")] == "sanctions":
			byPrefix["sanctions"] = in
		case len(in.Title) >= len("conflict_energy") && in.Title[:len("conflict_energy")] == "conflict_energy":
			byPrefix["conflict_energy"] = in
		case len(in.Title) >= len("supply_chain") && in.Title[:len("supply_chain")] == "supply_chain":
			byPrefix["supply_chain"] = in
		}
	}

	if len(byPrefix) != 3 {
		t.Fatalf("insight kinds found = %d, want 3", len(byPrefix))
	}

	sanctions := byPrefix["sanctions"]
	if len(sanctions.Signals) != 2 {
		t.Fatalf("sanctions signals = %d, want 2", len(sanctions.Signals))
	}
	testutil.SameStrings(t, sanctions.MarketScope, []string{"commodities", "equities", "fx"})

	conflict := byPrefix["conflict_energy"]
	if len(conflict.Signals) != 2 {
		t.Fatalf("conflict_energy signals = %d, want 2", len(conflict.Signals))
	}
	testutil.SameStrings(t, conflict.MarketScope, []string{"energy", "gas", "oil"})

	supply := byPrefix["supply_chain"]
	if len(supply.Signals) != 1 {
		t.Fatalf("supply_chain signals = %d, want 1", len(supply.Signals))
	}
	if supply.Severity != domain.SeverityMedium && supply.Severity != domain.SeverityHigh {
		t.Fatalf("supply_chain severity = %s, want medium or high", supply.Severity)
	}
}
