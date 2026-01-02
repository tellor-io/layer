package configs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/pelletier/go-toml"
	"github.com/tellor-io/layer/daemons/constants"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

// Note: any changes to the comments/variables/mapstructure must be reflected in the appropriate
// struct in daemons/pricefeed/client/static_market_param_startup_config.go.
const (
	defaultMarketParamTomlTemplate = `# This is a TOML config file.
	# StaticMarketParamStartupConfig represents the mapping of exchanges to the parameters for
	# querying from them.
	#
	# market_params - Unique string identifying an exchange.
	#
	# Id - Delays between sending API requests to get exchange market prices - cannot be 0.
	#
	# MinExchanges - The minimum number of exchanges that should be reporting a live price for
	# a price update to be considered valid.
	#
	# Exponent - The exponent of the price.
	#
	# Pair - The human-readable name of the market pair (e.g. "BTC-USD").
	#
	# QueryData - Layer representation of the market pair.
{{ range $exchangeId, $element := .}}
	[[market_params]]
	ExchangeConfigJson = "{{escapeTomlString $element.ExchangeConfigJson}}"
	QueryData = "{{escapeTomlString $element.QueryData}}"
	Exponent = {{$element.Exponent}}
	Id = {{$element.Id}}
	MinExchanges = {{$element.MinExchanges}}
	MinPriceChangePpm = {{$element.MinPriceChangePpm}}
	Pair = "{{escapeTomlString $element.Pair}}"
{{end}}
`
)

// escapeTomlString escapes a string for use in TOML by replacing backslashes and quotes
func escapeTomlString(s string) string {
	// Replace backslashes first, then quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// GenerateDefaultExchangeTomlString creates the toml file string containing the default marketParam configs.
func GenerateDefaultMarketParamsTomlString() bytes.Buffer {
	template, err := template.New("").Funcs(template.FuncMap{
		"escapeTomlString": escapeTomlString,
	}).Parse(defaultMarketParamTomlTemplate)
	// Panic if failure occurs when parsing the template.
	if err != nil {
		panic(err)
	}

	// Encode toml string into `defaultMarketParamsToml` and return if successful. Otherwise, panic.
	var defaultMarketParamsToml bytes.Buffer
	err = template.Execute(&defaultMarketParamsToml, constants.StaticMarketParamsConfig)
	if err != nil {
		panic(err)
	}
	return defaultMarketParamsToml
}

// MergeMarketParamsConfig merges missing market params from static config into existing config file.
// It preserves existing market params and only adds new ones with default values.
func MergeMarketParamsConfig(homeDir string) error {
	configFilePath := getMarketParamsConfigFilePath(homeDir)

	// Read existing config file
	tomlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read existing config file: %w", err)
	}

	// Unmarshal existing market params
	existingParams := map[string][]types.MarketParam{}
	if err = toml.Unmarshal(tomlFile, &existingParams); err != nil {
		return fmt.Errorf("failed to unmarshal existing config: %w", err)
	}

	// Unescape ExchangeConfigJson strings (TOML parser may preserve escaped quotes)
	// Use strconv.Unquote to properly unescape the JSON string
	for i := range existingParams["market_params"] {
		jsonStr := existingParams["market_params"][i].ExchangeConfigJson
		// If the string looks like it has escaped quotes, try to unquote it
		if strings.Contains(jsonStr, "\\\"") {
			// Wrap in quotes for Unquote to work
			if unquoted, err := strconv.Unquote(`"` + jsonStr + `"`); err == nil {
				existingParams["market_params"][i].ExchangeConfigJson = unquoted
			}
		}
		// Strip quotes from QueryData if present (constants store it with quotes)
		if len(existingParams["market_params"][i].QueryData) > 0 && existingParams["market_params"][i].QueryData[0] == '"' {
			existingParams["market_params"][i].QueryData = strings.Trim(existingParams["market_params"][i].QueryData, "\"")
		}
	}

	// Create a map of existing market param IDs for quick lookup
	existingParamMap := make(map[uint32]bool)
	for _, param := range existingParams["market_params"] {
		existingParamMap[param.Id] = true
	}

	// Find missing market params from static config
	missingParams := make([]types.MarketParam, 0)
	for paramId, defaultParam := range constants.StaticMarketParamsConfig {
		if !existingParamMap[paramId] {
			missingParams = append(missingParams, *defaultParam)
		}
	}

	// If no missing params, nothing to do
	if len(missingParams) == 0 {
		return nil
	}

	// Append missing params to existing ones
	allParams := append(existingParams["market_params"], missingParams...)

	// Create merged config map for template
	mergedConfigMap := make(map[uint32]*types.MarketParam)
	for _, param := range allParams {
		mergedConfigMap[param.Id] = &types.MarketParam{
			ExchangeConfigJson: param.ExchangeConfigJson,
			Exponent:           param.Exponent,
			Id:                 param.Id,
			MinExchanges:       param.MinExchanges,
			MinPriceChangePpm:  param.MinPriceChangePpm,
			Pair:               param.Pair,
			QueryData:          param.QueryData,
		}
	}

	// Generate merged TOML using template
	template, err := template.New("").Funcs(template.FuncMap{
		"escapeTomlString": escapeTomlString,
	}).Parse(defaultMarketParamTomlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var mergedToml bytes.Buffer
	if err = template.Execute(&mergedToml, mergedConfigMap); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Validate merged config by reading it back
	tempFile := configFilePath + ".tmp"
	if err = tmos.WriteFile(tempFile, mergedToml.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Validate by attempting to read it
	testConfig := map[string][]types.MarketParam{}
	testToml, err := os.ReadFile(tempFile)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to read temp file for validation: %w", err)
	}
	if err = toml.Unmarshal(testToml, &testConfig); err != nil {
		// Include a snippet of the generated TOML for debugging
		lines := bytes.Split(mergedToml.Bytes(), []byte("\n"))
		var context bytes.Buffer
		if len(lines) > 0 {
			context.WriteString("First 20 lines of generated TOML:\n")
			maxLines := 20
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				context.WriteString(fmt.Sprintf("%d: %s\n", i+1, string(lines[i])))
			}
		}
		os.Remove(tempFile)
		return fmt.Errorf("merged config validation failed: %w\n%s", err, context.String())
	}

	// Validate each param has required fields
	// Note: Exponent is int32 and can be negative (e.g., -5, -6). We only reject 0 (unset value).
	for _, param := range testConfig["market_params"] {
		if param.Exponent == 0 || param.MinExchanges == 0 || param.MinPriceChangePpm == 0 || param.QueryData == "" {
			os.Remove(tempFile)
			return fmt.Errorf("merged config has invalid market param: %v", param.Id)
		}
	}

	// Replace original file with validated merged config
	if err = os.Rename(tempFile, configFilePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
}

func WriteDefaultMarketParamsToml(homeDir string) {
	// Write file into config folder if file does not exist.
	// If the file exists, merge missing market params from static config.
	configFilePath := getMarketParamsConfigFilePath(homeDir)
	configDir := filepath.Dir(configFilePath)
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		// Check if directory actually exists (might have been created by another process)
		if _, statErr := os.Stat(configDir); statErr != nil {
			panic(fmt.Sprintf("failed to create config directory %s: %v", configDir, err))
		}
	}
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		buffer := GenerateDefaultMarketParamsTomlString()
		tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0o644)
	} else {
		// File exists, merge missing market params
		if err := MergeMarketParamsConfig(homeDir); err != nil {
			panic(fmt.Sprintf("failed to merge market params config: %v", err))
		}
	}
}

