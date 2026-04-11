package bayes

import (
	"context"
	"probability-engine/internal/domain"
	"testing"
)

func TestCalibratedProbability_NotOne(t *testing.T) {
	scores := []scored{
		{class: "sanctions", score: -2.0},
		{class: "supply_chain", score: -4.0},
		{class: "conflict_energy", score: -5.0},
	}

	got := calibratedProbability(scores, "sanctions")
	if got <= 0 {
		t.Fatalf("got non-positive probability: %v", got)
	}
	if got >= 1.0 {
		t.Fatalf("got probability %v, want < 1.0", got)
	}
}

type stubExtractor struct {
	tokens []string
}

func (s stubExtractor) Extract(ev domain.Event) []string {
	return append([]string(nil), s.tokens...)
}

func TestClassifier_DedupesTokens(t *testing.T) {
	model := Model{
		Version: "test",
		Classes: map[string]ClassModel{
			"sanctions": {
				Prior: 0.5,
				Likelihoods: map[string]float64{
					"sanctions": 0.8,
					"export":    0.7,
				},
				MarketScope: []string{"fx"},
				Direction:   "negative",
			},
			"supply_chain": {
				Prior: 0.5,
				Likelihoods: map[string]float64{
					"minerals": 0.7,
				},
				MarketScope: []string{"metals"},
				Direction:   "negative",
			},
		},
	}

	c := NewClassifier(model, stubExtractor{
		tokens: []string{"sanctions", "sanctions", "sanctions", "export"},
	})

	got, err := c.Classify(context.Background(), domain.Event{ID: "ev1"}, nil)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Kind != "sanctions" {
		t.Fatalf("kind = %s, want sanctions", got[0].Kind)
	}
	if got[0].Probability >= 1.0 {
		t.Fatalf("probability = %.4f, want < 1.0", got[0].Probability)
	}
}

func TestClassifier_Assess_NoClasses(t *testing.T) {
	c := NewClassifier(Model{}, stubExtractor{
		tokens: []string{"sanctions"},
	})

	result, err := c.Assess(context.Background(), domain.Event{ID: "ev1"}, nil)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}

	if result.Classifier != "naive_bayes" {
		t.Fatalf("classifier = %q, want %q", result.Classifier, "naive_bayes")
	}
	if result.Decision.Accepted {
		t.Fatalf("Decision.Accepted = true, want false")
	}
	if result.NaiveBayes.Evaluated {
		t.Fatalf("NaiveBayes.Evaluated = true, want false")
	}
	if result.Decision.State != domain.DecisionRejectedNoMatch {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionRejectedNoMatch)
	}
	if len(result.Signals) != 0 {
		t.Fatalf("signals = %d, want 0", len(result.Signals))
	}
}

func TestClassifier_Assess_NoTokens(t *testing.T) {
	model := Model{
		Version: "test",
		Classes: map[string]ClassModel{
			"sanctions": {
				Prior: 0.5,
				Likelihoods: map[string]float64{
					"sanctions": 0.8,
				},
				MarketScope: []string{"fx"},
				Direction:   "negative",
			},
		},
	}

	c := NewClassifier(model, stubExtractor{
		tokens: nil,
	})

	result, err := c.Assess(context.Background(), domain.Event{ID: "ev1"}, nil)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}

	if result.Features.HasFeatures {
		t.Fatalf("HasFeatures = true, want false")
	}
	if result.Decision.State != domain.DecisionRejectedNoFeatures {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionRejectedNoFeatures)
	}
	if result.NaiveBayes.Evaluated {
		t.Fatalf("NaiveBayes.Evaluated = true, want false")
	}
	if len(result.Signals) != 0 {
		t.Fatalf("signals = %d, want 0", len(result.Signals))
	}
}

func TestClassifier_Assess_RejectedNoMatch_WhenClassGateFails(t *testing.T) {
	model := Model{
		Version: "test",
		Classes: map[string]ClassModel{
			"supply_chain": {
				Prior: 1.0,
				Likelihoods: map[string]float64{
					"exports": 0.9,
					"policy":  0.8,
				},
				MarketScope: []string{"metals"},
				Direction:   "negative",
			},
			"sanctions": {
				Prior: 0.1,
				Likelihoods: map[string]float64{
					"embargo": 0.1,
				},
				MarketScope: []string{"fx"},
				Direction:   "negative",
			},
		},
	}

	c := NewClassifier(model, stubExtractor{
		tokens: []string{"exports", "policy"},
	})

	result, err := c.Assess(context.Background(), domain.Event{
		ID:     "ev1",
		Source: "newsdata",
	}, nil)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}

	if !result.Features.HasFeatures {
		t.Fatalf("HasFeatures = false, want true")
	}
	if !result.NaiveBayes.Evaluated {
		t.Fatalf("NaiveBayes.Evaluated = false, want true")
	}
	if result.NaiveBayes.PredictedClass != "supply_chain" {
		t.Fatalf("predicted class = %q, want %q", result.NaiveBayes.PredictedClass, "supply_chain")
	}
	if result.Decision.State != domain.DecisionRejectedNoMatch {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionRejectedNoMatch)
	}
	if len(result.Signals) != 0 {
		t.Fatalf("signals = %d, want 0", len(result.Signals))
	}
}

