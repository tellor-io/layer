package configs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/configs"
	"github.com/tellor-io/layer/daemons/constants"
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

func TestMergeMarketParamsConfig(t *testing.T) {
	pwd, _ := os.Getwd()

	tests := map[string]struct {
		// setup
		initialParams []types.MarketParam
		expectedAdded int
		expectError   bool
		errorContains string
	}{
		"merge adds missing market params": {
			initialParams: []types.MarketParam{
				{
					Id:                 exchange_common.BTCUSD_ID,
					Pair:               `"BTC-USD"`,
					Exponent:           -5,
					MinExchanges:       3,
					MinPriceChangePpm:  1000,
					ExchangeConfigJson: `{"exchanges":[]}`,
					QueryData:          `"test"`,
				},
			},
			expectedAdded: len(constants.StaticMarketParamsConfig) - 1, // All except BTC-USD
		},
		"merge preserves existing custom values": {
			initialParams: []types.MarketParam{
				{
					Id:                 exchange_common.BTCUSD_ID,
					Pair:               `"BTC-USD"`,
					Exponent:           -6,   // Custom value different from default
					MinExchanges:       5,    // Custom value different from default
					MinPriceChangePpm:  2000, // Custom value different from default
					ExchangeConfigJson: `{"exchanges":[]}`,
					QueryData:          `"test"`,
				},
			},
			expectedAdded: len(constants.StaticMarketParamsConfig) - 1,
		},
		"merge handles file with all entries already present": {
			initialParams: func() []types.MarketParam {
				allParams := make([]types.MarketParam, 0, len(constants.StaticMarketParamsConfig))
				for _, param := range constants.StaticMarketParamsConfig {
					allParams = append(allParams, *param)
				}
				return allParams
			}(),
			expectedAdded: 0, // No new params to add
		},
		"merge handles empty file": {
			initialParams: []types.MarketParam{},
			expectedAdded: len(constants.StaticMarketParamsConfig),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup: create config directory and write initial config
			err := os.Mkdir("config", 0o700)
			require.NoError(t, err)
			defer os.RemoveAll("config")

			// Write initial config file
			configPath := filepath.Join("config", constants.MarketParamsConfigFileName)
			initialConfig := map[string][]types.MarketParam{
				"market_params": tc.initialParams,
			}

			// Marshal to TOML
			tomlBytes, err := toml.Marshal(initialConfig)
			require.NoError(t, err)
			err = os.WriteFile(configPath, tomlBytes, 0o644)
			require.NoError(t, err)

			// Perform merge
			err = configs.MergeMarketParamsConfig(pwd)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)

			// Read merged config
			mergedParams := configs.ReadMarketParamsConfigFile(pwd)

			// Verify all static params are present
			require.Equal(t, len(constants.StaticMarketParamsConfig), len(mergedParams))

			// Verify initial params preserved their values
			mergedParamsMap := make(map[uint32]types.MarketParam)
			for _, param := range mergedParams {
				mergedParamsMap[param.Id] = param
			}

			for _, initialParam := range tc.initialParams {
				mergedParam, exists := mergedParamsMap[initialParam.Id]
				require.True(t, exists, "initial param %d should be present", initialParam.Id)
				require.Equal(t, initialParam.Exponent, mergedParam.Exponent, "custom Exponent should be preserved")
				require.Equal(t, initialParam.MinExchanges, mergedParam.MinExchanges, "custom MinExchanges should be preserved")
				require.Equal(t, initialParam.MinPriceChangePpm, mergedParam.MinPriceChangePpm, "custom MinPriceChangePpm should be preserved")
			}

			// Verify new params were added with default values
			existingIds := make(map[uint32]bool)
			for _, initialParam := range tc.initialParams {
				existingIds[initialParam.Id] = true
			}

			addedCount := 0
			for paramId, defaultParam := range constants.StaticMarketParamsConfig {
				if !existingIds[paramId] {
					mergedParam, exists := mergedParamsMap[paramId]
					require.True(t, exists, "missing param %d should be added", paramId)
					require.Equal(t, defaultParam.Exponent, mergedParam.Exponent, "new param should have default Exponent")
					require.Equal(t, defaultParam.MinExchanges, mergedParam.MinExchanges, "new param should have default MinExchanges")
					require.Equal(t, defaultParam.MinPriceChangePpm, mergedParam.MinPriceChangePpm, "new param should have default MinPriceChangePpm")
					addedCount++
				}
			}

			require.Equal(t, tc.expectedAdded, addedCount, "expected number of added params")
		})
	}
}
