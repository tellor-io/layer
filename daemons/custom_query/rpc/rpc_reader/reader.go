package rpc_reader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		timeout:      time.Duration(timeout) * time.Second,
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
		url := r.client.baseURL
		ctxWithTimeout, cancel := context.WithTimeout(ctx, r.timeout)
		req, err := http.NewRequestWithContext(ctxWithTimeout, "GET", url, nil)

		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to create request: %w", err)
			log.Warnf("Failed to create request (attempt %d/%d): %v", retry+1, r.maxRetries+1, err)
			if retry < r.maxRetries {
				time.Sleep(r.retryDelay * time.Duration(retry+1))
				continue
			}
			break
		}

		for key, value := range r.Headers {
			req.Header.Add(key, value)
		}

		resp, err := r.client.client.Do(req)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			log.Warnf("Request failed (attempt %d/%d): %v", retry+1, r.maxRetries+1, err)
			if retry < r.maxRetries {
				time.Sleep(r.retryDelay * time.Duration(retry+1))
				continue
			}
			break
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("received non-OK response code: %d", resp.StatusCode)
			log.Warnf("Non-OK response (attempt %d/%d): status=%d", retry+1, r.maxRetries+1, resp.StatusCode)
			if retry < r.maxRetries {
				time.Sleep(r.retryDelay * time.Duration(retry+1))
				continue
			}
			break
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("error reading response body: %w", err)
			log.Warnf("Failed to read response body (attempt %d/%d): %v", retry+1, r.maxRetries+1, err)
			if retry < r.maxRetries {
				time.Sleep(r.retryDelay * time.Duration(retry+1))
				continue
			}
			break
		}

		log.Debugf("RPC call successful: url=%s", url)
		metrics.RPCCallSuccess.Inc()
		return body, nil
	}

	metrics.RPCCallErrors.Inc()
	return nil, fmt.Errorf("all RPC endpoints failed: %w", lastErr)
}

func (r *Reader) ExtractValueFromJSON(data []byte, path []string) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	current := result
	for i, key := range path {
		if current == nil {
			return nil, fmt.Errorf("null value at path segment %d: %s", i, key)
		}

		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[key]
			if !ok {
				return nil, fmt.Errorf("key not found at path segment %d: %s", i, key)
			}
		case []interface{}:
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
