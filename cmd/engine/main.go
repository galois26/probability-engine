package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	agg "probability-engine/internal/aggregation/rolling"
	bayes "probability-engine/internal/classification/bayes"
	feat "probability-engine/internal/classification/features"
	rulecls "probability-engine/internal/classification/rules"
	"probability-engine/internal/engine"
	noop "probability-engine/internal/enrichment/noop"
	lokipersist "probability-engine/internal/persistence/loki"
	memstore "probability-engine/internal/persistence/memory"
	s3store "probability-engine/internal/persistence/s3"
	"probability-engine/internal/ports"
	rules "probability-engine/internal/rules"
	lokisrc "probability-engine/internal/source/loki"
)

func main() {
	log.Println("probability-engine starting")

	source := buildEventSource()
	ruleLoader := rules.NewLoader(envOrDefault("RULES_DIR", "./rules"))
	ruleClassifier := rulecls.NewClassifier()

	bayesModel := buildBayesModel()
	bayesClassifier := bayes.NewClassifier(bayesModel, feat.NewDefaultExtractor())

	signalStore, eventAssessmentStore, insightStore, runStateStore := buildStores()
	eventAssessmentPublisher := buildEventAssessmentPublisher()

	eng := engine.New(engine.Options{
		Source:                   source,
		RuleLoader:               ruleLoader,
		Classifiers:              []ports.SignalClassifier{ruleClassifier, bayesClassifier},
		Aggregator:               agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:                 noop.New(),
		SignalStore:              signalStore,
		EventAssessmentStore:     eventAssessmentStore,
		EventAssessmentPublisher: eventAssessmentPublisher,
		InsightStore:             insightStore,
	})

	worker := engine.NewWorker(engine.WorkerOptions{
		Engine:       eng,
		StateStore:   runStateStore,
		JobName:      envOrDefault("ENGINE_JOB_NAME", "probability-engine"),
		PollInterval: mustDuration("POLL_INTERVAL", 60*time.Second),
		Lookback:     mustDuration("INITIAL_LOOKBACK", 15*time.Minute),
	})

	ctx := context.Background()
	if err := worker.Run(ctx); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}

func buildEventSource() ports.EventSource {
	baseURL := envOrDefault("LOKI_BASE_URL", "http://loki:3100")
	username := envOrDefault("LOKI_USERNAME", "")
	password := envOrDefault("LOKI_PASSWORD", "")
	tenantID := envOrDefault("LOKI_TENANT_ID", "")
	timeout := mustDuration("LOKI_TIMEOUT", 15*time.Second)
	insecureSkipTLS := mustBool("LOKI_INSECURE_SKIP_TLS", false)

	query := envOrDefault("LOKI_EVENTS_QUERY", `{ingester="newsdata"}`)
	limit := mustInt("LOKI_EVENTS_LIMIT", 1000)
	direction := envOrDefault("LOKI_QUERY_DIRECTION", "forward")

	client := lokisrc.NewHTTPClient(
		baseURL,
		username,
		password,
		tenantID,
		timeout,
		insecureSkipTLS,
	)

	log.Printf(
		"source: backend=loki base_url=%s query=%s limit=%d direction=%s tenant=%t auth=%t timeout=%s insecure_skip_tls=%t",
		baseURL,
		query,
		limit,
		direction,
		tenantID != "",
		username != "" || password != "",
		timeout,
		insecureSkipTLS,
	)

	return lokisrc.New(
		client,
		query,
		limit,
		direction,
		nil,
	)
}

func buildStores() (
	ports.SignalStore,
	ports.EventAssessmentStore,
	ports.InsightStore,
	ports.RunStateStore,
) {
	backend := strings.ToLower(envOrDefault("PERSISTENCE_BACKEND", "memory"))

	switch backend {
	case "s3":
		ctx := context.Background()

		awsCfg, err := awsconfig.LoadDefaultConfig(
			ctx,
			awsconfig.WithRegion(envOrDefault("AWS_REGION", "eu-west-1")),
		)
		if err != nil {
			log.Fatalf("load aws config: %v", err)
		}

		client := awss3.NewFromConfig(awsCfg)
		bucket := envOrDefault("PROBABILITY_S3_BUCKET", "probability-engine-test-383874363596-us-east-1-an")
		prefix := envOrDefault("PROBABILITY_S3_PREFIX", "probability-engine")
		envName := envOrDefault("PROBABILITY_S3_ENV", "test")
		region := envOrDefault("AWS_REGION", "us-east-1")

		store := s3store.New(
			client,
			bucket,
			prefix,
			envName,
			nil,
		)

		log.Printf(
			"persistence: backend=s3 bucket=%q prefix=%q env=%q region=%q assessments=true",
			bucket,
			prefix,
			envName,
			region,
		)

		return store, store, store, store

	default:
		log.Printf("persistence: backend=memory assessments=true")
		return memstore.NewSignalStore(),
			memstore.NewEventAssessmentStore(),
			memstore.NewInsightStore(),
			memstore.NewRunStateStore()

	}
}

func buildBayesModel() bayes.Model {
	return bayes.Model{
		Version: "v0",
		Classes: map[string]bayes.ClassModel{
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

func envOrDefault(name, def string) string {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	return v
}

func mustEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("missing required env: %s", name)
	}
	return v
}

func mustInt(name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	out, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("invalid int for %s: %q", name, v)
	}
	return out
}

func mustDuration(name string, def time.Duration) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Fatalf("invalid duration for %s: %q", name, v)
	}
	return d
}

func mustBool(name string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return def
	}

	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		log.Fatalf("invalid bool for %s: %q", name, v)
		return def
	}
}

func buildEventAssessmentPublisher() ports.EventAssessmentPublisher {
	if !mustBool("LOKI_PUBLISH_ASSESSMENTS", false) {
		log.Printf("loki publisher: disabled")
		return nil
	}

	baseURL := envOrDefault("LOKI_PUSH_BASE_URL", envOrDefault("LOKI_BASE_URL", "http://loki:3100"))
	username := envOrDefault("LOKI_PUSH_USERNAME", envOrDefault("LOKI_USERNAME", ""))
	password := envOrDefault("LOKI_PUSH_PASSWORD", envOrDefault("LOKI_PASSWORD", ""))
	tenantID := envOrDefault("LOKI_PUSH_TENANT_ID", envOrDefault("LOKI_TENANT_ID", ""))
	timeout := mustDuration("LOKI_PUSH_TIMEOUT", 15*time.Second)
	insecureSkipTLS := mustBool("LOKI_PUSH_INSECURE_SKIP_TLS", false)

	pushClient := lokipersist.NewHTTPPushClient(
		baseURL,
		username,
		password,
		tenantID,
		timeout,
		insecureSkipTLS,
	)

	publisher := lokipersist.NewEventAssessmentPublisher(
		pushClient,
		envOrDefault("ENGINE_JOB_NAME", "probability-engine"),
		envOrDefault("APP_ENV", "dev"),
		nil,
	)

	log.Printf(
		"loki publisher: enabled base_url=%s tenant=%t auth=%t timeout=%s insecure_skip_tls=%t",
		baseURL,
		tenantID != "",
		username != "" || password != "",
		timeout,
		insecureSkipTLS,
	)

	return publisher
}
