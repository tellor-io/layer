package customquery

import (
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

// Result holds the value returned from an endpoint
type Result struct {
	Value      float64
	Err        error
	EndpointID string
}

// FetchPriceResult holds the result of a price fetch operation
type FetchPriceResult struct {
	EncodedValue string
	RawResults   []Result
	QueryID      string
	ResponseType string
	SuccessRate  float64
}

// FetchPrice fetches price data for the given query ID
func FetchPrice(ctx context.Context, query QueryConfig) (*FetchPriceResult, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	results := make(chan Result, len(query.Endpoints))
	var wg sync.WaitGroup

	// Launch a goroutine for each endpoint
	for _, endpoint := range query.BuiltEndpoints {
		wg.Add(1)
		go func(ep BuiltEndpoint) {
			defer wg.Done()
			result := fetchFromEndpoint(ctx, ep)
			results <- result
		}(endpoint)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []Result
	// Count successful results
	var successfulResults []Result
	for result := range results {
		allResults = append(allResults, result)
		if result.Err == nil {
			successfulResults = append(successfulResults, result)
		}
	}
	// Check if we have enough successful responses
	if len(successfulResults) < query.MinResponses {
		return nil, fmt.Errorf("insufficient successful responses: got %d, need %d",
			len(successfulResults), query.MinResponses)
	}
	fmt.Println("Successful results:", successfulResults)
	// Aggregate results
	aggregatedValue, err := aggregateResults(successfulResults, query.AggregationMethod, query.ResponseType)
	if err != nil {
		return nil, err
	}

	return &FetchPriceResult{
		EncodedValue: aggregatedValue,
		RawResults:   allResults,
		QueryID:      query.ID,
		ResponseType: query.ResponseType,
		SuccessRate:  float64(len(successfulResults)) / float64(len(query.Endpoints)),
	}, nil
}

// fetchFromEndpoint fetches data from a single endpoint
func fetchFromEndpoint(ctx context.Context, endpoint BuiltEndpoint) Result {
	// Create a context with the endpoint's timeout
	timeoutDuration := time.Duration(endpoint.Timeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	// Create request
	req, err := http.NewRequestWithContext(ctx, endpoint.Method, endpoint.URL, nil)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("error creating request: %w", err),
			EndpointID: endpoint.EndpointID,
		}
	}
	if endpoint.Headers != nil {
		addHeaders(req, endpoint.Headers)
	}
	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("request failed: %w", err),
			EndpointID: endpoint.EndpointID,
		}
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return Result{
			Err:        fmt.Errorf("received non-OK response code: %d", resp.StatusCode),
			EndpointID: endpoint.EndpointID,
		}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("error reading response body: %w", err),
			EndpointID: endpoint.EndpointID,
		}
	}

	// Extract value from response according to response path
	value, err := extractValueFromJSON(body, endpoint.ResponsePath)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("error extracting value: %w", err),
			EndpointID: endpoint.EndpointID,
		}
	}

	return Result{
		Value:      value,
		EndpointID: endpoint.EndpointID,
	}
}

// extractValueFromJSON extracts a float64 value from JSON based on the given path
func extractValueFromJSON(data []byte, path []string) (float64, error) {
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	// Navigate through the JSON path
	current := result
	for i, key := range path {
		if current == nil {
			return 0, fmt.Errorf("null value at path segment %d: %s", i, key)
		}

		mapValue, ok := current.(map[string]interface{})
		if !ok {
			// if not a map, check if it's an array
			if currentArray, ok := current.([]interface{}); ok {
				// Try to parse key as an index
				index, err := strconv.Atoi(key)
				if err != nil {
					return 0, fmt.Errorf("expected numeric index for array, got: %s", key)
				}

				if index < 0 || index >= len(currentArray) {
					return 0, fmt.Errorf("array index out of bounds: %d", index)
				}

				current = currentArray[index]
				continue
			}
			return 0, fmt.Errorf("expected object at path segment %d: %s, got %T", i, key, current)
		}

		current, ok = mapValue[key]
		if !ok {
			return 0, fmt.Errorf("key not found at path segment %d: %s", i, key)
		}
	}

	// Convert the final value to float64
	switch v := current.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		var f float64
		_, err := fmt.Sscanf(v, "%f", &f)
		if err != nil {
			return 0, fmt.Errorf("error parsing string as float: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported value type: %T", current)
	}
}

// aggregateResults aggregates results using the specified method
func aggregateResults(results []Result, method, responseType string) (string, error) {
	if len(results) == 0 {
		return "", fmt.Errorf("no results to aggregate")
	}

	// Extract values
	values := make([]float64, len(results))
	for i, result := range results {
		values[i] = result.Value
	}

	switch strings.ToLower(method) {
	case "median":
		return MedianInHex(values, responseType)
	// case "mode":
	// return ModeInHex(values, responseType)
	default:
		return "", fmt.Errorf("unsupported aggregation method: %s", method)
	}
}

func ConvertFloat64ToString(num float64) string {
	return strconv.FormatFloat(num, 'f', -1, 64)
}

func addHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		req.Header.Add(key, value)
	}
}
