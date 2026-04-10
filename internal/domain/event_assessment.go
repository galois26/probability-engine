package domain

import (
	"time"
)

type DecisionState string

const (
	DecisionAcceptedSignal         DecisionState = "accepted_signal"
	DecisionRejectedBelowThreshold DecisionState = "rejected_below_threshold"
	DecisionRejectedNoMatch        DecisionState = "rejected_no_match"
	DecisionRejectedNoFeatures     DecisionState = "rejected_no_features"
)

type EventAssessment struct {
	ID         string                 `json:"id"`
	RunID      string                 `json:"runId"`
	AssessedAt time.Time              `json:"assessedAt"`
	Event      EventSnapshot          `json:"event"`
	Features   FeatureAssessment      `json:"features"`
	Rules      RuleAssessment         `json:"rules"`
	NaiveBayes NaiveBayesAssessment   `json:"naiveBayes"`
	Decision   ClassificationDecision `json:"decision"`
	Signals    []SignalSnapshot       `json:"signals,omitempty"`
}

type EventSnapshot struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Title     string                 `json:"title"`
	Summary   string                 `json:"summary"`
	URL       string                 `json:"url"`
	Published time.Time              `json:"published"`
	Lang      string                 `json:"lang"`
	Country   string                 `json:"country"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Raw       map[string]interface{} `json:"raw,omitempty"`
}

type FeatureAssessment struct {
	HasFeatures bool     `json:"hasFeatures"`
	Tokens      []string `json:"tokens,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

type RuleAssessment struct {
	Evaluated bool              `json:"evaluated"`
	Matched   bool              `json:"matched"`
	Matches   []RuleMatchResult `json:"matches,omitempty"`
	Reason    string            `json:"reason,omitempty"`
}

type RuleMatchResult struct {
	RuleID       string   `json:"ruleId"`
	RuleName     string   `json:"ruleName,omitempty"`
	Matched      bool     `json:"matched"`
	Score        float64  `json:"score,omitempty"`
	MatchedTerms []string `json:"matchedTerms,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

type NaiveBayesAssessment struct {
	Evaluated      bool         `json:"evaluated"`
	PredictedClass string       `json:"predictedClass,omitempty"`
	Scores         []ClassScore `json:"scores,omitempty"`
	Reason         string       `json:"reason,omitempty"`
}

type ClassScore struct {
	Class       string  `json:"class"`
	Score       float64 `json:"score"`
	Probability float64 `json:"probability,omitempty"`
}

type ClassificationDecision struct {
	State        DecisionState `json:"state"`
	Accepted     bool          `json:"accepted"`
	PrimaryClass string        `json:"primaryClass,omitempty"`
	Confidence   float64       `json:"confidence,omitempty"`
	Threshold    float64       `json:"threshold,omitempty"`
	Reasons      []string      `json:"reasons,omitempty"`
}

type SignalSnapshot struct {
	ID          string            `json:"id"`
	EventID     string            `json:"eventId"`
	Kind        string            `json:"kind"`
	MarketScope []string          `json:"marketScope,omitempty"`
	Direction   Direction         `json:"direction"`
	Probability float64           `json:"probability"`
	Classifier  string            `json:"classifier"`
	Features    []string          `json:"features,omitempty"`
	Explanation string            `json:"explanation,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	Trace       SignalTrace       `json:"trace"`
}

func NewEventSnapshot(e Event) EventSnapshot {
	return EventSnapshot{
		ID:        e.ID,
		Source:    e.Source,
		Title:     e.Title,
		Summary:   e.Summary,
		URL:       e.URL,
		Published: e.Published,
		Lang:      e.Lang,
		Country:   e.Country,
		Labels:    e.Labels,
		Raw:       e.Raw,
	}
}

func NewSignalSnapshot(s Signal) SignalSnapshot {
	return SignalSnapshot{
		ID:          s.ID,
		EventID:     s.EventID,
		Kind:        s.Kind,
		MarketScope: s.MarketScope,
		Direction:   s.Direction,
		Probability: s.Probability,
		Classifier:  s.Classifier,
		Features:    s.Features,
		Explanation: s.Explanation,
		Labels:      s.Labels,
		CreatedAt:   s.CreatedAt,
		Trace:       s.Trace,
	}
}
