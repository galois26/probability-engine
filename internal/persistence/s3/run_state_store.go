package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"time"
)

type runState struct {
	Job       string    `json:"job"`
	LastRun   time.Time `json:"lastRun"`
	UpdatedAt time.Time `json:"updatedAt"`
	Schema    string    `json:"schema"`
}

func (s *Store) LoadLastRun(ctx context.Context, job string) (time.Time, error) {
	key := s.runStateKey(job)

	var st runState
	err := s.getJSON(ctx, key, &st)
	if err != nil {
		if isNotFoundErr(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return st.LastRun.UTC(), nil
}

func (s *Store) SaveLastRun(ctx context.Context, job string, t time.Time) error {
	now := s.clock.Now()
	st := runState{
		Job:       job,
		LastRun:   t.UTC(),
		UpdatedAt: now,
		Schema:    "v1",
	}
	log.Printf("s3 store: writing state key=%s", s.runStateKey(job))
	return s.putJSON(ctx, s.runStateKey(job), st)
}

func (s *Store) runStateKey(job string) string {
	return path.Join(s.prefix, s.env, "state", job+".json")
}

// optional compile-time JSON sanity if needed elsewhere
var _ = json.RawMessage{}
var _ = fmt.Sprintf
