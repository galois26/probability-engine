package loki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"probability-engine/internal/testutil"
)

func TestHTTPClient_QueryRange_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if got := r.URL.Query().Get("query"); got != `{job="multi-ingester"}` {
			t.Fatalf("query = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q", got)
		}
		if got := r.URL.Query().Get("direction"); got != "forward" {
			t.Fatalf("direction = %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "streams",
				"result": [
					{
						"stream": {
							"job": "multi-ingester",
							"host": "ingester-1"
						},
						"values": [
							["1711126800000000000", "{\"id\":\"ev1\",\"title\":\"one\"}"],
							["1711126860000000000", "{\"id\":\"ev2\",\"title\":\"two\"}"]
						]
					}
				]
			}
		}`))
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "", "", "", 15*time.Second, false)

	got, err := client.QueryRange(
		context.Background(),
		QuerySpec{
			LogQL:     `{job="multi-ingester"}`,
			From:      time.Unix(1711126700, 0).UTC(),
			To:        time.Unix(1711126900, 0).UTC(),
			Limit:     100,
			Direction: "forward",
		},
	)
	if err != nil {
		t.Fatalf("QueryRange() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	if got[0].Line != `{"id":"ev1","title":"one"}` {
		t.Fatalf("got[0].Line = %q", got[0].Line)
	}
	if got[1].Line != `{"id":"ev2","title":"two"}` {
		t.Fatalf("got[1].Line = %q", got[1].Line)
	}

	testutil.SameStrings(t, mapKeysString(got[0].Labels), []string{"host", "job"})

	want0 := time.Unix(0, 1711126800000000000).UTC()
	want1 := time.Unix(0, 1711126860000000000).UTC()

	if !got[0].Timestamp.Equal(want0) {
		t.Fatalf("got[0].Timestamp = %s, want %s", got[0].Timestamp, want0)
	}
	if !got[1].Timestamp.Equal(want1) {
		t.Fatalf("got[1].Timestamp = %s, want %s", got[1].Timestamp, want1)
	}
}

func TestHTTPClient_QueryRange_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad loki request", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "", "", "", 15*time.Second, false)

	_, err := client.QueryRange(
		context.Background(),
		QuerySpec{
			LogQL:     `{job="multi-ingester"}`,
			From:      time.Now().UTC().Add(-time.Minute),
			To:        time.Now().UTC(),
			Limit:     100,
			Direction: "forward",
		},
	)
	if err == nil {
		t.Fatal("QueryRange() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "status=400") {
		t.Fatalf("error = %q, want status=400", err.Error())
	}
}

func TestLokiQueryRangeResponse_ToEntries_BadTimestamp(t *testing.T) {
	resp := lokiQueryRangeResponse{}
	resp.Data.Result = []lokiStreamResult{
		{
			Stream: map[string]string{"job": "multi-ingester"},
			Values: [][]string{
				{"not-a-timestamp", `{"id":"ev1"}`},
			},
		},
	}

	_, err := resp.toEntries()
	if err == nil {
		t.Fatal("toEntries() error = nil, want non-nil")
	}
}

func mapKeysString(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
