package domain

import "time"

type Signal struct {
	ID          string            `json:"id"`
	EventID     string            `json:"eventId"`
	Kind        string            `json:"kind"`
	MarketScope []string          `json:"marketScope"`
	Direction   Direction         `json:"direction"`
	Probability float64           `json:"probability"`
	Classifier  string            `json:"classifier"`
	Features    []string          `json:"features,omitempty"`
	Explanation string            `json:"explanation,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	Trace       SignalTrace       `json:"trace"`
}

type SignalTrace struct {
	EventID       string   `json:"eventId"`
	MatchedRules  []string `json:"matchedRules,omitempty"`
	MatchedTerms  []string `json:"matchedTerms,omitempty"`
	ModelVersion  string   `json:"modelVersion,omitempty"`
	ClassifierRun string   `json:"classifierRun,omitempty"`
}
