package config_test

import (
	"testing"
	"time"

	"github.com/galois26/probability-engine/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env that might leak from the host.
	t.Setenv("PERSISTENCE_BACKEND", "")
	t.Setenv("LOKI_QUERY_DIRECTION", "")
	t.Setenv("POLL_INTERVAL", "")
	t.Setenv("INITIAL_LOOKBACK", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Persistence.Backend != "memory" {
		t.Errorf("backend: want memory, got %q", cfg.Persistence.Backend)
	}
	if cfg.Engine.PollInterval != 60*time.Second {
		t.Errorf("poll interval: want 60s, got %v", cfg.Engine.PollInterval)
	}
	if cfg.Engine.InitialLookback != 15*time.Minute {
		t.Errorf("lookback: want 15m, got %v", cfg.Engine.InitialLookback)
	}
}

func TestLoad_InvalidPersistenceBackend(t *testing.T) {
	t.Setenv("PERSISTENCE_BACKEND", "redis")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid backend, got nil")
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Setenv("POLL_INTERVAL", "not-a-duration")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid duration, got nil")
	}
}

func TestLoad_InvalidBool(t *testing.T) {
	t.Setenv("LOKI_INSECURE_SKIP_TLS", "maybe")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid bool, got nil")
	}
}

func TestLoad_S3Backend(t *testing.T) {
	t.Setenv("PERSISTENCE_BACKEND", "s3")
	t.Setenv("PROBABILITY_S3_BUCKET", "my-bucket")
	t.Setenv("AWS_REGION", "us-west-2")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Persistence.Backend != "s3" {
		t.Errorf("backend: want s3, got %q", cfg.Persistence.Backend)
	}
	if cfg.Persistence.S3Bucket != "my-bucket" {
		t.Errorf("bucket: want my-bucket, got %q", cfg.Persistence.S3Bucket)
	}
	if cfg.Persistence.AWSRegion != "us-west-2" {
		t.Errorf("region: want us-west-2, got %q", cfg.Persistence.AWSRegion)
	}
}

func TestLoad_LokiPushFallsBackToLokiCredentials(t *testing.T) {
	t.Setenv("LOKI_USERNAME", "shared-user")
	t.Setenv("LOKI_PASSWORD", "shared-pass")
	t.Setenv("LOKI_PUSH_USERNAME", "")
	t.Setenv("LOKI_PUSH_PASSWORD", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.LokiPush.Username != "shared-user" {
		t.Errorf("push username: want shared-user, got %q", cfg.LokiPush.Username)
	}
	if cfg.LokiPush.Password != "shared-pass" {
		t.Errorf("push password: want shared-pass, got %q", cfg.LokiPush.Password)
	}
}
