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
	# Pair - The human-readable name of the market pair (e.g. "BTC-USD").{{ range $exchangeId, $element := .}}
	[[market_params]]
	ExchangeConfigJson = "{{$element.ExchangeConfigJson}}"
	Exponent = {{$element.Exponent}}
	Id = {{$element.Id}}
	MinExchanges = {{$element.MinExchanges}}
	MinPriceChangePpm = {{$element.MinPriceChangePpm}}
	Pair = {{$element.Pair}}{{end}}
`
)

// GenerateDefaultExchangeTomlString creates the toml file string containing the default marketParam configs.
func GenerateDefaultMarketParamsTomlString() bytes.Buffer {
	template, err := template.New("").Parse(defaultMarketParamTomlTemplate)
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

func WriteDefaultMarketParamsToml(homeDir string) {
	// Write file into config folder if file does not exist.
	configFilePath := getMarketParamsConfigFilePath(homeDir)
	if !tmos.FileExists(configFilePath) {
		buffer := GenerateDefaultMarketParamsTomlString()
		tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0644)
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
		panic(err)
	}

	paramStartupConfigMap := make(map[uint32]*types.MarketParam, len(params))
	for _, param := range params["market_params"] {
		if param.Exponent == 0 ||
			param.MinExchanges == 0 ||
			param.MinPriceChangePpm == 0 {
			panic(
				fmt.Errorf(
					"One or more config values are unset or are set to zero for pair with id: '%v'",
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
