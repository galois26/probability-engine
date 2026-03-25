package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string, httpClient *http.Client) *HTTPClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *HTTPClient) QueryRange(ctx context.Context, query string, start, end time.Time, limit int) ([]Entry, error) {
	u, err := url.Parse(c.baseURL + "/loki/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("parse loki url: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.UTC().UnixNano(), 10))
	q.Set("end", strconv.FormatInt(end.UTC().UnixNano(), 10))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("direction", "forward")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build loki request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do loki request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read loki response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("loki query failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var parsed lokiQueryRangeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal loki response: %w", err)
	}

	return parsed.toEntries()
}

type lokiQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string             `json:"resultType"`
		Result     []lokiStreamResult `json:"result"`
	} `json:"data"`
}

type lokiStreamResult struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func (r lokiQueryRangeResponse) toEntries() ([]Entry, error) {
	var out []Entry
	for _, stream := range r.Data.Result {
		for _, pair := range stream.Values {
			if len(pair) != 2 {
				continue
			}
			ns, err := strconv.ParseInt(pair[0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse loki timestamp: %w", err)
			}
			out = append(out, Entry{
				Timestamp: time.Unix(0, ns).UTC(),
				Line:      pair[1],
				Labels:    stream.Stream,
			})
		}
	}
	return out, nil
}
