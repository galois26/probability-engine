package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/galois/probability-engine/internal/app"
	"github.com/galois/probability-engine/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("probability-engine starting")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	worker, err := app.Build(cfg, logger)
	if err != nil {
		slog.Error("failed to build worker", "error", err)
		os.Exit(1)
	}

	if err := worker.Run(context.Background()); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
