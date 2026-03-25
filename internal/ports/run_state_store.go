package ports

import (
	"context"
	"time"
)

type RunStateStore interface {
	LoadLastRun(ctx context.Context, job string) (time.Time, error)
	SaveLastRun(ctx context.Context, job string, t time.Time) error
}
