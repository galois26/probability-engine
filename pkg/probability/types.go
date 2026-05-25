package probability

import "time"

type Event struct {
	ID          string            `json:"id"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	Source      string            `json:"source"`
	Title       string            `json:"title"`
	Summary     string            `json:"summary"`
	URL         string            `json:"url"`
	PublishedAt time.Time         `json:"published_at"`
	Lang        string            `json:"lang"`
	Country     string            `json:"country"`
	Labels      map[string]string `json:"labels,omitempty"`
	Raw         map[string]any    `json:"raw,omitempty"`
}

type Assessment struct {
	EventID       string                       `json:"eventId"`
	ID            string                       `json:"id"`
	RunID         string                       `json:"runId,omitempty"`
	AssessedAt    time.Time                    `json:"assessedAt"`
	Event         EventSnapshot                `json:"event"`
	Decision      ClassificationDecision       `json:"decision"`
	Signals       []SignalSnapshot             `json:"signals,omitempty"`
	Classifiers   []ClassifierAssessmentResult `json:"classifiers,omitempty"`
	EngineVersion string                       `json:"engineVersion,omitempty"`
	RuleVersion   string                       `json:"ruleVersion,omitempty"`
}

type EventSnapshot struct {
	ID          string            `json:"id"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	Source      string            `json:"source"`
	Title       string            `json:"title"`
	Summary     string            `json:"summary"`
	URL         string            `json:"url"`
	Published   time.Time         `json:"published"`
	Lang        string            `json:"lang"`
	Country     string            `json:"country"`
	Labels      map[string]string `json:"labels,omitempty"`
	Raw         map[string]any    `json:"raw,omitempty"`
}

type ClassifierAssessmentResult struct {
	Classifier string                 `json:"classifier"`
	Features   FeatureAssessment      `json:"features"`
	Rules      RuleAssessment         `json:"rules"`
	NaiveBayes NaiveBayesAssessment   `json:"naiveBayes"`
	Decision   ClassificationDecision `json:"decision"`
	Signals    []Signal               `json:"signals,omitempty"`
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
	State        string   `json:"state"`
	Accepted     bool     `json:"accepted"`
	PrimaryClass string   `json:"primaryClass,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	Threshold    float64  `json:"threshold,omitempty"`
	Reasons      []string `json:"reasons,omitempty"`
}

type SignalSnapshot struct {
	ID          string            `json:"id"`
	EventID     string            `json:"eventId"`
	Kind        string            `json:"kind"`
	MarketScope []string          `json:"marketScope,omitempty"`
	Direction   string            `json:"direction"`
	Probability float64           `json:"probability"`
	Classifier  string            `json:"classifier"`
	Features    []string          `json:"features,omitempty"`
	Explanation string            `json:"explanation,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	Trace       any               `json:"trace,omitempty"`
}

type Signal struct {
	ID          string            `json:"id"`
	EventID     string            `json:"eventId"`
	Kind        string            `json:"kind"`
	MarketScope []string          `json:"marketScope,omitempty"`
	Direction   string            `json:"direction"`
	Probability float64           `json:"probability"`
	Classifier  string            `json:"classifier"`
	Features    []string          `json:"features,omitempty"`
	Explanation string            `json:"explanation,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	Trace       any               `json:"trace,omitempty"`
}
