package probability

import (
	"fmt"
	"log"
	"time"

	agg "github.com/galois26/probability-engine/internal/aggregation/rolling"
	bayes "github.com/galois26/probability-engine/internal/classification/bayes"
	feat "github.com/galois26/probability-engine/internal/classification/features"
	"github.com/galois26/probability-engine/internal/classification/keywordscoring"
	rulecls "github.com/galois26/probability-engine/internal/classification/rules"
	"github.com/galois26/probability-engine/internal/engine"
	noop "github.com/galois26/probability-engine/internal/enrichment/noop"
	"github.com/galois26/probability-engine/internal/ports"
	rules "github.com/galois26/probability-engine/internal/rules"
)

// NewDefaultEngine exposes a production-ready in-process probability engine.
//
// It intentionally does NOT wire:
//   - Loki source
//   - Loki publisher
//   - S3 persistence
//   - run state
//   - polling worker
//
// multi-ingester owns runtime orchestration and persistence.
func NewDefaultEngine(cfg Config) (Engine, error) {
	if cfg.RulesDir == "" {
		return nil, fmt.Errorf("probability: rules dir must not be empty")
	}

	eng := engine.New(engine.Options{
		RuleLoader:  rules.NewLoader(cfg.RulesDir),
		Classifiers: buildClassifiers(),
		Aggregator:  agg.NewAggregator(2, 1.2, 24*time.Hour),
		Enricher:    noop.New(),

		// Deliberately nil:
		// Source, stores, publisher, and run state belong to the application runtime.
	})
	log.Printf("probability: NewDefaultEngine rules_dir=%s engine_version=%s rule_version=%s",
		cfg.RulesDir,
		cfg.EngineVersion,
		cfg.RuleVersion,
	)
	return NewEngineAdapter(eng, cfg.EngineVersion, cfg.RuleVersion), nil
}

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
