package domain

type SignalRule struct {
	Name             string            `json:"name" yaml:"name"`
	MarketScope      []string          `json:"marketScope" yaml:"marketScope"`
	DirectionDefault Direction         `json:"directionDefault" yaml:"directionDefault"`
	Threshold        float64           `json:"threshold" yaml:"threshold"`
	PositiveFeatures []string          `json:"positiveFeatures,omitempty" yaml:"positiveFeatures"`
	NegativeFeatures []string          `json:"negativeFeatures,omitempty" yaml:"negativeFeatures"`
	WeightedFeatures []WeightedFeature `json:"weightedFeatures,omitempty" yaml:"weightedFeatures"`
	Examples         []string          `json:"examples,omitempty" yaml:"examples"`
	Labels           map[string]string `json:"labels,omitempty" yaml:"labels"`
}

type WeightedFeature struct {
	Token  string  `yaml:"token"`
	Weight float64 `yaml:"weight"`
}
