package bayes

// DefaultModel returns the default hand-tuned Bayes model used in production.
// Priors and likelihoods should be reviewed periodically against observed
// classification accuracy. For dynamic loading, see LoadModelFromFile.
func DefaultModel() Model {
	return Model{
		Version: "v0",
		Classes: map[string]ClassModel{
			"sanctions": {
				Prior: 0.4,
				Likelihoods: map[string]float64{
					"sanctions":                  0.85,
					"embargo":                    0.85,
					"sanctioned":                 0.75,
					"ban":                        0.70,
					"energy":                     0.35,
					"infrastructure":             0.20,
					"label:category=geopolitics": 0.70,
				},
				MarketScope: []string{"fx", "commodities"},
				Direction:   "negative",
			},
			"supply_chain": {
				Prior: 0.20,
				Likelihoods: map[string]float64{
					"minerals":             0.85,
					"shortages":            0.85,
					"bottleneck":           0.80,
					"disruption":           0.75,
					"logistics":            0.75,
					"industrial":           0.65,
					"metals":               0.75,
					"label:category=trade": 0.70,
					"supply":               0.35,
					"chain":                0.30,
				},
				MarketScope: []string{"metals", "commodities"},
				Direction:   "negative",
			},
			"conflict_energy": {
				Prior: 0.3,
				Likelihoods: map[string]float64{
					"conflict":              0.85,
					"infrastructure":        0.70,
					"pipeline":              0.70,
					"gas":                   0.60,
					"label:category=energy": 0.70,
				},
				MarketScope: []string{"oil", "gas", "energy"},
				Direction:   "negative",
			},
		},
	}
}
