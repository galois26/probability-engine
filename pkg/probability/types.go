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
	ID            string             `json:"id"`
	EventID       string             `json:"event_id"`
	Fingerprint   string             `json:"fingerprint,omitempty"`
	AssessedAt    time.Time          `json:"assessed_at"`
	Decision      Decision           `json:"decision"`
	Signals       []Signal           `json:"signals,omitempty"`
	Classifiers   []ClassifierResult `json:"classifiers,omitempty"`
	EngineVersion string             `json:"engine_version,omitempty"`
	RuleVersion   string             `json:"rule_version,omitempty"`
}

type Decision struct {
	State        string   `json:"state"`
	Accepted     bool     `json:"accepted"`
	PrimaryClass string   `json:"primary_class,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	Threshold    float64  `json:"threshold,omitempty"`
	Reasons      []string `json:"reasons,omitempty"`
}

type Signal struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Probability float64           `json:"probability"`
	Classifier  string            `json:"classifier"`
	Features    []string          `json:"features,omitempty"`
	Explanation string            `json:"explanation,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type ClassifierResult struct {
	Classifier string   `json:"classifier"`
	Accepted   bool     `json:"accepted"`
	State      string   `json:"state"`
	Confidence float64  `json:"confidence,omitempty"`
	Reasons    []string `json:"reasons,omitempty"`
}
