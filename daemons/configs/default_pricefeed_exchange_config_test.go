package configs_test

import (
	"errors"
	"fmt"
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

var (
	binanceId = exchange_common.EXCHANGE_ID_BINANCE
	filePath  = fmt.Sprintf("config/%v", constants.PricefeedExchangeConfigFileName)
)

const (
	tomlString = `# This is a TOML config file.
	# StaticExchangeStartupConfig represents the mapping of exchanges to the parameters for
	# querying from them.
	#
	# ExchangeId - Unique string identifying an exchange.
	#
	# IntervalMs - Delays between sending API requests to get exchange market prices - cannot be 0.
	#
	# TimeoutMs - Max time to wait on an API call to an exchange - cannot be 0.
	#
	# MaxQueries - Max api calls to get market prices for an exchange to make in a task-loop -
	# cannot be 0. For multi-market API exchanges, the behavior will default to 1.
	[[exchanges]]
	ExchangeId = "Binance"
	IntervalMs = 2500
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "BinanceUS"
	IntervalMs = 2500
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Bitfinex"
	IntervalMs = 2500
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Bitstamp"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "CoinbaseRates"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "CryptoCom"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Gate"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Huobi"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Kraken"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Kucoin"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Mexc"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "Okx"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 1
	[[exchanges]]
	ExchangeId = "TestFixedPriceExchange"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 3
	[[exchanges]]
	ExchangeId = "TestVolatileExchange"
	IntervalMs = 2000
	TimeoutMs = 3000
	MaxQueries = 3
`
)

func TestGenerateDefaultExchangeTomlString(t *testing.T) {
	defaultConfigStringBuffer := configs.GenerateDefaultExchangeTomlString()
	require.Equal(
		t,
		tomlString,
		defaultConfigStringBuffer.String(),
	)
}

func TestWriteDefaultPricefeedExchangeToml(t *testing.T) {
	err := os.Mkdir("config", 0o700)
	require.NoError(t, err)
	configs.WriteDefaultPricefeedExchangeToml("")

	buffer, err := os.ReadFile(filePath)
	require.NoError(t, err)

	require.Equal(t, tomlString, string(buffer))
	os.RemoveAll("config")
}

// func TestWriteDefaultPricefeedExchangeToml_FileExists(t *testing.T) {
// 	helloWorld := "Hello World"

// 	err := os.Mkdir("config", 0o700)
// 	require.NoError(t, err)

// 	tmos.MustWriteFile(filePath, bytes.NewBuffer([]byte(helloWorld)).Bytes(), 0o644)
// 	configs.WriteDefaultPricefeedExchangeToml("")

// 	buffer, err := os.ReadFile(filePath)
// 	require.NoError(t, err)

// 	require.Equal(t, helloWorld, string(buffer))
// 	os.RemoveAll("config")
// }

func TestReadExchangeStartupConfigFile(t *testing.T) {
	pwd, _ := os.Getwd()

	tests := map[string]struct {
		// parameters
		exchangeConfigSourcePath string
		doNotWriteFile           bool

		// expectations
		expectedExchangeId         types.ExchangeId
		expectedIntervalMsExchange uint32
		expectedTimeoutMsExchange  uint32
		expectedMaxQueries         uint32
		expectedPanic              error
	}{
		"valid": {
			exchangeConfigSourcePath:   "test_data/valid_test.toml",
			expectedExchangeId:         binanceId,
			expectedIntervalMsExchange: constants.StaticExchangeQueryConfig[binanceId].IntervalMs,
			expectedTimeoutMsExchange:  constants.StaticExchangeQueryConfig[binanceId].TimeoutMs,
			expectedMaxQueries:         constants.StaticExchangeQueryConfig[binanceId].MaxQueries,
		},
		"config file cannot be found": {
			exchangeConfigSourcePath: "test_data/notexisting_test.toml",
			doNotWriteFile:           true,
			expectedPanic: fmt.Errorf(
				"open %s%s: no such file or directory",
				pwd+"/config/",
				constants.PricefeedExchangeConfigFileName,
			),
		},
		"config file cannot be unmarshalled": {
			exchangeConfigSourcePath: "test_data/broken_test.toml",
			expectedPanic:            errors.New("(1, 12): was expecting token [[, but got unclosed table array key instead"),
		},
		"config file has malformed values": {
			exchangeConfigSourcePath: "test_data/missingvals_test.toml",
			expectedPanic: errors.New(
				"one or more query config values are unset or are set to zero for exchange with id: 'BinanceUS'",
			),
		},
		"config file has incorrect values": {
			exchangeConfigSourcePath: "test_data/wrongvaltype_test.toml",
			expectedPanic: errors.New(
				"(3, 1): Can't convert a(string) to uint32",
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if !tc.doNotWriteFile {
				err := os.Mkdir("config", 0o700)
				require.NoError(t, err)

				file, err := os.Open(tc.exchangeConfigSourcePath)
				require.NoError(t, err)

				config, err := os.Create(filepath.Join("config", constants.PricefeedExchangeConfigFileName))
				require.NoError(t, err)
				_, err = config.ReadFrom(file)
				require.NoError(t, err)
			}

			if tc.expectedPanic != nil {
				require.PanicsWithError(
					t,
					tc.expectedPanic.Error(),
					func() { configs.ReadExchangeQueryConfigFile(pwd) },
				)

				os.RemoveAll("config")
				return
			}

			exchangeStartupConfigMap := configs.ReadExchangeQueryConfigFile(pwd)

			require.Equal(
				t,
				&types.ExchangeQueryConfig{
					ExchangeId: tc.expectedExchangeId,
					IntervalMs: tc.expectedIntervalMsExchange,
					TimeoutMs:  tc.expectedTimeoutMsExchange,
					MaxQueries: tc.expectedMaxQueries,
				},
				exchangeStartupConfigMap[tc.expectedExchangeId],
			)

			os.RemoveAll("config")
		})
	}

	// In case tests fail and the path was never removed.
	os.RemoveAll("config")
}

func TestMergePricefeedExchangeConfig(t *testing.T) {
	pwd, _ := os.Getwd()

	tests := map[string]struct {
		// setup
		initialExchanges []types.ExchangeQueryConfig
		expectedAdded     int
		expectError       bool
		errorContains     string
	}{
		"merge adds missing exchanges": {
			initialExchanges: []types.ExchangeQueryConfig{
				{
					ExchangeId: exchange_common.EXCHANGE_ID_BINANCE,
					IntervalMs: 2500,
					TimeoutMs:  3000,
					MaxQueries: 1,
				},
			},
			expectedAdded: len(constants.StaticExchangeQueryConfig) - 1, // All except Binance
		},
		"merge preserves existing custom values": {
			initialExchanges: []types.ExchangeQueryConfig{
				{
					ExchangeId: exchange_common.EXCHANGE_ID_BINANCE,
					IntervalMs: 5000, // Custom value different from default
					TimeoutMs:  6000, // Custom value different from default
					MaxQueries: 5,    // Custom value different from default
				},
			},
			expectedAdded: len(constants.StaticExchangeQueryConfig) - 1,
		},
		"merge handles file with all entries already present": {
			initialExchanges: func() []types.ExchangeQueryConfig {
				allExchanges := make([]types.ExchangeQueryConfig, 0, len(constants.StaticExchangeQueryConfig))
				for _, config := range constants.StaticExchangeQueryConfig {
					allExchanges = append(allExchanges, *config)
				}
				return allExchanges
			}(),
			expectedAdded: 0, // No new exchanges to add
		},
		"merge handles empty file": {
			initialExchanges: []types.ExchangeQueryConfig{},
			expectedAdded:     len(constants.StaticExchangeQueryConfig),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup: create config directory and write initial config
			err := os.Mkdir("config", 0o700)
			require.NoError(t, err)
			defer os.RemoveAll("config")

			// Write initial config file
			configPath := filepath.Join("config", constants.PricefeedExchangeConfigFileName)
			initialConfig := map[string][]types.ExchangeQueryConfig{
				"exchanges": tc.initialExchanges,
			}

			// Marshal to TOML
			tomlBytes, err := toml.Marshal(initialConfig)
			require.NoError(t, err)
			err = os.WriteFile(configPath, tomlBytes, 0o644)
			require.NoError(t, err)

			// Perform merge
			err = configs.MergePricefeedExchangeConfig(pwd)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)

			// Read merged config
			mergedConfig := configs.ReadExchangeQueryConfigFile(pwd)

			// Verify all static exchanges are present
			require.Equal(t, len(constants.StaticExchangeQueryConfig), len(mergedConfig))

			// Verify initial exchanges preserved their values
			for _, initialExchange := range tc.initialExchanges {
				mergedExchange, exists := mergedConfig[initialExchange.ExchangeId]
				require.True(t, exists, "initial exchange %s should be present", initialExchange.ExchangeId)
				require.Equal(t, initialExchange.IntervalMs, mergedExchange.IntervalMs, "custom IntervalMs should be preserved")
				require.Equal(t, initialExchange.TimeoutMs, mergedExchange.TimeoutMs, "custom TimeoutMs should be preserved")
				require.Equal(t, initialExchange.MaxQueries, mergedExchange.MaxQueries, "custom MaxQueries should be preserved")
			}

			// Verify new exchanges were added with default values
			existingIds := make(map[types.ExchangeId]bool)
			for _, initialExchange := range tc.initialExchanges {
				existingIds[initialExchange.ExchangeId] = true
			}

			addedCount := 0
			for exchangeId, defaultConfig := range constants.StaticExchangeQueryConfig {
				if !existingIds[exchangeId] {
					mergedExchange, exists := mergedConfig[exchangeId]
					require.True(t, exists, "missing exchange %s should be added", exchangeId)
					require.Equal(t, defaultConfig.IntervalMs, mergedExchange.IntervalMs, "new exchange should have default IntervalMs")
					require.Equal(t, defaultConfig.TimeoutMs, mergedExchange.TimeoutMs, "new exchange should have default TimeoutMs")
					require.Equal(t, defaultConfig.MaxQueries, mergedExchange.MaxQueries, "new exchange should have default MaxQueries")
					addedCount++
				}
			}

			require.Equal(t, tc.expectedAdded, addedCount, "expected number of added exchanges")
		})
	}
}
