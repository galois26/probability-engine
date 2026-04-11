package loki

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type HTTPClient struct {
	baseURL    string
	username   string
	password   string
	tenantID   string
	httpClient *http.Client
}

func NewHTTPClient(baseURL, username, password, tenantID string, timeout time.Duration, insecureSkipTLS bool) *HTTPClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecureSkipTLS {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &HTTPClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		tenantID: tenantID,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *HTTPClient) QueryRange(ctx context.Context, spec QuerySpec) ([]Entry, error) {
	u, err := url.Parse(c.baseURL + "/loki/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("parse loki url: %w", err)
	}

	limit := spec.Limit
	if limit <= 0 {
		limit = 1000
	}

	direction := spec.Direction
	if direction == "" {
		direction = "forward"
	}

	q := u.Query()
	q.Set("query", spec.LogQL)
	q.Set("start", strconv.FormatInt(spec.From.UTC().UnixNano(), 10))
	q.Set("end", strconv.FormatInt(spec.To.UTC().UnixNano(), 10))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("direction", direction)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build loki request: %w", err)
	}

	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if c.tenantID != "" {
		req.Header.Set("X-Scope-OrgID", c.tenantID)
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
