package probability

import "time"

type Config struct {
	EngineVersion string
	RuleVersion   string
	RulesDir      string
	Timeout       time.Duration
}
