package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"probability-engine/internal/domain"
)

type Client interface {
	QueryRange(ctx context.Context, spec QuerySpec) ([]Entry, error)
}

type QuerySpec struct {
	LogQL     string
	From      time.Time
	To        time.Time
	Limit     int
	Direction string
	Lookback  time.Duration
}

type Entry struct {
	Timestamp time.Time
	Line      string
	Labels    map[string]string
}

type Source struct {
	client    Client
	query     string
	limit     int
	direction string
	lookback  time.Duration
	now       func() time.Time
}

func New(client Client, query string, limit int, direction string, lookback time.Duration, now func() time.Time) *Source {
	if limit <= 0 {
		limit = 1000
	}
	if direction == "" {
		direction = "forward"
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if lookback <= 0 {
		lookback = 48 * time.Hour
	}
	return &Source{
		client:    client,
		query:     query,
		limit:     limit,
		direction: direction,
		lookback:  lookback,
		now:       now,
	}
}

func (s *Source) FetchEvents(ctx context.Context, from time.Time) ([]domain.Event, error) {
	spec := QuerySpec{
		LogQL:     s.query,
		From:      from.UTC(),
		To:        s.now().UTC(),
		Limit:     s.limit,
		Direction: s.direction,
		Lookback:  48 * time.Hour, // TODO: make this configurable if needed
	}
	return s.FetchEventsWithQuery(ctx, spec)
}

func (s *Source) FetchEventsWithQuery(ctx context.Context, spec QuerySpec) ([]domain.Event, error) {
	if spec.LogQL == "" {
		return nil, fmt.Errorf("loki query is empty")
	}
	if spec.Limit <= 0 {
		spec.Limit = s.limit
		if spec.Limit <= 0 {
			spec.Limit = 1000
		}
	}
	if spec.Direction == "" {
		spec.Direction = s.direction
		if spec.Direction == "" {
			spec.Direction = "forward"
		}
	}
	if spec.To.IsZero() {
		spec.To = s.now().UTC()
	}

	entries, err := s.client.QueryRange(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("query loki: %w", err)
	}

	log.Printf(
		"loki source: fetched entries=%d query=%s from=%s to=%s direction=%s limit=%d",
		len(entries),
		spec.LogQL,
		spec.From.Format(time.RFC3339),
		spec.To.Format(time.RFC3339),
		spec.Direction,
		spec.Limit,
	)

	out := make([]domain.Event, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, e := range entries {
		ev, ok, err := decodeEvent(e)
		if err != nil {
			return nil, fmt.Errorf("decode loki event: %w", err)
		}
		if !ok {
			log.Printf("loki source: skipped line=%q", truncate(e.Line, 200))
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
	Labels    map[string]string `json:"labels"`
}

func decodeEvent(e Entry) (domain.Event, bool, error) {
	var raw wireEvent
	if err := json.Unmarshal([]byte(e.Line), &raw); err != nil {
		return domain.Event{}, false, nil
	}

	if raw.ID == "" || raw.Title == "" {
		return domain.Event{}, false, nil
	}

	labels := cloneMap(raw.Labels)
	normalizeLabels(labels)

	ev := domain.Event{
		ID:        raw.ID,
		Source:    raw.Source,
		Title:     raw.Title,
		Summary:   raw.Summary,
		URL:       raw.URL,
		Published: raw.Published,
		Country:   "",
		Labels:    labels,
	}

	if ev.Published.IsZero() {
		ev.Published = e.Timestamp.UTC()
	}
	if ev.Source == "" {
		ev.Source = raw.Labels["source"]
	}

	return ev, true, nil
}

func normalizeLabels(labels map[string]string) {
	if labels == nil {
		return
	}

	// Normalize publisher/source naming
	if labels["source"] == "" && labels["news_source"] != "" {
		labels["source"] = labels["news_source"]
	}

	// Map known numeric category codes into the semantic categories
	// the Bayes model expects.
	switch strings.TrimSpace(strings.ToLower(labels["category"])) {
	case "121":
		// TODO: verify this mapping against your upstream feed.
		// Temporary placeholder so we can confirm whether label drift
		// is the reason Bayes stopped contributing.
		labels["category"] = "crypto"
	}
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
