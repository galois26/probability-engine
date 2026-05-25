package engine

import (
	"context"
	"testing"
	"time"

	agg "github.com/galois26/probability-engine/internal/aggregation/rolling"
	bayes "github.com/galois26/probability-engine/internal/classification/bayes"
	feat "github.com/galois26/probability-engine/internal/classification/features"
	rulecls "github.com/galois26/probability-engine/internal/classification/rules"
	"github.com/galois26/probability-engine/internal/domain"
	noop "github.com/galois26/probability-engine/internal/enrichment/noop"
	memstore "github.com/galois26/probability-engine/internal/persistence/memory"
	"github.com/galois26/probability-engine/internal/ports"
	memsrc "github.com/galois26/probability-engine/internal/source/memory"
	"github.com/galois26/probability-engine/internal/testutil"
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

	assessmentStore := memstore.NewEventAssessmentStore()

	eng := New(Options{
		Source:               source,
		RuleLoader:           ruleLoader,
		Classifiers:          []ports.SignalClassifier{ruleClassifier, bayesClassifier},
		Aggregator:           agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:             noop.New(),
		SignalStore:          signalStore,
		EventAssessmentStore: assessmentStore,
		InsightStore:         insightStore,
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
	assessments := assessmentStore.Snapshot()
	if len(assessments) != 4 {
		t.Fatalf("stored assessments = %d, want 4", len(assessments))
	}

	assertResolvedSignals(t, signals)
	assertInsights(t, insights)
}

func TestEngine_Run_SavesAssessmentsWhenNoSignals(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

	events := []domain.Event{
		{
			ID:        "ev1",
			Source:    "newsdata",
			Title:     "Local weather remains stable",
			Summary:   "No major disruptions reported.",
			URL:       "https://example.com/1",
			Published: now.Add(-1 * time.Hour),
		},
		{
			ID:        "ev2",
			Source:    "newsdata",
			Title:     "Sports event scheduled for next week",
			Summary:   "Regional teams prepare for the tournament.",
			URL:       "https://example.com/2",
			Published: now.Add(-30 * time.Minute),
		},
	}

	source := memsrc.New(events)
	ruleLoader := stubRuleLoader{rules: nil}
	signalStore := memstore.NewSignalStore()
	assessmentStore := memstore.NewEventAssessmentStore()
	insightStore := memstore.NewInsightStore()

	eng := New(Options{
		Source:               source,
		RuleLoader:           ruleLoader,
		Classifiers:          []ports.SignalClassifier{stubSignalClassifier{name: "legacy"}},
		Aggregator:           agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:             noop.New(),
		SignalStore:          signalStore,
		EventAssessmentStore: assessmentStore,
		InsightStore:         insightStore,
		Clock:                fixedClock{t: now},
	})

	got, err := eng.Run(context.Background(), now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got.Events != 2 {
		t.Fatalf("Events = %d, want 2", got.Events)
	}
	if got.Signals != 0 {
		t.Fatalf("Signals = %d, want 0", got.Signals)
	}
	if got.Insights != 0 {
		t.Fatalf("Insights = %d, want 0", got.Insights)
	}

	assessments := assessmentStore.Snapshot()
	if len(assessments) != 2 {
		t.Fatalf("stored assessments = %d, want 2", len(assessments))
	}

	for _, a := range assessments {
		if a.Decision.Accepted {
			t.Fatalf("assessment for event %s unexpectedly accepted", a.Event.ID)
		}
		if a.Decision.State != domain.DecisionRejectedNoMatch {
			t.Fatalf("assessment state = %s, want %s", a.Decision.State, domain.DecisionRejectedNoMatch)
		}
		if len(a.Classifiers) != 1 {
			t.Fatalf("classifiers = %d, want 1", len(a.Classifiers))
		}
	}
}

func TestEngine_Run_UsesAssessingClassifierResults(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

	ev := domain.Event{
		ID:        "ev1",
		Source:    "newsdata",
		Title:     "Sanctions announced on key exports",
		Summary:   "Government expands export ban and sanctions package.",
		URL:       "https://example.com/1",
		Published: now.Add(-1 * time.Hour),
	}

	sig := domain.Signal{
		ID:          "sig1",
		EventID:     ev.ID,
		Kind:        "sanctions",
		MarketScope: []string{"fx", "commodities"},
		Direction:   domain.DirectionNegative,
		Probability: 0.81,
		Classifier:  "naive_bayes",
		Features:    []string{"sanctions", "export", "ban"},
		CreatedAt:   now,
		Trace: domain.SignalTrace{
			EventID:      ev.ID,
			ModelVersion: "test-v1",
		},
	}

	classifierResult := domain.ClassifierAssessmentResult{
		Classifier: "naive_bayes",
		Features: domain.FeatureAssessment{
			HasFeatures: true,
			Tokens:      []string{"sanctions", "export", "ban"},
			Keywords:    []string{"sanctions", "export", "ban"},
			Reason:      "feature extraction succeeded",
		},
		Rules: domain.RuleAssessment{
			Evaluated: false,
			Matched:   false,
			Reason:    "not applicable for naive bayes classifier",
		},
		NaiveBayes: domain.NaiveBayesAssessment{
			Evaluated:      true,
			PredictedClass: "sanctions",
			Scores: []domain.ClassScore{
				{Class: "sanctions", Score: -1.2, Probability: 0.81},
				{Class: "supply_chain", Score: -3.4, Probability: 0.11},
			},
		},
		Decision: domain.ClassificationDecision{
			State:        domain.DecisionAcceptedSignal,
			Accepted:     true,
			PrimaryClass: "sanctions",
			Confidence:   0.81,
			Threshold:    0.63,
		},
		Signals: []domain.Signal{sig},
	}

	source := memsrc.New([]domain.Event{ev})
	assessmentStore := memstore.NewEventAssessmentStore()

	eng := New(Options{
		Source:               source,
		RuleLoader:           stubRuleLoader{},
		Classifiers:          []ports.SignalClassifier{stubAssessingClassifier{name: "naive_bayes", result: classifierResult}},
		Aggregator:           agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:             noop.New(),
		SignalStore:          memstore.NewSignalStore(),
		EventAssessmentStore: assessmentStore,
		InsightStore:         memstore.NewInsightStore(),
		Clock:                fixedClock{t: now},
	})

	_, err := eng.Run(context.Background(), now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assessments := assessmentStore.Snapshot()
	if len(assessments) != 1 {
		t.Fatalf("stored assessments = %d, want 1", len(assessments))
	}

	a := assessments[0]
	if a.Decision.State != domain.DecisionAcceptedSignal {
		t.Fatalf("decision state = %s, want %s", a.Decision.State, domain.DecisionAcceptedSignal)
	}
	if len(a.Signals) != 1 {
		t.Fatalf("signals = %d, want 1", len(a.Signals))
	}
	if len(a.Classifiers) != 1 {
		t.Fatalf("classifier results = %d, want 1", len(a.Classifiers))
	}
	if a.Classifiers[0].Classifier != "naive_bayes" {
		t.Fatalf("classifier = %s, want naive_bayes", a.Classifiers[0].Classifier)
	}
	if !a.Classifiers[0].NaiveBayes.Evaluated {
		t.Fatalf("expected naive bayes evaluation")
	}
	if a.Classifiers[0].NaiveBayes.PredictedClass != "sanctions" {
		t.Fatalf("predicted class = %s, want sanctions", a.Classifiers[0].NaiveBayes.PredictedClass)
	}
}

func TestEngine_Run_LegacyClassifierGetsFallbackAssessment(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

	ev := domain.Event{
		ID:        "ev1",
		Source:    "newsdata",
		Title:     "Pipeline outage raises concern",
		Summary:   "Energy infrastructure disruption reported.",
		URL:       "https://example.com/1",
		Published: now.Add(-1 * time.Hour),
	}

	sig := domain.Signal{
		ID:          "sig1",
		EventID:     ev.ID,
		Kind:        "conflict_energy",
		MarketScope: []string{"energy"},
		Direction:   domain.DirectionNegative,
		Probability: 0.72,
		Classifier:  "legacy_rules",
		CreatedAt:   now,
	}

	assessmentStore := memstore.NewEventAssessmentStore()

	eng := New(Options{
		Source:               memsrc.New([]domain.Event{ev}),
		RuleLoader:           stubRuleLoader{},
		Classifiers:          []ports.SignalClassifier{stubSignalClassifier{name: "legacy_rules", signals: []domain.Signal{sig}}},
		Aggregator:           agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:             noop.New(),
		SignalStore:          memstore.NewSignalStore(),
		EventAssessmentStore: assessmentStore,
		InsightStore:         memstore.NewInsightStore(),
		Clock:                fixedClock{t: now},
	})

	_, err := eng.Run(context.Background(), now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assessments := assessmentStore.Snapshot()
	if len(assessments) != 1 {
		t.Fatalf("stored assessments = %d, want 1", len(assessments))
	}

	a := assessments[0]
	if len(a.Classifiers) != 1 {
		t.Fatalf("classifier results = %d, want 1", len(a.Classifiers))
	}
	if a.Classifiers[0].Classifier != "legacy_rules" {
		t.Fatalf("classifier = %s, want legacy_rules", a.Classifiers[0].Classifier)
	}
	if a.Classifiers[0].Features.HasFeatures {
		t.Fatalf("expected fallback classifier features.HasFeatures=false")
	}
	if a.Classifiers[0].Decision.State != domain.DecisionAcceptedSignal {
		t.Fatalf("decision state = %s, want %s", a.Classifiers[0].Decision.State, domain.DecisionAcceptedSignal)
	}
	if len(a.Signals) != 1 {
		t.Fatalf("signals = %d, want 1", len(a.Signals))
	}
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

type fixedClock struct {
	t time.Time
}

func (f fixedClock) Now() time.Time { return f.t }

type stubAssessingClassifier struct {
	name   string
	result domain.ClassifierAssessmentResult
}

func (s stubAssessingClassifier) Name() string { return s.name }

func (s stubAssessingClassifier) Classify(ctx context.Context, ev domain.Event, rules []domain.SignalRule) ([]domain.Signal, error) {
	return append([]domain.Signal(nil), s.result.Signals...), nil
}

func (s stubAssessingClassifier) Assess(ctx context.Context, ev domain.Event, rules []domain.SignalRule) (domain.ClassifierAssessmentResult, error) {
	return s.result, nil
}

type stubSignalClassifier struct {
	name    string
	signals []domain.Signal
}

func (s stubSignalClassifier) Name() string { return s.name }

func (s stubSignalClassifier) Classify(ctx context.Context, ev domain.Event, rules []domain.SignalRule) ([]domain.Signal, error) {
	return append([]domain.Signal(nil), s.signals...), nil
}
