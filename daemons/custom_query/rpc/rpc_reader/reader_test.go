package rpc_reader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewReader(t *testing.T) {
	url := "https://api.example.com/v1/data"
	headers := map[string]string{"X-API-Key": "test"}
	responsePath := []string{"data", "value"}

	reader, err := NewReader(url, headers, responsePath, 5)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	if reader.client.baseURL != url {
		t.Errorf("Expected URL %s, got %s", url, reader.client.baseURL)
	}

	if reader.timeout != 5*time.Millisecond {
		t.Errorf("Expected timeout of 5s, got %v", reader.timeout)
	}

	if len(reader.Headers) != 1 {
		t.Errorf("Expected 1 header, got %d", len(reader.Headers))
	}

	if len(reader.ResponsePath) != 2 {
		t.Errorf("Expected response path length 2, got %d", len(reader.ResponsePath))
	}
}

func TestFetchJSON(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("X-Test-Header") != "test-value" {
			t.Errorf("Expected header X-Test-Header=test-value, got %s", r.Header.Get("X-Test-Header"))
		}

		response := map[string]interface{}{
			"pool": map[string]interface{}{
				"id":                 "1136",
				"current_sqrt_price": "1.234567890",
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	// Create reader with test server URL
	headers := map[string]string{"X-Test-Header": "test-value"}
	reader, err := NewReader(server.URL, headers, []string{"pool", "current_sqrt_price"}, 5)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	// Test fetch
	ctx := context.Background()
	data, err := reader.FetchJSON(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch JSON: %v", err)
	}

	// Verify response
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	pool, ok := result["pool"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected pool object in response")
	}

	if pool["id"] != "1136" {
		t.Errorf("Expected pool id 1136, got %v", pool["id"])
	}
}

func TestExtractValueFromJSON(t *testing.T) {
	reader, _ := NewReader("http://example.com", nil, nil, 5)

	testCases := []struct {
		name     string
		data     string
		path     []string
		expected interface{}
	}{
		{
			name:     "Simple path",
			data:     `{"price": 100.5}`,
			path:     []string{"price"},
			expected: 100.5,
		},
		{
			name:     "Nested path",
			data:     `{"pool": {"current_sqrt_price": "1.234"}}`,
			path:     []string{"pool", "current_sqrt_price"},
			expected: "1.234",
		},
		{
			name:     "Array access",
			data:     `{"prices": [10, 20, 30]}`,
			path:     []string{"prices", "1"},
			expected: float64(20),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := reader.ExtractValueFromJSON([]byte(tc.data), tc.path)
			if err != nil {
				t.Fatalf("Failed to extract value: %v", err)
			}

			if value != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, value)
			}
		})
	}
}

func TestRetryLogic(t *testing.T) {
	failCount := 0
	// Create test server that fails initially then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"status": "ok"}))
	}))
	defer server.Close()

	// Create reader
	reader, err := NewReader(server.URL, nil, nil, 5)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	// Should retry and eventually succeed
	ctx := context.Background()
	data, err := reader.FetchJSON(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch JSON after retries: %v", err)
	}

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))
	if result["status"] != "ok" {
		t.Error("Expected successful response after retries")
	}

	if failCount != 3 {
		t.Errorf("Expected 3 attempts (2 failures + 1 success), got %d", failCount)
	}
}

func TestTimeout(t *testing.T) {
	// Create test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"status": "ok"}))
	}))
	defer server.Close()

	// Create reader with 1 second timeout
	reader, err := NewReader(server.URL, nil, nil, 1)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	// Should timeout
	ctx := context.Background()
	_, err = reader.FetchJSON(ctx)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}
