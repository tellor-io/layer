package rpc_reader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tellor-io/layer/daemons/custom_query/contracts/metrics"
)

type Reader struct {
	client       *httpClient
	timeout      time.Duration
	maxRetries   int
	retryDelay   time.Duration
	Headers      map[string]string
	ResponsePath []string
}

type httpClient struct {
	client  *http.Client
	baseURL string
}

func NewReader(url string, headers map[string]string, responsePath []string, timeout int) (*Reader, error) {
	if url == "" {
		return nil, fmt.Errorf("no RPC endpoint provided")
	}

	client := &httpClient{
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		baseURL: url,
	}

	reader := &Reader{
		client:       client,
		timeout:      time.Duration(timeout) * time.Millisecond,
		maxRetries:   3,
		retryDelay:   100 * time.Millisecond,
		Headers:      headers,
		ResponsePath: responsePath,
	}

	return reader, nil
}

func (r *Reader) FetchJSON(ctx context.Context) ([]byte, error) {
	startTime := time.Now()
	defer func() {
		metrics.RPCCallDuration.Observe(time.Since(startTime).Seconds())
	}()

	var lastErr error
	for retry := 0; retry <= r.maxRetries; retry++ {
		// Check if context is already canceled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Add retry delay (except for first attempt)
		if retry > 0 {
			delay := time.Duration(math.Pow(2, float64(retry-1))) * r.retryDelay
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		body, err := r.attemptFetch(ctx)
		if err == nil {
			log.Infof("RPC call successful: url=%s", r.client.baseURL)
			metrics.RPCCallSuccess.Inc()
			return body, nil
		}

		lastErr = err
		log.Warnf("Request failed (attempt %d/%d): %v", retry+1, r.maxRetries+1, err)
	}

	metrics.RPCCallErrors.Inc()
	return nil, fmt.Errorf("all RPC endpoints failed: %w", lastErr)
}

func (r *Reader) attemptFetch(ctx context.Context) ([]byte, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxWithTimeout, "GET", r.client.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range r.Headers {
		req.Header.Add(key, value)
	}

	resp, err := r.client.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK response code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Error reading response body: %v, url=%s", err, r.client.baseURL)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

func (r *Reader) ExtractValueFromJSON(data []byte, path []string) (interface{}, error) {
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	current := result
	for i, key := range path {
		if current == nil {
			return nil, fmt.Errorf("null value at path segment %d: %s", i, key)
		}

		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[key]
			if !ok {
				return nil, fmt.Errorf("key not found at path segment %d: %s", i, key)
			}
		case []any:
			var index int
			if _, err := fmt.Sscanf(key, "%d", &index); err != nil {
				return nil, fmt.Errorf("expected numeric index for array, got: %s", key)
			}
			if index < 0 || index >= len(v) {
				return nil, fmt.Errorf("array index out of bounds: %d", index)
			}
			current = v[index]
		default:
			return nil, fmt.Errorf("expected object or array at path segment %d: %s, got %T", i, key, current)
		}
	}

	return current, nil
}
