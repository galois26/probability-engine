package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"probability-engine/internal/domain"
)

type Client interface {
	QueryRange(ctx context.Context, query string, start, end time.Time, limit int) ([]Entry, error)
}

type Entry struct {
	Timestamp time.Time
	Line      string
	Labels    map[string]string
}

type Source struct {
	client Client
	query  string
	limit  int
	now    func() time.Time
}

func New(client Client, query string, limit int, now func() time.Time) *Source {
	if limit <= 0 {
		limit = 1000
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Source{
		client: client,
		query:  query,
		limit:  limit,
		now:    now,
	}
}

func (s *Source) FetchEvents(ctx context.Context, from time.Time) ([]domain.Event, error) {
	to := s.now()

	entries, err := s.client.QueryRange(ctx, s.query, from.UTC(), to.UTC(), s.limit)
	if err != nil {
		return nil, fmt.Errorf("query loki: %w", err)
	}
	if len(entries) == 0 {
		log.Printf("loki source: no entries found for query=%s", s.query)
	}

	out := make([]domain.Event, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, e := range entries {
		ev, ok, err := decodeEvent(e)
		if err != nil {
			return nil, fmt.Errorf("decode loki event: %w", err)
		}
		if !ok {
			continue
		}
		if ev.ID == "" {
			continue
		}
		if _, exists := seen[ev.ID]; exists {
			continue
		}
		seen[ev.ID] = struct{}{}
		out = append(out, ev)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Published.Before(out[j].Published)
	})

	return out, nil
}

type wireEvent struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	Title     string            `json:"title"`
	Summary   string            `json:"summary"`
	URL       string            `json:"url"`
	Published time.Time         `json:"published"`
	Country   string            `json:"country"`
	Labels    map[string]string `json:"labels"`
}

func decodeEvent(e Entry) (domain.Event, bool, error) {
	var raw wireEvent
	if err := json.Unmarshal([]byte(e.Line), &raw); err != nil {
		// ignore non-event lines if Loki contains mixed content
		return domain.Event{}, false, nil
	}

	ev := domain.Event{
		ID:        raw.ID,
		Source:    raw.Source,
		Title:     raw.Title,
		Summary:   raw.Summary,
		URL:       raw.URL,
		Published: raw.Published,
		Country:   raw.Country,
		Labels:    cloneMap(raw.Labels),
	}

	if ev.Published.IsZero() {
		ev.Published = e.Timestamp.UTC()
	}

	return ev, true, nil
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
