package configs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/pelletier/go-toml"
	"github.com/tellor-io/layer/daemons/constants"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

// Note: any changes to the comments/variables/mapstructure must be reflected in the appropriate
// struct in daemons/pricefeed/client/static_exchange_startup_config.go.
const (
	defaultTomlTemplate = `# This is a TOML config file.
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
	# cannot be 0. For multi-market API exchanges, the behavior will default to 1.{{ range $exchangeId, $element := .}}
	[[exchanges]]
	ExchangeId = "{{$element.ExchangeId}}"
	IntervalMs = {{$element.IntervalMs}}
	TimeoutMs = {{$element.TimeoutMs}}
	MaxQueries = {{$element.MaxQueries}}{{end}}
`
)

// GenerateDefaultExchangeTomlString creates the toml file string containing the default configs
// for querying each exchange.
func GenerateDefaultExchangeTomlString() bytes.Buffer {
	// Create the template for turning each `parsableExchangeStartupConfig` into a toml map config in
	// a stringified toml file.
	template, err := template.New("").Parse(defaultTomlTemplate)
	// Panic if failure occurs when parsing the template.
	if err != nil {
		panic(err)
	}

	// Encode toml string into `defaultExchangeToml` and return if successful. Otherwise, panic.
	var defaultExchangeToml bytes.Buffer
	err = template.Execute(&defaultExchangeToml, constants.StaticExchangeQueryConfig)
	if err != nil {
		panic(err)
	}
	return defaultExchangeToml
}

// MergePricefeedExchangeConfig merges missing exchanges from static config into existing config file.
// It preserves existing exchanges and only adds new ones with default values.
func MergePricefeedExchangeConfig(homeDir string) error {
	configFilePath := getConfigFilePath(homeDir)
	
	// Read existing config file
	tomlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read existing config file: %w", err)
	}

	// Unmarshal existing exchanges
	existingExchanges := map[string][]types.ExchangeQueryConfig{}
	if err = toml.Unmarshal(tomlFile, &existingExchanges); err != nil {
		return fmt.Errorf("failed to unmarshal existing config: %w", err)
	}

	// Create a map of existing exchange IDs for quick lookup
	existingExchangeMap := make(map[types.ExchangeId]bool)
	for _, exchange := range existingExchanges["exchanges"] {
		existingExchangeMap[exchange.ExchangeId] = true
	}

	// Find missing exchanges from static config
	missingExchanges := make([]types.ExchangeQueryConfig, 0)
	for exchangeId, defaultConfig := range constants.StaticExchangeQueryConfig {
		if !existingExchangeMap[exchangeId] {
			missingExchanges = append(missingExchanges, *defaultConfig)
		}
	}

	// If no missing exchanges, nothing to do
	if len(missingExchanges) == 0 {
		return nil
	}

	// Append missing exchanges to existing ones
	allExchanges := append(existingExchanges["exchanges"], missingExchanges...)

	// Create merged config map for template
	mergedConfigMap := make(map[types.ExchangeId]*types.ExchangeQueryConfig)
	for _, exchange := range allExchanges {
		mergedConfigMap[exchange.ExchangeId] = &types.ExchangeQueryConfig{
			ExchangeId: exchange.ExchangeId,
			IntervalMs: exchange.IntervalMs,
			TimeoutMs:  exchange.TimeoutMs,
			MaxQueries: exchange.MaxQueries,
		}
	}

	// Generate merged TOML using template
	template, err := template.New("").Parse(defaultTomlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var mergedToml bytes.Buffer
	if err = template.Execute(&mergedToml, mergedConfigMap); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Validate merged config by reading it back
	// We'll use a temporary approach: write to temp file, validate, then replace
	tempFile := configFilePath + ".tmp"
	if err = tmos.WriteFile(tempFile, mergedToml.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Validate by attempting to read it
	testConfig := map[string][]types.ExchangeQueryConfig{}
	testToml, err := os.ReadFile(tempFile)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to read temp file for validation: %w", err)
	}
	if err = toml.Unmarshal(testToml, &testConfig); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("merged config validation failed: %w", err)
	}

	// Validate each exchange has required fields
	for _, exchange := range testConfig["exchanges"] {
		if exchange.IntervalMs == 0 || exchange.TimeoutMs == 0 || exchange.MaxQueries == 0 {
			os.Remove(tempFile)
			return fmt.Errorf("merged config has invalid exchange: %v", exchange.ExchangeId)
		}
	}

	// Replace original file with validated merged config
	if err = os.Rename(tempFile, configFilePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
}

// WriteDefaultPricefeedExchangeToml reads in the toml string for the pricefeed client and
// writes said string to the config folder as a toml file if the config file does not exist.
// If the file exists, it merges missing exchanges from static config.
func WriteDefaultPricefeedExchangeToml(homeDir string) {
	configFilePath := getConfigFilePath(homeDir)
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		buffer := GenerateDefaultExchangeTomlString()
		tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0o644)
	} else {
		// File exists, merge missing exchanges
		if err := MergePricefeedExchangeConfig(homeDir); err != nil {
			panic(fmt.Sprintf("failed to merge pricefeed exchange config: %v", err))
		}
	}
}

// ReadExchangeQueryConfigFile gets a mapping of `exchangeIds` to `ExchangeQueryConfigs`
// where `ExchangeQueryConfig` for querying exchanges for market prices comes from parsing a TOML
// file in the config directory.
// NOTE: if the config file is not found for the price-daemon, return the static exchange query
// config.
func ReadExchangeQueryConfigFile(homeDir string) map[types.ExchangeId]*types.ExchangeQueryConfig {
	// Read file for exchange query configurations.
	tomlFile, err := os.ReadFile(getConfigFilePath(homeDir))
	if err != nil {
		panic(err)
	}

	// Unmarshal `tomlFile` into `exchanges` for `exchangeStartupConfigMap`.
	exchanges := map[string][]types.ExchangeQueryConfig{}
	if err = toml.Unmarshal(tomlFile, &exchanges); err != nil {
		panic(err)
	}

	// Populate configs for exchanges.
	exchangeStartupConfigMap := make(map[types.ExchangeId]*types.ExchangeQueryConfig, len(exchanges))
	for _, exchange := range exchanges["exchanges"] {
		// Zero is an invalid configuration value for all parameters. This could also point to the
		// configuration file being setup wrong with one or more exchange parameters unset.
		if exchange.IntervalMs == 0 ||
			exchange.TimeoutMs == 0 ||
			exchange.MaxQueries == 0 {
			panic(
				fmt.Errorf(
					"one or more query config values are unset or are set to zero for exchange with id: '%v'",
					exchange.ExchangeId,
				),
			)
		}

		// Insert Key-Value pair into `exchangeStartupConfigMap`.
		exchangeStartupConfigMap[exchange.ExchangeId] = &types.ExchangeQueryConfig{
			ExchangeId: exchange.ExchangeId,
			IntervalMs: exchange.IntervalMs,
			TimeoutMs:  exchange.TimeoutMs,
			MaxQueries: exchange.MaxQueries,
		}
	}

	return exchangeStartupConfigMap
}

// getConfigFilePath returns the path to the pricefeed exchange config file.
func getConfigFilePath(homeDir string) string {
	return filepath.Join(
		homeDir,
		"config",
		constants.PricefeedExchangeConfigFileName,
	)
}
