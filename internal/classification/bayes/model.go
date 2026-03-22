package bayes

type Model struct {
	Version string                `json:"version"`
	Classes map[string]ClassModel `json:"classes"`
}

type ClassModel struct {
	Prior       float64            `json:"prior"`
	Likelihoods map[string]float64 `json:"likelihoods"`
	MarketScope []string           `json:"marketScope,omitempty"`
	Direction   string             `json:"direction,omitempty"`
}
