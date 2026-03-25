package main

import (
	"context"
	"log"
	"net/http"
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
	memstore "probability-engine/internal/persistence/memory"
	s3store "probability-engine/internal/persistence/s3"
	"probability-engine/internal/ports"
	"probability-engine/internal/rules"
	lokisrc "probability-engine/internal/source/loki"
)

func main() {
	log.Println("probability-engine starting")

	source := buildEventSource()
	ruleLoader := rules.NewLoader(envOrDefault("RULES_DIR", "./rules"))
	ruleClassifier := rulecls.NewClassifier()

	bayesModel := buildBayesModel()
	bayesClassifier := bayes.NewClassifier(bayesModel, feat.NewDefaultExtractor())

	signalStore, insightStore, runStateStore := buildStores()

	eng := engine.New(engine.Options{
		Source:       source,
		RuleLoader:   ruleLoader,
		Classifiers:  []ports.SignalClassifier{ruleClassifier, bayesClassifier},
		Aggregator:   agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:     noop.New(),
		SignalStore:  signalStore,
		InsightStore: insightStore,
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
	client := lokisrc.NewHTTPClient(
		envOrDefault("LOKI_BASE_URL", "http://loki:3100"),
		&http.Client{Timeout: 15 * time.Second},
	)

	query := envOrDefault("LOKI_EVENTS_QUERY", `{ingester="newsdata"}`)
	limit := mustInt("LOKI_EVENTS_LIMIT", 1000)

	log.Printf("source: backend=loki base_url=%s query=%s limit=%d",
		envOrDefault("LOKI_BASE_URL", "http://loki:3100"),
		query,
		limit,
	)

	return lokisrc.New(client, query, limit, nil)
}

func buildStores() (ports.SignalStore, ports.InsightStore, ports.RunStateStore) {
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
		store := s3store.New(
			client,
			envOrDefault("PROBABILITY_S3_BUCKET", "probability-engine-test-383874363596-us-east-1-an"),
			envOrDefault("PROBABILITY_S3_PREFIX", "probability-engine"),
			envOrDefault("PROBABILITY_S3_ENV", "dev"),
			nil,
		)

		log.Printf("persistence: backend=s3 bucket=%s prefix=%s env=%s",
			envOrDefault("PROBABILITY_S3_BUCKET", "probability-engine-test-383874363596-us-east-1-an"),
			envOrDefault("PROBABILITY_S3_PREFIX", "probability-engine"),
			envOrDefault("PROBABILITY_S3_ENV", "dev"),
		)

		return store, store, store

	default:
		log.Printf("persistence: backend=memory")
		return memstore.NewSignalStore(), memstore.NewInsightStore(), memstore.NewRunStateStore()
	}
}

func buildBayesModel() bayes.Model {
	return bayes.Model{
		Version: "v0",
		Classes: map[string]bayes.ClassModel{
			"sanctions": {
				Prior: 0.4,
				Likelihoods: map[string]float64{
					"sanctions":                  0.8,
					"export":                     0.6,
					"restrictions":               0.7,
					"energy":                     0.5,
					"infrastructure":             0.4,
					"label:category=geopolitics": 0.7,
				},
				MarketScope: []string{"fx", "commodities"},
				Direction:   "negative",
			},
			"supply_chain": {
				Prior: 0.3,
				Likelihoods: map[string]float64{
					"minerals":             0.8,
					"supply":               0.6,
					"chain":                0.5,
					"disruption":           0.7,
					"label:category=trade": 0.6,
				},
				MarketScope: []string{"metals", "commodities"},
				Direction:   "negative",
			},
			"conflict_energy": {
				Prior: 0.3,
				Likelihoods: map[string]float64{
					"conflict":              0.7,
					"energy":                0.7,
					"infrastructure":        0.6,
					"pipeline":              0.7,
					"gas":                   0.6,
					"label:category=energy": 0.7,
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
