package loki

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type PushClient interface {
	Push(ctx context.Context, req PushRequest) error
}

type HTTPPushClient struct {
	baseURL    string
	username   string
	password   string
	tenantID   string
	httpClient *http.Client
}

func NewHTTPPushClient(
	baseURL string,
	username string,
	password string,
	tenantID string,
	timeout time.Duration,
	insecureSkipTLS bool,
) *HTTPPushClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecureSkipTLS {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &HTTPPushClient{
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

func (c *HTTPPushClient) Push(ctx context.Context, req PushRequest) error {
	if len(req.Streams) == 0 {
		return nil
	}

	body, err := marshalPushRequest(req)
	if err != nil {
		return fmt.Errorf("marshal push request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/loki/api/v1/push",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build loki push request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	if c.username != "" || c.password != "" {
		httpReq.SetBasicAuth(c.username, c.password)
	}
	if c.tenantID != "" {
		httpReq.Header.Set("X-Scope-OrgID", c.tenantID)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do loki push request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki push failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

func marshalPushRequest(req PushRequest) ([]byte, error) {
	type wireStream struct {
		Stream map[string]string `json:"stream"`
		Values [][]string        `json:"values"`
	}

	type wireRequest struct {
		Streams []wireStream `json:"streams"`
	}

	out := wireRequest{
		Streams: make([]wireStream, 0, len(req.Streams)),
	}

	for _, s := range req.Streams {
		values := make([][]string, 0, len(s.Values))
		for _, v := range s.Values {
			values = append(values, []string{
				fmt.Sprintf("%d", v.Timestamp.UTC().UnixNano()),
				v.Line,
			})
		}

		out.Streams = append(out.Streams, wireStream{
			Stream: s.Stream,
			Values: values,
		})
	}

	return json.Marshal(out)
}
