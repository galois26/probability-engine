package config

import "time"

type Config struct {
	Run  RunConfig  `yaml:"run"`
	Loki LokiConfig `yaml:"loki"`
}

type RunConfig struct {
	Lookback time.Duration `yaml:"lookback"`
}

type LokiConfig struct {
	Enabled         bool          `yaml:"enabled"`
	BaseURL         string        `yaml:"baseURL"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	TenantID        string        `yaml:"tenantID"`
	Query           string        `yaml:"query"`
	Limit           int           `yaml:"limit"`
	Direction       string        `yaml:"direction"`
	Timeout         time.Duration `yaml:"timeout"`
	QueryLookback   time.Duration `yaml:"queryLookback"`
	InsecureSkipTLS bool          `yaml:"insecureSkipTLS"`
}
