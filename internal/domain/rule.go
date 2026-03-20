package domain

type SignalRule struct {
	Name             string            `json:"name"`
	MarketScope      []string          `json:"marketScope"`
	DirectionDefault Direction         `json:"directionDefault"`
	Threshold        float64           `json:"threshold"`
	PositiveFeatures []string          `json:"positiveFeatures,omitempty"`
	NegativeFeatures []string          `json:"negativeFeatures,omitempty"`
	Examples         []string          `json:"examples,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
}
