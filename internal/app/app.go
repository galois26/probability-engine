// Package app is the composition root for the probability engine. It wires
// together all dependencies and returns a ready-to-run worker. Nothing in this
// package contains business logic — it only constructs and connects components.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	agg "github.com/galois26/probability-engine/internal/aggregation/rolling"
	bayes "github.com/galois26/probability-engine/internal/classification/bayes"
	feat "github.com/galois26/probability-engine/internal/classification/features"
	"github.com/galois26/probability-engine/internal/classification/keywordscoring"
	rulecls "github.com/galois26/probability-engine/internal/classification/rules"
	"github.com/galois26/probability-engine/internal/config"
	"github.com/galois26/probability-engine/internal/engine"
	noop "github.com/galois26/probability-engine/internal/enrichment/noop"
	lokipersist "github.com/galois26/probability-engine/internal/persistence/loki"
	memstore "github.com/galois26/probability-engine/internal/persistence/memory"
	s3store "github.com/galois26/probability-engine/internal/persistence/s3"
	"github.com/galois26/probability-engine/internal/ports"
	rules "github.com/galois26/probability-engine/internal/rules"
	lokisrc "github.com/galois26/probability-engine/internal/source/loki"
)

// Build constructs and wires all dependencies, returning a worker that is
// ready to run but has not yet started. This is the single place in the
// codebase that knows about every component.
func Build(cfg *config.Config, logger *slog.Logger) (*engine.Worker, error) {
	source := buildEventSource(cfg.Loki, logger)

	stores, err := buildStores(cfg.Persistence, logger)
	if err != nil {
		return nil, fmt.Errorf("build stores: %w", err)
	}

	publisher := buildPublisher(cfg.LokiPush, cfg.Engine, logger)

	eng := engine.New(engine.Options{
		Source:                   source,
		RuleLoader:               rules.NewLoader(cfg.Engine.RulesDir),
		Classifiers:              buildClassifiers(),
		Aggregator:               agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:                 noop.New(),
		SignalStore:              stores.Signal,
		EventAssessmentStore:     stores.EventAssessment,
		EventAssessmentPublisher: publisher,
		InsightStore:             stores.Insight,
	})

	worker := engine.NewWorker(engine.WorkerOptions{
		Engine:       eng,
		StateStore:   stores.RunState,
		JobName:      cfg.Engine.JobName,
		PollInterval: cfg.Engine.PollInterval,
		Lookback:     cfg.Engine.InitialLookback,
		Overlap:      cfg.Engine.QueryOverlap,
		IgnoreState:  cfg.Engine.IgnoreRunState,
	})

	return worker, nil
}

// buildClassifiers returns the ordered list of signal classifiers. Order
// matters: classifiers are applied in sequence and results are merged.
func buildClassifiers() []ports.SignalClassifier {
	bayesClassifier := bayes.NewClassifier(
		bayes.DefaultModel(),
		feat.NewDefaultExtractor(),
	)
	return []ports.SignalClassifier{
		rulecls.NewClassifier(),
		bayesClassifier,
		keywordscoring.NewClassifier(),
	}
}

// buildEventSource constructs the Loki event source from config.
func buildEventSource(cfg config.LokiConfig, logger *slog.Logger) ports.EventSource {
	client := lokisrc.NewHTTPClient(
		cfg.BaseURL,
		cfg.Username,
		cfg.Password,
		cfg.TenantID,
		cfg.Timeout,
		cfg.InsecureSkipTLS,
	)

	logger.Info("event source configured",
		"backend", "loki",
		"base_url", cfg.BaseURL,
		"query", cfg.EventsQuery,
		"limit", cfg.EventsLimit,
		"direction", cfg.QueryDirection,
		"has_tenant", cfg.TenantID != "",
		"has_auth", cfg.Username != "" || cfg.Password != "",
		"timeout", cfg.Timeout,
		"lookback", cfg.Lookback,
		"insecure_skip_tls", cfg.InsecureSkipTLS,
	)

	return lokisrc.New(client, cfg.EventsQuery, cfg.EventsLimit, cfg.QueryDirection, cfg.Lookback, nil)
}

// stores groups the four store interfaces together to avoid a 4-value return.
type stores struct {
	Signal          ports.SignalStore
	EventAssessment ports.EventAssessmentStore
	Insight         ports.InsightStore
	RunState        ports.RunStateStore
}

// buildStores constructs the persistence backend specified by config.
func buildStores(cfg config.PersistenceConfig, logger *slog.Logger) (stores, error) {
	switch cfg.Backend {
	case "s3":
		return buildS3Stores(cfg, logger)
	default:
		logger.Info("persistence configured", "backend", "memory")
		return stores{
			Signal:          memstore.NewSignalStore(),
			EventAssessment: memstore.NewEventAssessmentStore(),
			Insight:         memstore.NewInsightStore(),
			RunState:        memstore.NewRunStateStore(),
		}, nil
	}
}

func buildS3Stores(cfg config.PersistenceConfig, logger *slog.Logger) (stores, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		return stores{}, fmt.Errorf("load aws config: %w", err)
	}

	store := s3store.New(
		awss3.NewFromConfig(awsCfg),
		cfg.S3Bucket,
		cfg.S3Prefix,
		cfg.S3Env,
		nil,
	)

	logger.Info("persistence configured",
		"backend", "s3",
		"bucket", cfg.S3Bucket,
		"prefix", cfg.S3Prefix,
		"env", cfg.S3Env,
		"region", cfg.AWSRegion,
	)

	return stores{
		Signal:          store,
		EventAssessment: store,
		Insight:         store,
		RunState:        store,
	}, nil
}

// buildPublisher constructs the Loki assessment publisher, or returns nil if
// publishing is disabled.
func buildPublisher(cfg config.LokiPushConfig, engCfg config.EngineConfig, logger *slog.Logger) ports.EventAssessmentPublisher {
	if !cfg.Enabled {
		logger.Info("assessment publisher disabled")
		return nil
	}

	pushClient := lokipersist.NewHTTPPushClient(
		cfg.BaseURL,
		cfg.Username,
		cfg.Password,
		cfg.TenantID,
		cfg.Timeout,
		cfg.InsecureSkipTLS,
	)

	logger.Info("assessment publisher configured",
		"backend", "loki",
		"base_url", cfg.BaseURL,
		"has_tenant", cfg.TenantID != "",
		"has_auth", cfg.Username != "" || cfg.Password != "",
		"timeout", cfg.Timeout,
		"insecure_skip_tls", cfg.InsecureSkipTLS,
	)

	return lokipersist.NewEventAssessmentPublisher(
		pushClient,
		engCfg.JobName,
		engCfg.AppEnv,
		nil,
	)
}
