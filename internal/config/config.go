package config

import "time"

type Config struct {
	Run RunConfig `yaml:"run"`
}

type RunConfig struct {
	Lookback time.Duration `yaml:"lookback"`
}
