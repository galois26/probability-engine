package loki

import (
	"context"
	"testing"
	"time"

	"github.com/galois/probability-engine/internal/domain"
	"github.com/galois/probability-engine/internal/testutil"
)

func TestDecodeEvent_ValidJSON(t *testing.T) {
	ts := time.Date(2026, 3, 22, 17, 5, 0, 0, time.UTC)

	entry := Entry{
		Timestamp: ts,
		Line: `{
			"id": "fba17af004c947601f63111b39ca35dd",
			"labels": {
				"category": "business",
				"ingester": "newsdata",
				"source": "newsdata",
				"source_id": "themarketsdaily",
				"source_name": "Markets Daily"
			},
			"published": "2026-03-22T17:02:44Z",
			"source": "newsdata",
			"summary": "Figure Technology Solutions...",
			"title": "Promising Blockchain Stocks Worth Watching – March 22nd",
			"url": "https://www.themarketsdaily.com/2026/03/22/promising-blockchain-stocks-worth-watching-march-22nd.html"
		}`,
	}

	got, ok, err := decodeEvent(entry)
	if err != nil {
		t.Fatalf("decodeEvent() error = %v", err)
	}
	if !ok {
		t.Fatal("decodeEvent() ok = false, want true")
	}

	if got.ID != "fba17af004c947601f63111b39ca35dd" {
		t.Fatalf("ID = %q", got.ID)
	}
	if got.Source != "newsdata" {
		t.Fatalf("Source = %q", got.Source)
	}
	if got.Title != "Promising Blockchain Stocks Worth Watching – March 22nd" {
		t.Fatalf("Title = %q", got.Title)
	}
	if got.URL != "https://www.themarketsdaily.com/2026/03/22/promising-blockchain-stocks-worth-watching-march-22nd.html" {
		t.Fatalf("URL = %q", got.URL)
	}
	if got.Country != "" {
		t.Fatalf("Country = %q, want empty", got.Country)
	}
	if got.Published.Format(time.RFC3339) != "2026-03-22T17:02:44Z" {
		t.Fatalf("Published = %s", got.Published.Format(time.RFC3339))
	}

	testutil.SameStrings(t, mapKeys(got.Labels), []string{
		"category", "ingester", "source", "source_id", "source_name",
	})
}

func TestDecodeEvent_FallbackPublishedAndSourceFromLabels(t *testing.T) {
	ts := time.Date(2026, 3, 22, 17, 5, 0, 0, time.UTC)

	entry := Entry{
		Timestamp: ts,
		Line: `{
			"id": "ev1",
			"labels": {
				"source": "newsdata",
				"category": "business"
			},
			"title": "Example title",
			"summary": "Example summary",
			"url": "https://example.com/1"
		}`,
	}

	got, ok, err := decodeEvent(entry)
	if err != nil {
		t.Fatalf("decodeEvent() error = %v", err)
	}
	if !ok {
		t.Fatal("decodeEvent() ok = false, want true")
	}

	if got.Source != "newsdata" {
		t.Fatalf("Source = %q, want newsdata", got.Source)
	}
	if !got.Published.Equal(ts) {
		t.Fatalf("Published = %s, want %s", got.Published, ts)
	}
}

func TestDecodeEvent_SkipsInvalidOrIncompleteLines(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "not json",
			line: `level=info msg="not an event"`,
		},
		{
			name: "missing id",
			line: `{"title":"hello","summary":"world"}`,
		},
		{
			name: "missing title",
			line: `{"id":"ev1","summary":"world"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := decodeEvent(Entry{
				Timestamp: time.Now().UTC(),
				Line:      tt.line,
			})
			if err != nil {
				t.Fatalf("decodeEvent() error = %v", err)
			}
			if ok {
				t.Fatalf("decodeEvent() ok = true, want false; got=%+v", got)
			}
		})
	}
}

func TestSource_FetchEvents_DedupesAndSorts(t *testing.T) {
	now := time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC)

	client := stubClient{
		entries: []Entry{
			{
				Timestamp: now.Add(-1 * time.Minute),
				Line: `{
					"id":"ev2",
					"source":"gta",
					"title":"Later event",
					"summary":"second",
					"url":"https://example.com/2",
					"published":"2026-03-22T17:59:00Z",
					"labels":{"category":"trade"}
				}`,
			},
			{
				Timestamp: now.Add(-2 * time.Minute),
				Line: `{
					"id":"ev1",
					"source":"newsdata",
					"title":"Earlier event",
					"summary":"first",
					"url":"https://example.com/1",
					"published":"2026-03-22T17:58:00Z",
					"labels":{"category":"business"}
				}`,
			},
			{
				Timestamp: now.Add(-30 * time.Second),
				Line: `{
					"id":"ev2",
					"source":"gta",
					"title":"Duplicate event",
					"summary":"dup",
					"url":"https://example.com/2-dup",
					"published":"2026-03-22T17:59:30Z",
					"labels":{"category":"trade"}
				}`,
			},
			{
				Timestamp: now.Add(-10 * time.Second),
				Line:      `not json`,
			},
		},
	}

	src := New(client, `{job="multi-ingester"}`, 100, "forward", 48, func() time.Time { return now })

	got, err := src.FetchEvents(context.Background(), now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("FetchEvents() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	if got[0].ID != "ev1" {
		t.Fatalf("got[0].ID = %q, want ev1", got[0].ID)
	}
	if got[1].ID != "ev2" {
		t.Fatalf("got[1].ID = %q, want ev2", got[1].ID)
	}

	if !got[0].Published.Before(got[1].Published) {
		t.Fatalf("events not sorted by Published: %s then %s", got[0].Published, got[1].Published)
	}
}

func TestSource_FetchEvents_PropagatesClientError(t *testing.T) {
	src := New(errorClient{}, `{job="multi-ingester"}`, 100, "forward", 48, func() time.Time {
		return time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC)
	})

	_, err := src.FetchEvents(context.Background(), time.Date(2026, 3, 22, 17, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("FetchEvents() error = nil, want non-nil")
	}
}

type stubClient struct {
	entries []Entry
}

func (s stubClient) QueryRange(ctx context.Context, spec QuerySpec) ([]Entry, error) {
	return append([]Entry(nil), s.entries...), nil
}

type errorClient struct{}

func (errorClient) QueryRange(ctx context.Context, spec QuerySpec) ([]Entry, error) {
	return nil, context.DeadlineExceeded
}

func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

var _ domain.Event