func TestClassifier_Assess_RejectedBelowThreshold(t *testing.T) {
	model := Model{
		Version: "test",
		Classes: map[string]ClassModel{
			"sanctions": {
				Prior: 0.51,
				Likelihoods: map[string]float64{
					"sanctions": 0.10,
					"export":    0.10,
					"ban":       0.10,
				},
				MarketScope: []string{"fx", "commodities"},
				Direction:   "negative",
			},
			"supply_chain": {
				Prior: 0.49,
				Likelihoods: map[string]float64{
					"sanctions": 0.09,
					"export":    0.09,
					"ban":       0.09,
				},
				MarketScope: []string{"metals"},
				Direction:   "negative",
			},
		},
	}

	c := NewClassifier(model, stubExtractor{
		tokens: []string{"sanctions", "export", "ban"},
	})

	result, err := c.Assess(context.Background(), domain.Event{
		ID:     "ev1",
		Source: "newsdata",
		Labels: map[string]string{
			"source_id": "techbullion",
		},
	}, nil)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}
	t.Logf("predicted=%s confidence=%.4f threshold=%.4f scores=%+v",
		result.NaiveBayes.PredictedClass,
		result.Decision.Confidence,
		result.Decision.Threshold,
		result.NaiveBayes.Scores,
	)
	if !result.NaiveBayes.Evaluated {
		t.Fatalf("NaiveBayes.Evaluated = false, want true")
	}
	if result.NaiveBayes.PredictedClass != "sanctions" {
		t.Fatalf("predicted class = %q, want %q", result.NaiveBayes.PredictedClass, "sanctions")
	}
	if result.Decision.State != domain.DecisionRejectedBelowThreshold {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionRejectedBelowThreshold)
	}
	if result.Decision.Accepted {
		t.Fatalf("Decision.Accepted = true, want false")
	}
	if result.Decision.Threshold <= 0 {
		t.Fatalf("threshold = %.4f, want > 0", result.Decision.Threshold)
	}
	if len(result.NaiveBayes.Scores) == 0 {
		t.Fatalf("scores empty, want populated")
	}
	if len(result.Signals) != 0 {
		t.Fatalf("signals = %d, want 0", len(result.Signals))
	}
}

func TestClassifier_Assess_AcceptedSignal(t *testing.T) {
	model := Model{
		Version: "test-v1",
		Classes: map[string]ClassModel{
			"sanctions": {
				Prior: 0.8,
				Likelihoods: map[string]float64{
					"sanctions": 0.95,
					"export":    0.90,
					"ban":       0.90,
				},
				MarketScope: []string{"fx", "commodities"},
				Direction:   "negative",
			},
			"supply_chain": {
				Prior: 0.2,
				Likelihoods: map[string]float64{
					"minerals": 0.2,
				},
				MarketScope: []string{"metals"},
				Direction:   "negative",
			},
		},
	}

	c := NewClassifier(model, stubExtractor{
		tokens: []string{"sanctions", "export", "ban"},
	})

	result, err := c.Assess(context.Background(), domain.Event{
		ID:     "ev1",
		Source: "newsdata",
	}, nil)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}

	if result.Classifier != "naive_bayes" {
		t.Fatalf("classifier = %q, want %q", result.Classifier, "naive_bayes")
	}
	if !result.Features.HasFeatures {
		t.Fatalf("HasFeatures = false, want true")
	}
	if !result.NaiveBayes.Evaluated {
		t.Fatalf("NaiveBayes.Evaluated = false, want true")
	}
	if result.NaiveBayes.PredictedClass != "sanctions" {
		t.Fatalf("predicted class = %q, want %q", result.NaiveBayes.PredictedClass, "sanctions")
	}
	if result.Decision.State != domain.DecisionAcceptedSignal {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionAcceptedSignal)
	}
	if !result.Decision.Accepted {
		t.Fatalf("Decision.Accepted = false, want true")
	}
	if len(result.NaiveBayes.Scores) != 2 {
		t.Fatalf("scores = %d, want 2", len(result.NaiveBayes.Scores))
	}
	if len(result.Signals) != 1 {
		t.Fatalf("signals = %d, want 1", len(result.Signals))
	}
	if result.Signals[0].Kind != "sanctions" {
		t.Fatalf("signal kind = %q, want %q", result.Signals[0].Kind, "sanctions")
	}
	if result.Signals[0].Trace.ModelVersion != "test-v1" {
		t.Fatalf("model version = %q, want %q", result.Signals[0].Trace.ModelVersion, "test-v1")
	}
}
