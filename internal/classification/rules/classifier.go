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
	result, err := c.Assess(ctx, ev, rules)
	if err != nil {
		return nil, err
	}
	return result.Signals, nil
}

func (c *Classifier) Assess(ctx context.Context, ev domain.Event, rules []domain.SignalRule) (domain.ClassifierAssessmentResult, error) {
	_ = ctx

	text := strings.ToLower(strings.TrimSpace(ev.Title + "\n" + ev.Summary))
	tokens := extractRuleAssessmentTokens(ev)

	if len(tokens) == 0 {
		return domain.ClassifierAssessmentResult{
			Classifier: c.Name(),
			Features: domain.FeatureAssessment{
				HasFeatures: false,
				Reason:      "no extractable features from title/summary",
			},
			Rules: domain.RuleAssessment{
				Evaluated: false,
				Matched:   false,
				Reason:    "skipped because no extractable features were found",
			},
			NaiveBayes: domain.NaiveBayesAssessment{
				Evaluated: false,
				Reason:    "not applicable for rules classifier",
			},
			Decision: domain.ClassificationDecision{
				State:    domain.DecisionRejectedNoFeatures,
				Accepted: false,
				Reasons:  []string{"no extractable features from title/summary"},
			},
		}, nil
	}

	out := make([]domain.Signal, 0, len(rules))
	matches := make([]domain.RuleMatchResult, 0, len(rules))

	for _, rule := range rules {
		matchedTerms := matchTerms(text, rule.PositiveFeatures)
		score := scoreRuleMatch(text, rule)

		match := domain.RuleMatchResult{
			RuleID:       rule.Name,
			RuleName:     rule.Name,
			Matched:      false,
			Score:        score,
			MatchedTerms: matchedTerms,
		}

		switch {
		case len(matchedTerms) == 0:
			match.Reason = "no positive features matched"
		case score < rule.Threshold:
			match.Reason = "matched positive features but score below threshold"
		default:
			match.Matched = true
			match.Reason = "rule matched and passed threshold"
		}

		matches = append(matches, match)

		if !match.Matched {
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

	ruleAssessment := domain.RuleAssessment{
		Evaluated: true,
		Matched:   len(out) > 0,
		Matches:   matches,
	}

	decision := domain.ClassificationDecision{
		State:    domain.DecisionRejectedNoMatch,
		Accepted: false,
		Reasons:  []string{"no rules matched above threshold"},
	}

	if len(out) > 0 {
		top := out[0]
		for _, s := range out[1:] {
			if s.Probability > top.Probability {
				top = s
			}
		}

		decision = domain.ClassificationDecision{
			State:        domain.DecisionAcceptedSignal,
			Accepted:     true,
			PrimaryClass: top.Kind,
			Confidence:   top.Probability,
			Reasons:      []string{"one or more rules matched above threshold"},
		}
	} else if anyRuleBelowThreshold(matches) {
		decision = domain.ClassificationDecision{
			State:    domain.DecisionRejectedBelowThreshold,
			Accepted: false,
			Reasons:  []string{"one or more rules matched terms but scored below threshold"},
		}
	}

	return domain.ClassifierAssessmentResult{
		Classifier: c.Name(),
		Features: domain.FeatureAssessment{
			HasFeatures: true,
			Tokens:      tokens,
			Keywords:    tokens,
			Reason:      "extracted features from title/summary",
		},
		Rules: ruleAssessment,
		NaiveBayes: domain.NaiveBayesAssessment{
			Evaluated: false,
			Reason:    "not applicable for rules classifier",
		},
		Decision: decision,
		Signals:  out,
	}, nil
}

func matchTerms(text string, terms []string) []string {
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		tt := strings.ToLower(strings.TrimSpace(t))
		if tt == "" {
			continue
		}
		if matchesFeature(text, tt) {
			out = append(out, tt)
		}
	}
	return out
}

func scoreRuleMatch(text string, rule domain.SignalRule) float64 {
	pos := 0
	for _, f := range rule.PositiveFeatures {
		ff := strings.ToLower(strings.TrimSpace(f))
		if ff != "" && matchesFeature(text, ff) {
			pos++
		}
	}

	neg := 0
	for _, f := range rule.NegativeFeatures {
		ff := strings.ToLower(strings.TrimSpace(f))
		if ff != "" && matchesFeature(text, ff) {
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

func extractRuleAssessmentTokens(ev domain.Event) []string {
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

func matchesFeature(text, feature string) bool {
	if strings.Contains(text, feature) {
		return true
	}

	// avoid breaking multi-word phrases
	if strings.Contains(feature, " ") {
		return false
	}

	// "shortage" -> "shortages"
	if strings.HasSuffix(feature, "y") {
		if strings.Contains(text, strings.TrimSuffix(feature, "y")+"ies") {
			return true
		}
	}

	// "shipment" -> "shipments"
	if strings.Contains(text, feature+"s") {
		return true
	}

	return false
}
