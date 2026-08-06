package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client talks to one or more CometBFT HTTP RPC endpoints with failover.
type Client struct {
	urls        []string
	httpClient  *http.Client
	minInterval time.Duration

	mu       sync.Mutex
	lastCall time.Time
	urlIndex int
}

// NewClient constructs an RPC client. urls must be non-empty absolute URLs.
func NewClient(urls []string, timeout, minInterval time.Duration) (*Client, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("at least one RPC URL is required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	cleaned := make([]string, len(urls))
	copy(cleaned, urls)

	return &Client{
		urls: cleaned,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		minInterval: minInterval,
	}, nil
}

// LatestHeight returns the node's latest committed block height.
func (c *Client) LatestHeight(ctx context.Context) (uint64, error) {
	var resp statusResponse
	if err := c.call(ctx, "status", nil, &resp); err != nil {
		return 0, err
	}
	if resp.Error != nil {
		return 0, fmt.Errorf("status rpc error: %s", resp.Error.Message)
	}
	h, err := strconv.ParseUint(resp.Result.SyncInfo.LatestBlockHeight, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse latest height: %w", err)
	}
	return h, nil
}

// GetBlockResult fetches block header + block_results for a height.
func (c *Client) GetBlockResult(ctx context.Context, height uint64) (*BlockResult, error) {
	params := map[string]interface{}{
		"height": strconv.FormatUint(height, 10),
	}

	var blockResp blockResponse
	if err := c.call(ctx, "block", params, &blockResp); err != nil {
		return nil, err
	}
	if blockResp.Error != nil {
		if isNotFound(blockResp.Error) {
			return nil, fmt.Errorf("%w: height %d: %s", ErrBlockNotFound, height, blockResp.Error.Message)
		}
		return nil, fmt.Errorf("block rpc error: %s", blockResp.Error.Message)
	}
	if blockResp.Result.Block.Header.Height == "" {
		return nil, fmt.Errorf("%w: height %d (empty header)", ErrBlockNotFound, height)
	}

	var resultsResp blockResultsResponse
	if err := c.call(ctx, "block_results", params, &resultsResp); err != nil {
		return nil, err
	}
	if resultsResp.Error != nil {
		if isNotFound(resultsResp.Error) {
			return nil, fmt.Errorf("%w: height %d: %s", ErrBlockNotFound, height, resultsResp.Error.Message)
		}
		return nil, fmt.Errorf("block_results rpc error: %s", resultsResp.Error.Message)
	}

	return &BlockResult{
		Height:              height,
		Time:                blockResp.Result.Block.Header.Time,
		ChainID:             blockResp.Result.Block.Header.ChainID,
		FinalizeBlockEvents: resultsResp.Result.FinalizeBlockEvents,
		TxsResults:          resultsResp.Result.TxsResults,
	}, nil
}

// TotalVotingPower sums validator voting power from the validators endpoint.
func (c *Client) TotalVotingPower(ctx context.Context) (uint64, error) {
	var resp validatorsResponse
	if err := c.call(ctx, "validators", nil, &resp); err != nil {
		return 0, err
	}
	if resp.Error != nil {
		return 0, fmt.Errorf("validators rpc error: %s", resp.Error.Message)
	}
	var total uint64
	for _, v := range resp.Result.Validators {
		p, err := strconv.ParseUint(v.VotingPower, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse voting power %q: %w", v.VotingPower, err)
		}
		total += p
	}
	return total, nil
}

func (c *Client) call(ctx context.Context, method string, params interface{}, out interface{}) error {
	c.throttle()

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		ID:      1,
		Params:  params,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	start := c.currentURLIndex()
	for i := 0; i < len(c.urls); i++ {
		idx := (start + i) % len(c.urls)
		url := c.urls[idx]

		if err := c.doPost(ctx, url, payload, out); err != nil {
			lastErr = fmt.Errorf("%s via %s: %w", method, url, err)
			c.advanceURL()
			continue
		}
		// Prefer a working URL next time.
		c.setURLIndex(idx)
		return nil
	}
	return lastErr
}

func (c *Client) doPost(ctx context.Context, url string, payload []byte, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

func (c *Client) throttle() {
	if c.minInterval <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	elapsed := time.Since(c.lastCall)
	if elapsed < c.minInterval {
		time.Sleep(c.minInterval - elapsed)
	}
	c.lastCall = time.Now()
}

func (c *Client) currentURLIndex() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.urlIndex
}

func (c *Client) setURLIndex(i int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.urlIndex = i
}

func (c *Client) advanceURL() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.urlIndex = (c.urlIndex + 1) % len(c.urls)
}

func isNotFound(err *jsonRPCError) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Message + " " + err.Data)
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "could not find") ||
		strings.Contains(msg, "height must be less") ||
		err.Code == -32603 // often used for height errors
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
