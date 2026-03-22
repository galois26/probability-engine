package rules

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	"probability-engine/internal/domain"
)

type Classifier struct{}

func NewClassifier() *Classifier {
	return &Classifier{}
}

func (c *Classifier) Name() string { return "rules" }

func (c *Classifier) Classify(ctx context.Context, ev domain.Event, rules []domain.SignalRule) ([]domain.Signal, error) {
	_ = ctx

	text := strings.ToLower(ev.Title + "\n" + ev.Summary)

	out := make([]domain.Signal, 0, len(rules))

	for _, rule := range rules {
		matchedTerms := matchTerms(text, rule.PositiveFeatures)
		if len(matchedTerms) == 0 {
			continue
		}

		score := scoreRuleMatch(text, rule)
		if score < rule.Threshold {
			continue
		}

		now := time.Now().UTC()
		out = append(out, domain.Signal{
			ID:          buildSignalID(ev.ID, rule.Name, "rules"),
			EventID:     ev.ID,
			Kind:        rule.Name,
			MarketScope: append([]string(nil), rule.MarketScope...),
			Direction:   rule.DirectionDefault,
			Probability: score,
			Classifier:  c.Name(),
			Features:    matchedTerms,
			Explanation: "matched rule features in title/summary",
			Labels:      cloneMap(rule.Labels),
			CreatedAt:   now,
			Trace: domain.SignalTrace{
				EventID:      ev.ID,
				MatchedRules: []string{rule.Name},
				MatchedTerms: matchedTerms,
			},
		})
	}

	return out, nil
}

func matchTerms(text string, terms []string) []string {
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		tt := strings.ToLower(strings.TrimSpace(t))
		if tt == "" {
			continue
		}
		if strings.Contains(text, tt) {
			out = append(out, tt)
		}
	}
	return out
}

func scoreRuleMatch(text string, rule domain.SignalRule) float64 {
	pos := 0
	for _, f := range rule.PositiveFeatures {
		ff := strings.ToLower(strings.TrimSpace(f))
		if ff != "" && strings.Contains(text, ff) {
			pos++
		}
	}

	neg := 0
	for _, f := range rule.NegativeFeatures {
		ff := strings.ToLower(strings.TrimSpace(f))
		if ff != "" && strings.Contains(text, ff) {
			neg++
		}
	}

	if pos == 0 {
		return 0
	}

	score := 0.5 + 0.15*float64(pos) - 0.10*float64(neg)
	if score < 0 {
		score = 0
	}
	if score > 0.92 {
		score = 0.92
	}
	return score
}

func buildSignalID(eventID, ruleName, classifier string) string {
	h := sha1.New()
	h.Write([]byte(eventID))
	h.Write([]byte("|"))
	h.Write([]byte(ruleName))
	h.Write([]byte("|"))
	h.Write([]byte(classifier))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
