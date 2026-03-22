package rules

import (
	"context"
	"probability-engine/internal/domain"
	"probability-engine/internal/testutil"
	"testing"
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
