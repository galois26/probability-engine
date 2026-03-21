package engine

import (
	"crypto/sha1"
	"encoding/hex"
	"probability-engine/internal/domain"
	"sort"
	"strings"
)

func resolveSignals(in []domain.Signal) []domain.Signal {
	grouped := map[string][]domain.Signal{}

	for _, s := range in {
		key := signalKey(s)
		grouped[key] = append(grouped[key], s)
	}

	out := make([]domain.Signal, 0, len(grouped))

	for _, group := range grouped {
		out = append(out, mergeSignalGroup(group))
	}

	return out
}

func signalKey(s domain.Signal) string {
	return s.EventID + "|" + s.Kind + "|" + string(s.Direction)
}

func mergeSignalGroup(group []domain.Signal) domain.Signal {
	if len(group) == 1 {
		return canonicalize(group[0])
	}

	sort.Slice(group, func(i, j int) bool {
		return group[i].Probability > group[j].Probability
	})

	base := group[0]

	scopeSet := map[string]struct{}{}
	featureSet := map[string]struct{}{}
	labelMap := map[string]string{}
	classifiers := map[string]struct{}{}

	var maxProb float64
	var explanations []string

	var trace domain.SignalTrace
	trace.EventID = base.EventID

	for _, s := range group {
		if s.Probability > maxProb {
			maxProb = s.Probability
		}

		// scope
		for _, sc := range s.MarketScope {
			sc = strings.TrimSpace(sc)
			if sc != "" {
				scopeSet[sc] = struct{}{}
			}
		}

		// features
		for _, f := range s.Features {
			f = strings.TrimSpace(f)
			if f != "" {
				featureSet[f] = struct{}{}
			}
		}

		// labels
		for k, v := range s.Labels {
			if _, ok := labelMap[k]; !ok {
				labelMap[k] = v
			}
		}

		classifiers[s.Classifier] = struct{}{}

		if s.Explanation != "" {
			explanations = append(explanations, s.Explanation)
		}

		// trace merge
		trace.MatchedRules = append(trace.MatchedRules, s.Trace.MatchedRules...)
		trace.MatchedTerms = append(trace.MatchedTerms, s.Trace.MatchedTerms...)
		if trace.ModelVersion == "" {
			trace.ModelVersion = s.Trace.ModelVersion
		}
	}

	return domain.Signal{
		ID:          stableSignalID(base.EventID, base.Kind, base.Direction),
		EventID:     base.EventID,
		Kind:        base.Kind,
		Direction:   base.Direction,
		Probability: maxProb,
		MarketScope: setToSortedSlice(scopeSet),
		Features:    setToSortedSlice(featureSet),
		Labels:      labelMap,
		Classifier:  joinKeys(classifiers),
		Explanation: strings.Join(explanations, "; "),
		CreatedAt:   base.CreatedAt,
		Trace:       dedupeTrace(trace),
	}
}

func canonicalize(s domain.Signal) domain.Signal {
	s.MarketScope = normalizeSlice(s.MarketScope)
	s.Features = normalizeSlice(s.Features)
	return s
}

func normalizeSlice(in []string) []string {
	set := map[string]struct{}{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" {
			set[v] = struct{}{}
		}
	}
	return setToSortedSlice(set)
}

func setToSortedSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinKeys(m map[string]struct{}) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, "+")
}

func stableSignalID(eventID, kind string, dir domain.Direction) string {
	h := sha1.New()
	h.Write([]byte(eventID))
	h.Write([]byte("|"))
	h.Write([]byte(kind))
	h.Write([]byte("|"))
	h.Write([]byte(string(dir)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func dedupeTrace(t domain.SignalTrace) domain.SignalTrace {
	t.MatchedRules = normalizeSlice(t.MatchedRules)
	t.MatchedTerms = normalizeSlice(t.MatchedTerms)
	return t
}
