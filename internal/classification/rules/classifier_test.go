package rules

import (
	"context"
	"testing"

	"github.com/galois/probability-engine/internal/domain"
	"github.com/galois/probability-engine/internal/testutil"
)

func TestClassifier_Classify(t *testing.T) {
	c := NewClassifier()

	rules := []domain.SignalRule{
		{
			Name:             "sanctions",
			Threshold:        0.65,
			PositiveFeatures: []string{"sanctions", "sanctions package", "financial sanctions", "secondary sanctions", "export ban", "embargo", "sanctioned"},
			NegativeFeatures: []string{"sanctions lifted", "exemptions granted"},
			MarketScope:      []string{"fx", "commodities", "equities"},
			DirectionDefault: domain.DirectionNegative,
		},
		{
			Name:             "supply_chain",
			Threshold:        0.60,
			PositiveFeatures: []string{"minerals", "supply chain", "disruption", "shortages", "bottleneck", "industrial metals"},
			NegativeFeatures: []string{"supply restored", "bottleneck eased"},
			MarketScope:      []string{"metals", "commodities", "industrials"},
			DirectionDefault: domain.DirectionNegative,
		},
		{
			Name:             "conflict_energy",
			Threshold:        0.60,
			PositiveFeatures: []string{"conflict", "energy infrastructure", "attacked", "pipeline", "refinery"},
			NegativeFeatures: []string{"ceasefire", "operations restored"},
			MarketScope:      []string{"oil", "gas", "energy"},
			DirectionDefault: domain.DirectionNegative,
		},
	}

	tests := []struct {
		name      string
		ev        domain.Event
		wantKinds []string
	}{
		{
			name: "sanctions and conflict energy",
			ev: domain.Event{
				ID:      "ev1",
				Title:   "Sanctions imposed on country A after energy infrastructure attack",
				Summary: "New export restrictions and sanctions may affect commodities and FX markets.",
			},
			wantKinds: []string{"conflict_energy", "sanctions"},
		},
		{
			name: "supply chain only",
			ev: domain.Event{
				ID:      "ev2",
				Title:   "Minerals export restrictions announced",
				Summary: "Supply chain disruption risk increases for industrial metals.",
			},
			wantKinds: []string{"supply_chain"},
		},
		{
			name: "conflict energy only",
			ev: domain.Event{
				ID:      "ev3",
				Title:   "Pipeline outage raises concern over regional gas supply",
				Summary: "Conflict near energy infrastructure may disrupt gas markets.",
			},
			wantKinds: []string{"conflict_energy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Classify(context.Background(), tt.ev, rules)
			if err != nil {
				t.Fatalf("Classify() error = %v", err)
			}

			gotKinds := make([]string, 0, len(got))
			for _, s := range got {
				gotKinds = append(gotKinds, s.Kind)
				if s.Probability > 0.92 {
					t.Fatalf("probability exceeded cap: %.2f", s.Probability)
				}
			}

			testutil.SameStrings(t, gotKinds, tt.wantKinds)

		})
	}
}

func TestClassifier_Assess_NoFeatures(t *testing.T) {
	c := NewClassifier()

	result, err := c.Assess(context.Background(), domain.Event{
		ID:      "ev-empty",
		Title:   "",
		Summary: "",
	}, nil)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}

	if result.Classifier != "rules" {
		t.Fatalf("classifier = %q, want %q", result.Classifier, "rules")
	}
	if result.Features.HasFeatures {
		t.Fatalf("HasFeatures = true, want false")
	}
	if result.Decision.State != domain.DecisionRejectedNoFeatures {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionRejectedNoFeatures)
	}
	if result.Rules.Evaluated {
		t.Fatalf("Rules.Evaluated = true, want false")
	}
	if len(result.Signals) != 0 {
		t.Fatalf("signals = %d, want 0", len(result.Signals))
	}
}

func TestClassifier_Assess_RejectedBelowThreshold(t *testing.T) {
	c := NewClassifier()

	rules := []domain.SignalRule{
		{
			Name:             "conflict_energy",
			Threshold:        0.80,
			PositiveFeatures: []string{"pipeline"},
			NegativeFeatures: nil,
			MarketScope:      []string{"oil", "gas", "energy"},
			DirectionDefault: domain.DirectionNegative,
		},
	}

	result, err := c.Assess(context.Background(), domain.Event{
		ID:      "ev-threshold",
		Title:   "Pipeline disruption reported",
		Summary: "",
	}, rules)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}

	if !result.Features.HasFeatures {
		t.Fatalf("HasFeatures = false, want true")
	}
	if !result.Rules.Evaluated {
		t.Fatalf("Rules.Evaluated = false, want true")
	}
	if result.Rules.Matched {
		t.Fatalf("Rules.Matched = true, want false")
	}
	if result.Decision.State != domain.DecisionRejectedBelowThreshold {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionRejectedBelowThreshold)
	}
	if len(result.Rules.Matches) != 1 {
		t.Fatalf("rule matches = %d, want 1", len(result.Rules.Matches))
	}
	if result.Rules.Matches[0].Matched {
		t.Fatalf("match.Matched = true, want false")
	}
	if len(result.Rules.Matches[0].MatchedTerms) == 0 {
		t.Fatalf("matched terms empty, want at least one term")
	}
	if len(result.Signals) != 0 {
		t.Fatalf("signals = %d, want 0", len(result.Signals))
	}
}

func TestClassifier_Assess_AcceptedSignal(t *testing.T) {
	c := NewClassifier()

	rules := []domain.SignalRule{
		{
			Name:             "conflict_energy",
			Threshold:        0.60,
			PositiveFeatures: []string{"pipeline", "refinery"},
			NegativeFeatures: nil,
			MarketScope:      []string{"oil", "gas", "energy"},
			DirectionDefault: domain.DirectionNegative,
		},
	}

	result, err := c.Assess(context.Background(), domain.Event{
		ID:      "ev-accepted",
		Title:   "Pipeline and refinery outage reported",
		Summary: "Pipeline disruption spreads across refinery network.",
	}, rules)
	if err != nil {
		t.Fatalf("Assess() error = %v", err)
	}

	if result.Classifier != "rules" {
		t.Fatalf("classifier = %q, want %q", result.Classifier, "rules")
	}
	if !result.Features.HasFeatures {
		t.Fatalf("HasFeatures = false, want true")
	}
	if !result.Rules.Evaluated {
		t.Fatalf("Rules.Evaluated = false, want true")
	}
	if !result.Rules.Matched {
		t.Fatalf("Rules.Matched = false, want true")
	}
	if result.Decision.State != domain.DecisionAcceptedSignal {
		t.Fatalf("decision state = %s, want %s", result.Decision.State, domain.DecisionAcceptedSignal)
	}
	if !result.Decision.Accepted {
		t.Fatalf("Decision.Accepted = false, want true")
	}
	if result.Decision.PrimaryClass != "conflict_energy" {
		t.Fatalf("primary class = %q, want %q", result.Decision.PrimaryClass, "conflict_energy")
	}
	if len(result.Signals) != 1 {
		t.Fatalf("signals = %d, want 1", len(result.Signals))
	}
	if result.Signals[0].Kind != "conflict_energy" {
		t.Fatalf("signal kind = %q, want %q", result.Signals[0].Kind, "conflict_energy")
	}
	if result.NaiveBayes.Evaluated {
		t.Fatalf("NaiveBayes.Evaluated = true, want false")
	}
}
