package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Config struct {
	Loki        LokiConfig
	LokiPush    LokiPushConfig
	Persistence PersistenceConfig
	Engine      EngineConfig
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
	EventsQuery     string        `yaml:"eventsQuery"`
	EventsLimit     int           `yaml:"eventsLimit"`
	QueryDirection  string        `yaml:"queryDirection"`
	Lookback        time.Duration `yaml:"lookback"`
}

// LokiPushConfig holds configuration for publishing assessments back to Loki.
type LokiPushConfig struct {
	Enabled         bool          `yaml:"enabled"`
	BaseURL         string        `yaml:"baseURL"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	TenantID        string        `yaml:"tenantID"`
	Timeout         time.Duration `yaml:"timeout"`
	InsecureSkipTLS bool
}

// PersistenceConfig holds configuration for the persistence backend.
type PersistenceConfig struct {
	// Backend is one of "memory" or "s3".
	Backend   string
	S3Bucket  string
	S3Prefix  string
	S3Env     string
	AWSRegion string
}

// EngineConfig holds configuration for the engine worker.
type EngineConfig struct {
	JobName         string
	AppEnv          string
	RulesDir        string
	PollInterval    time.Duration
	InitialLookback time.Duration
	QueryOverlap    time.Duration
	IgnoreRunState  bool
}

// Load reads all configuration from environment variables, applying defaults
// where appropriate. It returns an error if any required value is missing or
// any value fails to parse.
func Load() (*Config, error) {
	var errs []error

	collect := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}

	// --- Loki source ---
	lokiTimeout, err := envDuration("LOKI_TIMEOUT", 15*time.Second)
	collect(err)

	ignoreRunState, err := envBool("ENGINE_IGNORE_RUN_STATE", false)
	collect(err)
	lokiInsecure, err := envBool("LOKI_INSECURE_SKIP_TLS", false)
	collect(err)

	lokiLimit, err := envInt("LOKI_EVENTS_LIMIT", 1000)
	collect(err)

	lokiBaseURL := envOrDefault("LOKI_BASE_URL", "http://loki:3100")

	lokiLookback, err := envDuration("LOKI_QUERY_LOOKBACK", 48*time.Hour)
	collect(err)

	queryOverlap, err := envDuration("ENGINE_QUERY_OVERLAP", 0)
	collect(err)
	// --- Loki push ---
	lokiPushEnabled, err := envBool("LOKI_PUBLISH_ASSESSMENTS", false)
	collect(err)

	lokiPushTimeout, err := envDuration("LOKI_PUSH_TIMEOUT", 15*time.Second)
	collect(err)

	lokiPushInsecure, err := envBool("LOKI_PUSH_INSECURE_SKIP_TLS", false)
	collect(err)

	// Loki push falls back to main Loki credentials if not explicitly set.
	lokiPushBaseURL := envOrDefault("LOKI_PUSH_BASE_URL", lokiBaseURL)
	lokiPushUsername := envOrDefault("LOKI_PUSH_USERNAME", envOrDefault("LOKI_USERNAME", ""))
	lokiPushPassword := envOrDefault("LOKI_PUSH_PASSWORD", envOrDefault("LOKI_PASSWORD", ""))
	lokiPushTenantID := envOrDefault("LOKI_PUSH_TENANT_ID", envOrDefault("LOKI_TENANT_ID", ""))

	// --- Engine ---
	pollInterval, err := envDuration("POLL_INTERVAL", 60*time.Second)
	collect(err)

	initialLookback, err := envDuration("INITIAL_LOOKBACK", 24*time.Hour)
	collect(err)

	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("config errors:\n  %s", strings.Join(msgs, "\n  "))
	}

	cfg := &Config{
		Loki: LokiConfig{
			BaseURL:         lokiBaseURL,
			Username:        envOrDefault("LOKI_USERNAME", ""),
			Password:        envOrDefault("LOKI_PASSWORD", ""),
			TenantID:        envOrDefault("LOKI_TENANT_ID", ""),
			Timeout:         lokiTimeout,
			InsecureSkipTLS: lokiInsecure,
			EventsQuery:     envOrDefault("LOKI_EVENTS_QUERY", `{ingester="newsdata"}`),
			EventsLimit:     lokiLimit,
			QueryDirection:  envOrDefault("LOKI_QUERY_DIRECTION", "forward"),
			Lookback:        lokiLookback,
		},
		LokiPush: LokiPushConfig{
			Enabled:         lokiPushEnabled,
			BaseURL:         lokiPushBaseURL,
			Username:        lokiPushUsername,
			Password:        lokiPushPassword,
			TenantID:        lokiPushTenantID,
			Timeout:         lokiPushTimeout,
			InsecureSkipTLS: lokiPushInsecure,
		},
		Persistence: PersistenceConfig{
			Backend:   strings.ToLower(envOrDefault("PERSISTENCE_BACKEND", "memory")),
			S3Bucket:  envOrDefault("PROBABILITY_S3_BUCKET", "probability-engine-test-383874363596-us-east-1-an"),
			S3Prefix:  envOrDefault("PROBABILITY_S3_PREFIX", "probability-engine"),
			S3Env:     envOrDefault("PROBABILITY_S3_ENV", "test"),
			AWSRegion: envOrDefault("AWS_REGION", "eu-west-1"),
		},
		Engine: EngineConfig{
			JobName:         envOrDefault("ENGINE_JOB_NAME", "probability-engine"),
			AppEnv:          envOrDefault("APP_ENV", "dev"),
			RulesDir:        envOrDefault("RULES_DIR", "./rules"),
			PollInterval:    pollInterval,
			InitialLookback: initialLookback,
			IgnoreRunState:  ignoreRunState,
			QueryOverlap:    queryOverlap,
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate performs cross-field validation that cannot be expressed as simple
// parse errors (e.g. required combinations, mutually exclusive values).
func (c *Config) validate() error {
	if c.Persistence.Backend != "memory" && c.Persistence.Backend != "s3" {
		return fmt.Errorf("config: PERSISTENCE_BACKEND must be 'memory' or 's3', got %q", c.Persistence.Backend)
	}
	if c.Loki.QueryDirection != "forward" && c.Loki.QueryDirection != "backward" {
		return errors.New("config: LOKI_QUERY_DIRECTION must be 'forward' or 'backward'")
	}
	return nil
}
