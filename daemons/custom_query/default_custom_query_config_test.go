package customquery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
	customquery "github.com/tellor-io/layer/daemons/custom_query"
)

func TestMergeCustomQueryConfig(t *testing.T) {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, "test_config")
	localDir := "config"
	fileName := "custom_query_config.toml"

	tests := map[string]struct {
		// setup
		initialConfig customquery.Config
		expectedAdded struct {
			endpoints    int
			rpcEndpoints int
			queries      int
		}
		expectError   bool
		errorContains string
	}{
		"merge adds missing entries": {
			initialConfig: customquery.Config{
				Endpoints: map[string]customquery.EndpointTemplate{
					"coingecko": {
						URLTemplate: "https://api.coingecko.com/api/v3/simple/price?ids={coin_id}&vs_currencies=usd",
						Method:      "GET",
						Timeout:     5000,
					},
				},
				RPCEndpoints: map[string]customquery.RPCEndpointTemplate{},
				Queries:      map[string]customquery.QueryConfig{},
			},
			expectedAdded: struct {
				endpoints    int
				rpcEndpoints int
				queries      int
			}{
				endpoints:    len(customquery.StaticEndpointTemplateConfig) - 1, // All except coingecko
				rpcEndpoints: len(customquery.StaticRPCEndpointTemplateConfig),
				queries:      len(customquery.StaticQueriesConfig),
			},
		},
		"merge preserves existing custom values": {
			initialConfig: customquery.Config{
				Endpoints: map[string]customquery.EndpointTemplate{
					"coingecko": {
						URLTemplate: "https://custom.coingecko.com/api", // Custom value
						Method:      "POST",                             // Custom value
						Timeout:     10000,                              // Custom value
					},
				},
				RPCEndpoints: map[string]customquery.RPCEndpointTemplate{
					"ethereum": {
						URLs: []string{"https://custom.rpc.url"}, // Custom value
					},
				},
				Queries: map[string]customquery.QueryConfig{
					"05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6": {
						ID:                "05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6",
						AggregationMethod: "mean", // Custom value
						MaxSpreadPercent:  25.0,   // Custom value
						MinResponses:      5,      // Custom value
						ResponseType:      "ufixed256x18",
						Endpoints:         []customquery.EndpointConfig{},
					},
				},
			},
			expectedAdded: struct {
				endpoints    int
				rpcEndpoints int
				queries      int
			}{
				endpoints:    len(customquery.StaticEndpointTemplateConfig) - 1,
				rpcEndpoints: len(customquery.StaticRPCEndpointTemplateConfig) - 1,
				queries:      len(customquery.StaticQueriesConfig) - 1,
			},
		},
		"merge handles file with all entries already present": {
			initialConfig: func() customquery.Config {
				config := customquery.Config{
					Endpoints:    make(map[string]customquery.EndpointTemplate),
					RPCEndpoints: make(map[string]customquery.RPCEndpointTemplate),
					Queries:      make(map[string]customquery.QueryConfig),
				}
				for k, v := range customquery.StaticEndpointTemplateConfig {
					config.Endpoints[k] = *v
				}
				for k, v := range customquery.StaticRPCEndpointTemplateConfig {
					config.RPCEndpoints[k] = *v
				}
				for k, v := range customquery.StaticQueriesConfig {
					config.Queries[k] = *v
				}
				return config
			}(),
			expectedAdded: struct {
				endpoints    int
				rpcEndpoints int
				queries      int
			}{
				endpoints:    0,
				rpcEndpoints: 0,
				queries:      0,
			},
		},
		"merge handles empty file": {
			initialConfig: customquery.Config{
				Endpoints:    map[string]customquery.EndpointTemplate{},
				RPCEndpoints: map[string]customquery.RPCEndpointTemplate{},
				Queries:      map[string]customquery.QueryConfig{},
			},
			expectedAdded: struct {
				endpoints    int
				rpcEndpoints int
				queries      int
			}{
				endpoints:    len(customquery.StaticEndpointTemplateConfig),
				rpcEndpoints: len(customquery.StaticRPCEndpointTemplateConfig),
				queries:      len(customquery.StaticQueriesConfig),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup: create config directory
			err := os.MkdirAll(filepath.Join(testDir, localDir), 0o700)
			require.NoError(t, err)
			defer os.RemoveAll(testDir)

			// Write initial config file
			configPath := filepath.Join(testDir, localDir, fileName)
			tomlBytes, err := toml.Marshal(tc.initialConfig)
			require.NoError(t, err)
			err = os.WriteFile(configPath, tomlBytes, 0o644)
			require.NoError(t, err)

			// Perform merge
			err = customquery.MergeCustomQueryConfig(testDir, localDir, fileName)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)

			// Read merged config
			mergedConfig, err := customquery.BuildQueryEndpoints(testDir, localDir, fileName)
			require.NoError(t, err)

			// Read raw config to check endpoints and rpc_endpoints
			rawConfig := customquery.Config{}
			rawToml, err := os.ReadFile(configPath)
			require.NoError(t, err)
			err = toml.Unmarshal(rawToml, &rawConfig)
			require.NoError(t, err)

			// Verify all static endpoints are present
			require.Equal(t, len(customquery.StaticEndpointTemplateConfig), len(rawConfig.Endpoints))

			// Verify all static RPC endpoints are present
			require.Equal(t, len(customquery.StaticRPCEndpointTemplateConfig), len(rawConfig.RPCEndpoints))

			// Verify all static queries are present
			require.Equal(t, len(customquery.StaticQueriesConfig), len(rawConfig.Queries))
			require.Equal(t, len(customquery.StaticQueriesConfig), len(mergedConfig))

			// Verify initial endpoints preserved their values
			for key, initialEndpoint := range tc.initialConfig.Endpoints {
				mergedEndpoint, exists := rawConfig.Endpoints[key]
				require.True(t, exists, "initial endpoint %s should be present", key)
				require.Equal(t, initialEndpoint.URLTemplate, mergedEndpoint.URLTemplate, "custom URLTemplate should be preserved")
				require.Equal(t, initialEndpoint.Method, mergedEndpoint.Method, "custom Method should be preserved")
				require.Equal(t, initialEndpoint.Timeout, mergedEndpoint.Timeout, "custom Timeout should be preserved")
			}

			// Verify initial RPC endpoints preserved their values
			for key, initialRPCEndpoint := range tc.initialConfig.RPCEndpoints {
				mergedRPCEndpoint, exists := rawConfig.RPCEndpoints[key]
				require.True(t, exists, "initial RPC endpoint %s should be present", key)
				require.Equal(t, initialRPCEndpoint.URLs, mergedRPCEndpoint.URLs, "custom URLs should be preserved")
			}

			// Verify initial queries preserved their values
			for key, initialQuery := range tc.initialConfig.Queries {
				mergedQuery, exists := rawConfig.Queries[key]
				require.True(t, exists, "initial query %s should be present", key)
				require.Equal(t, initialQuery.AggregationMethod, mergedQuery.AggregationMethod, "custom AggregationMethod should be preserved")
				require.Equal(t, initialQuery.MaxSpreadPercent, mergedQuery.MaxSpreadPercent, "custom MaxSpreadPercent should be preserved")
				require.Equal(t, initialQuery.MinResponses, mergedQuery.MinResponses, "custom MinResponses should be preserved")
			}

			// Count added entries
			existingEndpoints := make(map[string]bool)
			for key := range tc.initialConfig.Endpoints {
				existingEndpoints[key] = true
			}

			existingRPCEndpoints := make(map[string]bool)
			for key := range tc.initialConfig.RPCEndpoints {
				existingRPCEndpoints[key] = true
			}

			existingQueries := make(map[string]bool)
			for key := range tc.initialConfig.Queries {
				existingQueries[key] = true
			}

			addedEndpoints := 0
			for key := range customquery.StaticEndpointTemplateConfig {
				if !existingEndpoints[key] {
					_, exists := rawConfig.Endpoints[key]
					require.True(t, exists, "missing endpoint %s should be added", key)
					addedEndpoints++
				}
			}

			addedRPCEndpoints := 0
			for key := range customquery.StaticRPCEndpointTemplateConfig {
				if !existingRPCEndpoints[key] {
					_, exists := rawConfig.RPCEndpoints[key]
					require.True(t, exists, "missing RPC endpoint %s should be added", key)
					addedRPCEndpoints++
				}
			}

			addedQueries := 0
			for key := range customquery.StaticQueriesConfig {
				if !existingQueries[key] {
					_, exists := rawConfig.Queries[key]
					require.True(t, exists, "missing query %s should be added", key)
					addedQueries++
				}
			}

			require.Equal(t, tc.expectedAdded.endpoints, addedEndpoints, "expected number of added endpoints")
			require.Equal(t, tc.expectedAdded.rpcEndpoints, addedRPCEndpoints, "expected number of added RPC endpoints")
			require.Equal(t, tc.expectedAdded.queries, addedQueries, "expected number of added queries")
		})
	}
}
