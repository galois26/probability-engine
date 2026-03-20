package domain

import "time"

type Insight struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	MarketScope []string          `json:"marketScope"`
	Probability float64           `json:"probability"`
	Direction   Direction         `json:"direction"`
	Severity    Severity          `json:"severity"`
	Status      InsightStatus     `json:"status"`
	Summary     string            `json:"summary"`
	Signals     []string          `json:"signals"`
	Evidence    []InsightEvidence `json:"evidence"`
	Flags       InsightFlags      `json:"flags"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type InsightEvidence struct {
	SignalID    string    `json:"signalId"`
	EventID     string    `json:"eventId"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Published   time.Time `json:"published"`
	WhyIncluded string    `json:"whyIncluded"`
}

type InsightFlags struct {
	Acknowledged  bool   `json:"acknowledged"`
	Silenced      bool   `json:"silenced"`
	DuplicateOf   string `json:"duplicateOf,omitempty"`
	FalsePositive bool   `json:"falsePositive"`
	Notes         string `json:"notes,omitempty"`
}