func ReadMarketParamsConfigFile(homeDir string) []types.MarketParam {
	// Read file for exchange query configurations.
	tomlFile, err := os.ReadFile(getMarketParamsConfigFilePath(homeDir))
	if err != nil {
		panic(err)
	}

	params := map[string][]types.MarketParam{}
	if err = toml.Unmarshal(tomlFile, &params); err != nil {
		fmt.Println("Error unmarshalling toml file", err.Error())
		panic(err)
	}

	// Unescape ExchangeConfigJson strings (TOML parser may preserve escaped quotes)
	// Use strconv.Unquote to properly unescape the JSON string
	for i := range params["market_params"] {
		jsonStr := params["market_params"][i].ExchangeConfigJson
		// If the string looks like it has escaped quotes, try to unquote it
		if strings.Contains(jsonStr, "\\\"") {
			// Wrap in quotes for Unquote to work
			if unquoted, err := strconv.Unquote(`"` + jsonStr + `"`); err == nil {
				params["market_params"][i].ExchangeConfigJson = unquoted
			}
		}
		// Strip quotes from QueryData if present (constants store it with quotes)
		if len(params["market_params"][i].QueryData) > 0 && params["market_params"][i].QueryData[0] == '"' {
			params["market_params"][i].QueryData = strings.Trim(params["market_params"][i].QueryData, "\"")
		}
	}

	paramStartupConfigMap := make(map[uint32]*types.MarketParam, len(params))
	for _, param := range params["market_params"] {
		if param.Exponent == 0 ||
			param.MinExchanges == 0 ||
			param.MinPriceChangePpm == 0 ||
			param.QueryData == "" {
			panic(
				fmt.Errorf(
					"one or more config values are unset or are set to zero for pair with id: '%v'",
					param.Id,
				),
			)
		}

		// Insert Key-Value pair into `exchangeStartupConfigMap`.
		paramStartupConfigMap[param.Id] = &types.MarketParam{
			ExchangeConfigJson: param.ExchangeConfigJson,
			Exponent:           param.Exponent,
			Id:                 param.Id,
			MinExchanges:       param.MinExchanges,
			MinPriceChangePpm:  param.MinPriceChangePpm,
			Pair:               param.Pair,
			QueryData:          param.QueryData,
		}
	}
	marketParams := make([]types.MarketParam, 0, len(paramStartupConfigMap))
	for _, param := range paramStartupConfigMap {
		marketParams = append(marketParams, *param)
	}
	return marketParams
}

// getConfigFilePath returns the path to the pricefeed exchange config file.
func getMarketParamsConfigFilePath(homeDir string) string {
	return filepath.Join(
		homeDir,
		"config",
		constants.MarketParamsConfigFileName,
	)
}
