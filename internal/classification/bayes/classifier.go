package bayes

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"math"
	"sort"
	"strings"
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

	tokens := c.extractor.Extract(ev)
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

	if !passesClassGate(top.class, tokens) {
		return nil, nil
	}

	prob := calibratedProbability(scores, top.class)
	prob = applySourceWeight(prob, ev.Source, ev.Labels)

	if prob < classThreshold(top.class) {
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
	log.Printf("BAYES_DEBUG entered classify event=%s", ev.ID)

	log.Printf("BAYES_DEBUG classified event=%s kind=%s prob=%.2f source=%s title=%q tokens=%v",
		ev.ID, top.class, prob, ev.Source, ev.Title, tokens)
	return []domain.Signal{sig}, nil
}

func passesClassGate(className string, tokens []string) bool {
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		tokenSet[t] = struct{}{}
	}

	switch className {
	case "supply_chain":
		if hasAny(tokenSet, "minerals", "shortages", "bottleneck", "logistics") {
			return true
		}
		return hasAll(tokenSet, "industrial", "metals")

	case "sanctions":
		return hasAny(tokenSet, "sanctions", "embargo", "sanctioned") ||
			hasAll(tokenSet, "export", "ban")

	case "conflict_energy":
		if hasAny(tokenSet, "pipeline", "refinery") {
			return true
		}
		return hasAll(tokenSet, "energy", "infrastructure") ||
			hasAll(tokenSet, "conflict", "gas")

	default:
		return true
	}
}

func hasAll(set map[string]struct{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := set[k]; !ok {
			return false
		}
	}
	return true
}

func hasAny(set map[string]struct{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := set[k]; ok {
			return true
		}
	}
	return false
}

func classThreshold(className string) float64 {
	switch className {
	case "supply_chain":
		return 0.68
	case "sanctions":
		return 0.63
	case "conflict_energy":
		return 0.68
	default:
		return 0.65
	}
}

func applySourceWeight(prob float64, source string, labels map[string]string) float64 {
	adjusted := prob * sourceWeight(source, labels)
	return clampProbability(adjusted)
}

func sourceWeight(source string, labels map[string]string) float64 {
	source = strings.ToLower(strings.TrimSpace(source))

	weights := map[string]float64{
		"newsdata": 1.00,
		"gta":      1.10,
		"coindesk": 1.05,
	}

	w := weights[source]
	if w == 0 {
		w = 1.00
	}

	switch strings.ToLower(strings.TrimSpace(labels["source_id"])) {
	case "techbullion", "menafn", "newsbtc", "themarketsdaily":
		w *= 0.80
	}

	switch strings.ToLower(strings.TrimSpace(labels["source_name"])) {
	case "markets daily":
		w *= 0.92
	}

	return w
}

func calibratedProbability(scores []scored, winner string) float64 {
	base := normalizeTopProbability(scores, winner)
	base = minFloat(base, 0.95)

	if len(scores) < 2 {
		return minFloat(base, 0.93)
	}

	margin := scores[0].score - scores[1].score
	marginConf := marginConfidence(margin)

	p := 0.6*base + 0.4*marginConf
	return minFloat(p, 0.93)
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

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func clampProbability(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 0.95 {
		return 0.95
	}
	return v
}
