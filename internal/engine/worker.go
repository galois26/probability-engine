package engine

import (
	"context"
	"log"
	"time"

	"probability-engine/internal/ports"
)

type runner interface {
	Run(ctx context.Context, from time.Time) (RunResult, error)
}

type Worker struct {
	engine       runner
	stateStore   ports.RunStateStore
	jobName      string
	pollInterval time.Duration
	lookback     time.Duration
	clock        Clock
}

type WorkerOptions struct {
	Engine       runner
	StateStore   ports.RunStateStore
	JobName      string
	PollInterval time.Duration
	Lookback     time.Duration
	Clock        Clock
}

func NewWorker(opts WorkerOptions) *Worker {
	c := opts.Clock
	if c == nil {
		c = realClock{}
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 60 * time.Second
	}
	if opts.Lookback <= 0 {
		opts.Lookback = 5 * time.Minute
	}
	if opts.JobName == "" {
		opts.JobName = "probability-engine"
	}

	return &Worker{
		engine:       opts.Engine,
		stateStore:   opts.StateStore,
		jobName:      opts.JobName,
		pollInterval: opts.PollInterval,
		lookback:     opts.Lookback,
		clock:        c,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		if err := w.runOnce(ctx); err != nil {
			log.Printf("worker: run failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) error {
	now := w.clock.Now().UTC()

	from, err := w.stateStore.LoadLastRun(ctx, w.jobName)
	if err != nil {
		return err
	}
	if from.IsZero() {
		from = now.Add(-w.lookback)
	}

	log.Printf("worker: starting run from=%s to=%s", from.Format(time.RFC3339), now.Format(time.RFC3339))

	res, err := w.engine.Run(ctx, from)
	if err != nil {
		return err
	}
	if res.Events == 0 {
		log.Printf("worker: no new events found in window (from=%s to=%s)",
			from.Format(time.RFC3339),
			now.Format(time.RFC3339),
		)
	}

	if err := w.stateStore.SaveLastRun(ctx, w.jobName, now); err != nil {
		return err
	}

	log.Printf("worker: completed run events=%d signals=%d insights=%d", res.Events, res.Signals, res.Insights)
	return nil
}
