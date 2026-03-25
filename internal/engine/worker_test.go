package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorker_RunOnce_UsesLookbackWhenNoWatermark(t *testing.T) {
	clock := fixedClock{
		now: time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
	}

	state := &stubRunStateStore{}
	eng := &stubRunnerEngine{}

	w := NewWorker(WorkerOptions{
		Engine:       eng,
		StateStore:   state,
		JobName:      "probability-engine",
		PollInterval: time.Minute,
		Lookback:     15 * time.Minute,
		Clock:        clock,
	})

	err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("runOnce() error = %v", err)
	}

	wantFrom := clock.now.Add(-15 * time.Minute)
	if !eng.lastFrom.Equal(wantFrom) {
		t.Fatalf("run from = %s, want %s", eng.lastFrom, wantFrom)
	}

	if !state.saved.Equal(clock.now) {
		t.Fatalf("saved watermark = %s, want %s", state.saved, clock.now)
	}
	if state.savedJob != "probability-engine" {
		t.Fatalf("saved job = %q", state.savedJob)
	}
}

func TestWorker_RunOnce_UsesStoredWatermark(t *testing.T) {
	clock := fixedClock{
		now: time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
	}

	stored := time.Date(2026, 3, 22, 17, 45, 0, 0, time.UTC)
	state := &stubRunStateStore{loaded: stored}
	eng := &stubRunnerEngine{}

	w := NewWorker(WorkerOptions{
		Engine:       eng,
		StateStore:   state,
		JobName:      "probability-engine",
		PollInterval: time.Minute,
		Lookback:     15 * time.Minute,
		Clock:        clock,
	})

	err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("runOnce() error = %v", err)
	}

	if !eng.lastFrom.Equal(stored) {
		t.Fatalf("run from = %s, want %s", eng.lastFrom, stored)
	}

	if !state.saved.Equal(clock.now) {
		t.Fatalf("saved watermark = %s, want %s", state.saved, clock.now)
	}
}

func TestWorker_RunOnce_DoesNotSaveWatermarkOnEngineFailure(t *testing.T) {
	clock := fixedClock{
		now: time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
	}

	state := &stubRunStateStore{}
	eng := &stubRunnerEngine{err: errors.New("boom")}

	w := NewWorker(WorkerOptions{
		Engine:       eng,
		StateStore:   state,
		JobName:      "probability-engine",
		PollInterval: time.Minute,
		Lookback:     15 * time.Minute,
		Clock:        clock,
	})

	err := w.runOnce(context.Background())
	if err == nil {
		t.Fatal("runOnce() error = nil, want non-nil")
	}

	if !state.saved.IsZero() {
		t.Fatalf("saved watermark = %s, want zero", state.saved)
	}
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time { return f.now }

type stubRunStateStore struct {
	loaded   time.Time
	saved    time.Time
	savedJob string
}

func (s *stubRunStateStore) LoadLastRun(ctx context.Context, job string) (time.Time, error) {
	return s.loaded, nil
}

func (s *stubRunStateStore) SaveLastRun(ctx context.Context, job string, t time.Time) error {
	s.saved = t
	s.savedJob = job
	return nil
}

type stubRunnerEngine struct {
	lastFrom time.Time
	err      error
}

func (s *stubRunnerEngine) Run(ctx context.Context, from time.Time) (RunResult, error) {
	s.lastFrom = from
	if s.err != nil {
		return RunResult{}, s.err
	}
	return RunResult{Events: 1, Signals: 1, Insights: 1}, nil
}
