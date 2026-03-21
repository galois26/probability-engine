package rolling

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"

	"probability-engine/internal/domain"
)

type Aggregator struct {
	minSignals int
	threshold  float64
	window     time.Duration
}

func NewAggregator(minSignals int, threshold float64, window time.Duration) *Aggregator {
	if minSignals <= 0 {
		minSignals = 2
	}
	if threshold <= 0 {
		threshold = 1.25
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	return &Aggregator{
		minSignals: minSignals,
		threshold:  threshold,
		window:     window,
	}
}

func (a *Aggregator) Aggregate(ctx context.Context, signals []domain.Signal, eventsByID map[string]domain.Event) ([]domain.Insight, error) {
	_ = ctx

	cutoff := time.Now().UTC().Add(-a.window)

	type bucket struct {
		signals []domain.Signal
		score   float64
	}

	grouped := map[string]*bucket{}

	for _, sig := range signals {
		ev, ok := eventsByID[sig.EventID]
		if !ok {
			continue
		}
		if ev.Published.Before(cutoff) {
			continue
		}

		key := buildGroupKey(sig)
		b := grouped[key]
		if b == nil {
			b = &bucket{}
			grouped[key] = b
		}
		b.signals = append(b.signals, sig)
		b.score += sig.Probability
	}

	out := make([]domain.Insight, 0, len(grouped))

	for key, b := range grouped {
		if len(b.signals) < a.minSignals {
			continue
		}
		if b.score < a.threshold {
			continue
		}

		sort.Slice(b.signals, func(i, j int) bool {
			return b.signals[i].Probability > b.signals[j].Probability
		})

		first := b.signals[0]

		insight := domain.Insight{
			ID:          buildInsightID(key, b.signals),
			Title:       buildTitle(first, len(b.signals)),
			MarketScope: append([]string(nil), first.MarketScope...),
			Probability: clampProbability(b.score / float64(len(b.signals))),
			Direction:   first.Direction,
			Severity:    severityFromScore(b.score),
			Status:      domain.InsightStatusOpen,
			Summary:     buildSummary(first, len(b.signals)),
			Signals:     collectSignalIDs(b.signals),
			Evidence:    buildEvidence(b.signals, eventsByID),
			Flags:       domain.InsightFlags{},
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		out = append(out, insight)
	}

	return out, nil
}

func buildGroupKey(sig domain.Signal) string {
	scope := append([]string(nil), sig.MarketScope...)
	sort.Strings(scope)
	return strings.Join(scope, ",") + "|" + string(sig.Direction) + "|" + sig.Kind
}

func buildInsightID(groupKey string, signals []domain.Signal) string {
	h := sha1.New()
	h.Write([]byte(groupKey))
	for _, s := range signals {
		h.Write([]byte("|"))
		h.Write([]byte(s.ID))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func buildTitle(sig domain.Signal, n int) string {
	scope := "market"
	if len(sig.MarketScope) > 0 {
		scope = strings.Join(sig.MarketScope, ", ")
	}
	return sig.Kind + " signal cluster affecting " + scope + " (" + strconv.Itoa(n) + " signals)"
}

func buildSummary(sig domain.Signal, n int) string {
	_ = n
	return "Multiple related signals indicate a potential market-moving development."
}

func collectSignalIDs(signals []domain.Signal) []string {
	out := make([]string, 0, len(signals))
	for _, s := range signals {
		out = append(out, s.ID)
	}
	return out
}

func buildEvidence(signals []domain.Signal, eventsByID map[string]domain.Event) []domain.InsightEvidence {
	out := make([]domain.InsightEvidence, 0, len(signals))
	for _, s := range signals {
		ev, ok := eventsByID[s.EventID]
		if !ok {
			continue
		}
		out = append(out, domain.InsightEvidence{
			SignalID:    s.ID,
			EventID:     ev.ID,
			Source:      ev.Source,
			URL:         ev.URL,
			Published:   ev.Published,
			WhyIncluded: s.Explanation,
		})
	}
	return out
}

func severityFromScore(score float64) domain.Severity {
	switch {
	case score >= 3.0:
		return domain.SeverityCritical
	case score >= 2.0:
		return domain.SeverityHigh
	case score >= 1.25:
		return domain.SeverityMedium
	default:
		return domain.SeverityLow
	}
}

func clampProbability(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
