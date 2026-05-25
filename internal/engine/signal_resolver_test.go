package engine

import (
	"testing"
	"time"

	"github.com/galois26/probability-engine/internal/domain"
)

func TestResolveSignals_MergesSameEventKindDirection(t *testing.T) {
	now := time.Now().UTC()

	in := []domain.Signal{
		{
			ID:          "a",
			EventID:     "ev1",
			Kind:        "sanctions",
			Direction:   domain.DirectionNegative,
			Probability: 0.80,
			Classifier:  "rules",
			MarketScope: []string{"fx", "commodities", "equities"},
			Features:    []string{"sanctions"},
			CreatedAt:   now,
			Trace:       domain.SignalTrace{EventID: "ev1", MatchedRules: []string{"sanctions"}},
		},
		{
			ID:          "b",
			EventID:     "ev1",
			Kind:        "sanctions",
			Direction:   domain.DirectionNegative,
			Probability: 0.93,
			Classifier:  "naive_bayes",
			MarketScope: []string{"fx", "commodities"},
			Features:    []string{"sanctions", "export"},
			CreatedAt:   now,
			Trace:       domain.SignalTrace{EventID: "ev1", ModelVersion: "v1"},
		},
	}

	got := resolveSignals(in)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}

	s := got[0]
	if s.EventID != "ev1" || s.Kind != "sanctions" {
		t.Fatalf("unexpected merged signal: %+v", s)
	}
	if s.Probability != 0.93 {
		t.Fatalf("probability = %.2f, want 0.93", s.Probability)
	}
	if s.Classifier != "naive_bayes+rules" && s.Classifier != "rules+naive_bayes" {
		t.Fatalf("classifier = %s", s.Classifier)
	}
	if !sameStrings(s.MarketScope, []string{"commodities", "equities", "fx"}) {
		t.Fatalf("scope = %v", s.MarketScope)
	}
	if !sameStrings(s.Features, []string{"export", "sanctions"}) {
		t.Fatalf("features = %v", s.Features)
	}
}

func TestResolveSignals_DoesNotMergeDifferentKinds(t *testing.T) {
	in := []domain.Signal{
		{ID: "a", EventID: "ev1", Kind: "sanctions", Direction: domain.DirectionNegative},
		{ID: "b", EventID: "ev1", Kind: "conflict_energy", Direction: domain.DirectionNegative},
	}

	got := resolveSignals(in)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]int, len(a))
	for _, v := range a {
		am[v]++
	}
	for _, v := range b {
		am[v]--
	}
	for _, n := range am {
		if n != 0 {
			return false
		}
	}
	return true
}
