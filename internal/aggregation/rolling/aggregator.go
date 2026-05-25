package rolling

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/galois/probability-engine/internal/domain"
)

type Aggregator struct {
	minSignals           int
	threshold            float64
	window               time.Duration
	singletonProbability float64
}

type bucket struct {
	signals []domain.Signal
	score   float64
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
		minSignals:           minSignals,
		threshold:            threshold,
		window:               window,
		singletonProbability: 0.90,
	}
}

func (a *Aggregator) Aggregate(ctx context.Context, signals []domain.Signal, eventsByID map[string]domain.Event) ([]domain.Insight, error) {
	_ = ctx

	cutoff := time.Now().UTC().Add(-a.window)
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
		if !a.qualifies(b) {
			continue
		}

		sort.Slice(b.signals, func(i, j int) bool {
			return b.signals[i].Probability > b.signals[j].Probability
		})

		first := b.signals[0]
		scope := mergedScope(b.signals)
		now := time.Now().UTC()

		insight := domain.Insight{
			ID:          buildInsightID(key, b.signals),
			Title:       buildTitle(first.Kind, scope, len(b.signals)),
			MarketScope: scope,
			Probability: clampProbability(b.score / float64(len(b.signals))),
			Direction:   first.Direction,
			//		Severity:    severityFromScore(b.score),
			Severity:  severityForBucket(b.signals, b.score),
			Status:    domain.InsightStatusOpen,
			Summary:   buildSummary(first, len(b.signals)),
			Signals:   collectSignalIDs(b.signals),
			Evidence:  buildEvidence(b.signals, eventsByID),
			Flags:     domain.InsightFlags{},
			CreatedAt: now,
			UpdatedAt: now,
		}

		out = append(out, insight)
	}

	return out, nil
}

func severityForBucket(signals []domain.Signal, score float64) domain.Severity {
	if len(signals) == 1 {
		p := signals[0].Probability
		switch {
		case p >= 0.95:
			return domain.SeverityHigh
		case p >= 0.85:
			return domain.SeverityMedium
		default:
			return domain.SeverityLow
		}
	}
	return severityFromScore(score)
}

func (a *Aggregator) qualifies(b *bucket) bool {
	if len(b.signals) >= a.minSignals && b.score >= a.threshold {
		return true
	}
	if len(b.signals) == 1 && b.signals[0].Probability >= a.singletonProbability {
		return true
	}
	return false
}

func buildGroupKey(sig domain.Signal) string {
	return string(sig.Direction) + "|" + sig.Kind
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

func buildTitle(kind string, scope []string, n int) string {
	scopeText := "market"
	if len(scope) > 0 {
		scopeText = strings.Join(scope, ", ")
	}
	if n == 1 {
		return kind + " signal affecting " + scopeText + " (1 signal)"
	}
	return kind + " signal cluster affecting " + scopeText + " (" + strconv.Itoa(n) + " signals)"
}

func buildSummary(sig domain.Signal, n int) string {
	if n == 1 {
		return "A high-confidence signal indicates a potential market-moving development."
	}
	return "Multiple related signals indicate a potential market-moving development."
}

func mergedScope(signals []domain.Signal) []string {
	set := make(map[string]struct{})
	for _, s := range signals {
		for _, sc := range s.MarketScope {
			sc = strings.TrimSpace(sc)
			if sc == "" {
				continue
			}
			set[sc] = struct{}{}
		}
	}

	out := make([]string, 0, len(set))
	for sc := range set {
		out = append(out, sc)
	}
	sort.Strings(out)
	return out
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
			InsightID:   s.ID,
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
