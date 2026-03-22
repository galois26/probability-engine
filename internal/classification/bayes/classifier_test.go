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
