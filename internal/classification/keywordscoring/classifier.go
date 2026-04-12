package keywordscoring

import (
	"context"
	"strings"
	"time"

	"probability-engine/internal/domain"
)

type Classifier struct{}

func NewClassifier() *Classifier {
	return &Classifier{}
}

func (c *Classifier) Name() string { return "keyword_scoring" }

func (c *Classifier) Classify(ctx context.Context, ev domain.Event, rules []domain.SignalRule) ([]domain.Signal, error) {
	result, err := c.Assess(ctx, ev, rules)
	if err != nil {
		return nil, err
	}
	return result.Signals, nil
}

func (c *Classifier) Assess(ctx context.Context, ev domain.Event, rules []domain.SignalRule) (domain.ClassifierAssessmentResult, error) {
	_ = ctx

	text := strings.ToLower(strings.TrimSpace(ev.Title + "\n" + ev.Summary))

	out := make([]domain.Signal, 0)
	matches := make([]domain.RuleMatchResult, 0)

	for _, rule := range rules {
		if len(rule.WeightedFeatures) == 0 {
			continue
		}

		score := 0.0
		matched := make([]string, 0)

		for _, wf := range rule.WeightedFeatures {
			tok := strings.ToLower(strings.TrimSpace(wf.Token))
			if tok == "" {
				continue
			}
			if strings.Contains(text, tok) {
				score += wf.Weight
				matched = append(matched, tok)
			}
		}

		match := domain.RuleMatchResult{
			RuleID:       rule.Name,
			RuleName:     rule.Name,
			Matched:      score >= rule.Threshold,
			Score:        score,
			MatchedTerms: matched,
		}

		if len(matched) == 0 {
			match.Reason = "no weighted features matched"
		} else if score < rule.Threshold {
			match.Reason = "matched tokens but score below threshold"
		} else {
			match.Reason = "weighted features matched threshold"
		}

		matches = append(matches, match)

		if !match.Matched {
			continue
		}

		now := time.Now().UTC()
		out = append(out, domain.Signal{
			EventID:     ev.ID,
			Kind:        rule.Name,
			MarketScope: append([]string(nil), rule.MarketScope...),
			Direction:   rule.DirectionDefault,
			Probability: score,
			Classifier:  c.Name(),
			Features:    matched,
			Explanation: "matched weighted keyword features",
			Labels:      cloneMap(rule.Labels),
			CreatedAt:   now,
		})
	}

	decision := domain.ClassificationDecision{
		State:    domain.DecisionRejectedNoMatch,
		Accepted: false,
		Reasons:  []string{"no weighted rules matched"},
	}

	if len(out) > 0 {
		decision = domain.ClassificationDecision{
			State:    domain.DecisionAcceptedSignal,
			Accepted: true,
			Reasons:  []string{"weighted features matched threshold"},
		}
	} else {
		for _, m := range matches {
			if len(m.MatchedTerms) > 0 {
				decision.State = domain.DecisionRejectedBelowThreshold
				decision.Reasons = []string{"matched tokens but below threshold"}
				break
			}
		}
	}

	return domain.ClassifierAssessmentResult{
		Classifier: c.Name(),
		Rules: domain.RuleAssessment{
			Evaluated: true,
			Matched:   len(out) > 0,
			Matches:   matches,
		},
		Decision: decision,
		Signals:  out,
	}, nil
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
func extractAssessmentTokens(ev domain.Event) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)

	add := func(text string) {
		for _, part := range strings.Fields(strings.ToLower(text)) {
			token := strings.Trim(part, " \t\r\n,.;:!?()[]{}\"'")
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			out = append(out, token)
		}
	}

	add(ev.Title)
	add(ev.Summary)

	return out
}

func anyRuleBelowThreshold(matches []domain.RuleMatchResult) bool {
	for _, m := range matches {
		if !m.Matched && len(m.MatchedTerms) > 0 {
			return true
		}
	}
	return false
}
