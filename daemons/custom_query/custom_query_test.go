package customquery

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessApiKeys(t *testing.T) {
	os.Setenv("TEST_API_KEY", "abc123")
	os.Setenv("ANOTHER_API_KEY", "xyz789")
	defer os.Unsetenv("TEST_API_KEY")
	defer os.Unsetenv("ANOTHER_API_KEY")

	config := &Config{
		Endpoints: map[string]EndpointTemplate{
			"1": {
				ApiKey: "${TEST_API_KEY}",
			},
			"2": {
				ApiKey: "hardcoded-key",
			},
			"3": {
				ApiKey: "${ANOTHER_API_KEY}",
			},
			"4": {
				ApiKey: "${NONEXISTENT_KEY}",
			},
		},
	}

	processApiKeys(config)

	testCases := map[string]struct {
		endpoint    string
		expectedKey string
	}{
		"environment variable exists": {
			endpoint:    "1",
			expectedKey: "abc123",
		},
		"hardcoded key unchanged": {
			endpoint:    "2",
			expectedKey: "hardcoded-key",
		},
		"another environment variable": {
			endpoint:    "3",
			expectedKey: "xyz789",
		},
		"non-existent environment variable": {
			endpoint:    "4",
			expectedKey: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actual := config.Endpoints[tc.endpoint].ApiKey
			require.Equal(t, tc.expectedKey, actual)
		})
	}
}

func TestBuildQueryEndpoints(t *testing.T) {
	os.Setenv("ETHERSCAN_API_KEY", "testetherscankey123")
	defer os.Unsetenv("ETHERSCAN_API_KEY")
	pwd, _ := os.Getwd()

	queryMap, err := BuildQueryEndpoints(pwd, "testdata", "test_config.toml")
	require.NoError(t, err)
	require.Equal(t, len(queryMap), 2)

	sdaiQuery, exists := queryMap["sdai_test_id"]
	require.True(t, exists)
	require.Equal(t, sdaiQuery.AggregationMethod, "median")
	require.Equal(t, len(sdaiQuery.RpcReaders), 3)

	var coingeckoEndpoint RpcHandler
	for _, endpoint := range sdaiQuery.RpcReaders {
		if endpoint.EndpointID == "coingecko" {
			coingeckoEndpoint = endpoint
			break
		}
	}

	// expectedCoingeckoURL := "https://api.coingecko.com/api/v3/simple/price?ids=savings-dai&vs_currencies=usd"
	require.Equal(t, coingeckoEndpoint.Reader.ResponsePath, []string{"savings-dai", "usd"})

	trbQuery, exists := queryMap["trb_test_id"]
	require.True(t, exists)
	require.Equal(t, sdaiQuery.AggregationMethod, "median")
	require.Equal(t, len(trbQuery.RpcReaders), 3)

	var etherscanEndpoint RpcHandler
	for _, endpoint := range trbQuery.RpcReaders {
		if endpoint.EndpointID == "etherscan" {
			etherscanEndpoint = endpoint
			break
		}
	}
	require.NotEmpty(t, etherscanEndpoint)
	require.Contains(t, etherscanEndpoint.Reader.ResponsePath, "result")
}

func TestBuildQueryEndpointsErrors(t *testing.T) {
	pwd, _ := os.Getwd()
	testCases := []struct {
		name        string
		configFile  string
		expectError bool
	}{
		{
			name:        "missing endpoint template",
			configFile:  "missing_template.toml",
			expectError: true,
		},
		{
			name:        "missing required parameter",
			configFile:  "missing_param.toml",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildQueryEndpoints(pwd, "testdata", tc.configFile)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
