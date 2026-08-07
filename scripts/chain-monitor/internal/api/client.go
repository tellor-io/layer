package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const reportsByAggregateLimit = 1000

// Client talks to a Cosmos REST / gRPC-gateway (LCD) base URL.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient constructs an LCD client. baseURL must be an absolute http(s) URL.
// Empty baseURL returns a nil client (lookups disabled).
func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, nil
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, fmt.Errorf("api url must be absolute http(s): %q", baseURL)
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// BaseURL returns the configured LCD base.
func (c *Client) BaseURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

// LooksLikeTendermint reports whether the base likely points at CometBFT RPC
// (:26657), which does not serve /tellor-io/layer/... routes.
func LooksLikeTendermint(base string) bool {
	return strings.Contains(base, ":26657")
}

type reportsByAggregateResponse struct {
	MicroReports []microReport `json:"microReports"`
}

type microReport struct {
	Reporter string `json:"reporter"`
}

// ReportsByAggregate returns reporter addresses that submitted for the aggregate.
func (c *Client) ReportsByAggregate(ctx context.Context, queryID string, timestamp uint64) ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("api client not configured")
	}
	queryID = strings.TrimSpace(queryID)
	if queryID == "" {
		return nil, fmt.Errorf("query_id is required")
	}

	endpoint := fmt.Sprintf("%s/tellor-io/layer/oracle/get_reports_by_aggregate/%s/%d",
		c.baseURL, url.PathEscape(queryID), timestamp)
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}
	q := u.Query()
	q.Set("pagination.limit", strconv.Itoa(reportsByAggregateLimit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get_reports_by_aggregate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		hint := ""
		if LooksLikeTendermint(c.baseURL) {
			hint = " (api.url looks like Tendermint :26657; set api.url or LAYER_API_URL to LCD, e.g. :1317)"
		}
		return nil, fmt.Errorf("get_reports_by_aggregate http %d%s: %s",
			resp.StatusCode, hint, truncate(string(body), 200))
	}

	var result reportsByAggregateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode get_reports_by_aggregate: %w", err)
	}

	out := make([]string, 0, len(result.MicroReports))
	for _, r := range result.MicroReports {
		reporter := strings.TrimSpace(r.Reporter)
		if reporter != "" {
			out = append(out, reporter)
		}
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
