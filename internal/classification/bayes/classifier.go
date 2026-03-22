package bayes

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"math"
	"sort"
	"time"

	"probability-engine/internal/classification/features"
	"probability-engine/internal/domain"
)

type scored struct {
	class string
	score float64
}

type Classifier struct {
	model     Model
	extractor features.Extractor
}

func NewClassifier(model Model, extractor features.Extractor) *Classifier {
	return &Classifier{
		model:     model,
		extractor: extractor,
	}
}

func (c *Classifier) Name() string { return "naive_bayes" }

func (c *Classifier) Classify(ctx context.Context, ev domain.Event, rules []domain.SignalRule) ([]domain.Signal, error) {
	_ = ctx
	_ = rules

	if len(c.model.Classes) == 0 {
		return nil, nil
	}

	//tokens := c.extractor.Extract(ev)
	tokens := uniqueStrings(c.extractor.Extract(ev))
	if len(tokens) == 0 {
		return nil, nil
	}

	scores := make([]scored, 0, len(c.model.Classes))

	for className, classModel := range c.model.Classes {
		logProb := math.Log(maxFloat(classModel.Prior, 1e-9))
		for _, tok := range tokens {
			p := classModel.Likelihoods[tok]
			if p == 0 {
				p = 0.05
			}
			logProb += math.Log(p)
		}
		scores = append(scores, scored{class: className, score: logProb})
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	top := scores[0]

	prob := calibratedProbability(scores, top.class)
	if prob < 0.55 {
		return nil, nil
	}
	classModel := c.model.Classes[top.class]
	now := time.Now().UTC()

	sig := domain.Signal{
		ID:          buildSignalID(ev.ID, top.class, "naive_bayes"),
		EventID:     ev.ID,
		Kind:        top.class,
		MarketScope: append([]string(nil), classModel.MarketScope...),
		Direction:   domain.Direction(classModel.Direction),
		Probability: prob,
		Classifier:  c.Name(),
		Features:    tokens,
		Explanation: "classified by naive Bayes model",
		CreatedAt:   now,
		Trace: domain.SignalTrace{
			EventID:      ev.ID,
			ModelVersion: c.model.Version,
		},
	}

	if sig.Direction == "" {
		sig.Direction = domain.DirectionUnknown
	}

	return []domain.Signal{sig}, nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func calibratedProbability(scores []scored, winner string) float64 {
	base := normalizeTopProbability(scores, winner)
	base = minFloat(base, 0.95)

	if len(scores) < 2 {
		return base
	}

	margin := scores[0].score - scores[1].score
	marginConf := marginConfidence(margin)

	// Blend posterior-like score with margin-based confidence.
	// Keeps ranking behavior while reducing overconfidence.
	p := 0.6*base + 0.4*marginConf

	// Conservative ceiling for a small hand-built model.
	return minFloat(p, 0.93)
}

func marginConfidence(margin float64) float64 {
	switch {
	case margin >= 3.0:
		return 0.90
	case margin >= 2.0:
		return 0.82
	case margin >= 1.0:
		return 0.72
	case margin >= 0.5:
		return 0.62
	default:
		return 0.55
	}
}
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func normalizeTopProbability(scores []scored, winner string) float64 {
	const temperature = 2.5

	maxScore := scores[0].score
	var sum float64
	var winnerExp float64

	for _, s := range scores {
		v := math.Exp((s.score - maxScore) / temperature)
		sum += v
		if s.class == winner {
			winnerExp = v
		}
	}
	if sum == 0 {
		return 0
	}
	return winnerExp / sum
}

func buildSignalID(eventID, className, classifier string) string {
	h := sha1.New()
	h.Write([]byte(eventID))
	h.Write([]byte("|"))
	h.Write([]byte(className))
	h.Write([]byte("|"))
	h.Write([]byte(classifier))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
