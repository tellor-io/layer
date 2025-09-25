package customquery

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
)

type EndpointTemplate struct {
	URLTemplate string            `toml:"url_template"`
	Query       string            `toml:"query"` // for POST requests
	Method      string            `toml:"method"`
	Timeout     int               `toml:"timeout"`
	ApiKey      string            `toml:"api_key"`
	Headers     map[string]string `toml:"headers"`
}

type RPCEndpointTemplate struct {
	URLs []string `toml:"urls"`
}
type Config struct {
	Endpoints    map[string]EndpointTemplate    `toml:"endpoints"`
	RPCEndpoints map[string]RPCEndpointTemplate `toml:"rpc_endpoints"`
	Queries      map[string]QueryConfig         `toml:"queries"`
}

type ContractHandler struct {
	Handler  string
	Reader   *contractreader.Reader
	MarketId string
	SourceId string
}

type RpcHandler struct {
	Handler    string
	Reader     *rpcreader.Reader
	Invert     bool
	UsdViaID   uint32
	Method     string
	EndpointID string
	MarketId   string
	SourceId   string
}

type CombinedHandler struct {
	Handler         string
	ContractReaders map[string]*contractreader.Reader
	RpcReaders      map[string]*rpcreader.Reader
	Config          map[string]any
}
type QueryConfig struct {
	ID                string            `toml:"id"`
	AggregationMethod string            `toml:"aggregation_method"`
	MinResponses      int               `toml:"min_responses"`
	ResponseType      string            `toml:"response_type"`
	Endpoints         []EndpointConfig  `toml:"endpoints"`
	ContractReaders   []ContractHandler `toml:"-"`
	RpcReaders        []RpcHandler      `toml:"-"`
	CombinedReaders   []CombinedHandler `toml:"-"`
}

type EndpointConfig struct {
	EndpointType string            `toml:"endpoint_type"`
	ResponsePath []string          `toml:"response_path"`
	Params       map[string]string `toml:"params"`

	// telemtry fields
	MarketId string `toml:"market_id"`

	// Contract-specific fields
	Handler string `toml:"handler"`
	Chain   string `toml:"chain"`
	// cosmosis
	Invert   bool   `toml:"invert"`
	UsdViaID uint32 `toml:"usd_via_id"`

	// Combined handler fields
	CombinedSources map[string]string `toml:"combined_sources"`
	CombinedConfig  map[string]any    `toml:"combined_config"`
}

func BuildQueryEndpoints(homeDir, localDir, file string) (map[string]QueryConfig, error) {
	// Read the TOML configuration file
	tomlFile, err := os.ReadFile(getCustomQueryConfigFilePath(homeDir, localDir, file))
	if err != nil {
		return nil, fmt.Errorf("error reading toml file: %w", err)
	}

	var config Config
	if err = toml.Unmarshal(tomlFile, &config); err != nil {
		fmt.Println("Error unmarshalling toml file", err.Error())
		return nil, fmt.Errorf("error unmarshalling toml file: %w", err)
	}

	// Process RPC endpoints
	processedRPCEndpoints := make(map[string][]string)
	for chain, endpointConfig := range config.RPCEndpoints {
		var urls []string
		for _, url := range endpointConfig.URLs {
			expandedURL := os.ExpandEnv(url)
			// Skip if env var still contains ${}
			if strings.Contains(expandedURL, "${") && strings.Contains(expandedURL, "}") {
				fmt.Printf("Skipping RPC endpoint with missing env var: %s\n", url)
				continue
			}
			urls = append(urls, expandedURL)
		}
		if len(urls) > 0 {
			processedRPCEndpoints[chain] = urls
		}
	}

	// loop through the queries and create a map of query ID to query config
	queryMap := make(map[string]QueryConfig)
	for _, query := range config.Queries {
		queryMap[query.ID] = query
	}
	// process api keys if any
	fmt.Println("\nProcessing API keys for endpoints...")
	processApiKeys(&config)
	time.Sleep(2 * time.Second) // brief pause for readability

	// for each query in the query map, build the endpoints
	for _, query := range config.Queries {
		contractReaders := make([]ContractHandler, 0)
		rpcReaders := make([]RpcHandler, 0)
		combinedReaders := make([]CombinedHandler, 0)
		for _, endpoint := range query.Endpoints {
			// Handle combined endpoints
			if endpoint.EndpointType == "combined" {
				if endpoint.Handler == "" {
					return nil, fmt.Errorf("combined endpoint missing handler for query %s", query.ID)
				}

				// Build contract readers for combined handler
				contractReadersMap := make(map[string]*contractreader.Reader)
				rpcReadersMap := make(map[string]*rpcreader.Reader)

				for sourceName, sourceType := range endpoint.CombinedSources {
					chain, found := strings.CutPrefix(sourceType, "contract:")
					if found {
						urls, exists := processedRPCEndpoints[chain]
						if !exists {
							return nil, fmt.Errorf("no RPC endpoints configured for chain %s in combined source %s for query %s",
								chain, sourceName, query.ID)
						}
						reader, err := contractreader.NewReader(urls, 3)
						if err != nil {
							return nil, fmt.Errorf("failed to create contract reader for combined source %s in query %s: %w",
								sourceName, query.ID, err)
						}
						contractReadersMap[sourceName] = reader
					} else {
						endpointType, found := strings.CutPrefix(sourceType, "rpc:")
						if found {
							template, exists := config.Endpoints[endpointType]
							if !exists {
								return nil, fmt.Errorf("RPC endpoint template not found: %s for combined source %s in query %s",
									endpointType, sourceName, query.ID)
							}

							// Build RPC URL from template
							url := template.URLTemplate

							// Process source-specific parameters (e.g., "sushiswap_api_params" or "coingecko_api_params")
							paramsKey := sourceName + "_params"
							if paramsRaw, exists := endpoint.CombinedConfig[paramsKey]; exists {
								for key, value := range paramsRaw.(map[string]any) {
									placeholder := fmt.Sprintf("{%s}", key)
									url = strings.ReplaceAll(url, placeholder, fmt.Sprintf("%v", value))
								}
							}

							// Replace API key if needed
							url = strings.ReplaceAll(url, "{api_key}", template.ApiKey)

							processedHeaders := make(map[string]string)
							for key, value := range template.Headers {
								if strings.EqualFold(value, "api_key") {
									value = template.ApiKey
								}
								processedHeaders[key] = value
							}

							// Get source-specific response path (e.g., "sushiswap_api_response_path")
							var responsePath []string
							respPathKey := sourceName + "_response_path"
							if respPathRaw, exists := endpoint.CombinedConfig[respPathKey]; exists {
								if respPath, ok := respPathRaw.([]string); ok {
									responsePath = respPath
								} else if respPathInterface, ok := respPathRaw.([]any); ok {
									for _, p := range respPathInterface {
										if str, ok := p.(string); ok {
											responsePath = append(responsePath, str)
										}
									}
								}
							}

							reader, err := rpcreader.NewReader(url, template.Method, template.Query,
								processedHeaders, responsePath, template.Timeout)
							if err != nil {
								return nil, fmt.Errorf("failed to create RPC reader for combined source %s in query %s: %w",
									sourceName, query.ID, err)
							}
							rpcReadersMap[sourceName] = reader
						}
					}
				}

				combinedReaders = append(combinedReaders, CombinedHandler{
					Handler:         endpoint.Handler,
					ContractReaders: contractReadersMap,
					RpcReaders:      rpcReadersMap,
					Config:          endpoint.CombinedConfig,
				})
				continue
			}

			if endpoint.EndpointType == "contract" {
				if endpoint.Handler == "" || endpoint.Chain == "" {
					return nil, fmt.Errorf("contract endpoint missing required fields (handler, chain) for query %s", query.ID)
				}

				urls, exists := processedRPCEndpoints[endpoint.Chain]
				if !exists {
					return nil, fmt.Errorf("no RPC endpoints configured for chain %s in query %s", endpoint.Chain, query.ID)
				}
				contractReader, err := contractreader.NewReader(urls, 3) // 3 second timeout
				if err != nil {
					return nil, fmt.Errorf("failed to create contract reader for chain %s in query %s: %w", endpoint.Chain, query.ID, err)
				}

				contractReaders = append(contractReaders, ContractHandler{
					Handler:  endpoint.Handler,
					Reader:   contractReader,
					MarketId: endpoint.MarketId,
					SourceId: endpoint.EndpointType,
				})
				continue
			}

			// Regular REST API endpoint handling (existing logic)
			template, exists := config.Endpoints[endpoint.EndpointType]
			if !exists {
				return nil, fmt.Errorf("endpoint template not found: %s for query %s",
					endpoint.EndpointType, query.ID)
			}
			url := template.URLTemplate
			// find the placeholders in the URL template
			placeholderRegex := regexp.MustCompile(`\{([^{}]+)\}`)
			matches := placeholderRegex.FindAllStringSubmatch(url, -1)

			for _, match := range matches {
				if len(match) < 2 {
					continue
				}

				paramName := match[1]
				if _, exists := endpoint.Params[paramName]; !exists {
					if paramName == "api_key" {
						// replace with the api key from the config
						url = strings.ReplaceAll(url, "{api_key}", template.ApiKey)
						continue
					}
					return nil, fmt.Errorf("missing required parameter %s for endpoint %s in query %s",
						paramName, endpoint.EndpointType, query.ID)
				}
			}
			// replace all placeholders with their values
			for key, value := range endpoint.Params {
				placeholder := fmt.Sprintf("{%s}", key)
				url = strings.ReplaceAll(url, placeholder, value)
			}

			// Check if any placeholders remain
			if placeholderRegex.MatchString(url) {
				return nil, fmt.Errorf("some placeholders were not replaced in URL: %s", url)
			}
			processedHeaders := make(map[string]string)
			for key, value := range template.Headers {
				if strings.EqualFold(value, "api_key") {
					value = template.ApiKey
				}
				processedHeaders[key] = value
			}

			// Process the query field - replace placeholders with params
			processedQuery := template.Query
			for key, value := range endpoint.Params {
				placeholder := fmt.Sprintf("{%s}", key)
				processedQuery = strings.ReplaceAll(processedQuery, placeholder, value)
			}

			rpcReader, err := rpcreader.NewReader(url, template.Method, processedQuery, processedHeaders, endpoint.ResponsePath, template.Timeout)
			if err != nil {
				return nil, fmt.Errorf("failed to create RPC reader for endpoint %s in query %s: %w", endpoint.EndpointType, query.ID, err)
			}
			rpcReaders = append(rpcReaders, RpcHandler{
				Handler:    endpoint.Handler,
				Reader:     rpcReader,
				Invert:     endpoint.Invert,
				UsdViaID:   endpoint.UsdViaID,
				Method:     template.Method,
				EndpointID: endpoint.EndpointType,
				MarketId:   endpoint.MarketId,
				SourceId:   endpoint.EndpointType,
			})
		}
		queryMap[query.ID] = QueryConfig{
			ID:                query.ID,
			AggregationMethod: query.AggregationMethod,
			MinResponses:      query.MinResponses,
			ResponseType:      query.ResponseType,
			ContractReaders:   contractReaders,
			RpcReaders:        rpcReaders,
			CombinedReaders:   combinedReaders,
		}
	}

	return queryMap, nil
}

func processApiKeys(config *Config) {
	envRegex := regexp.MustCompile(`\${([^{}]+)}`)

	for endpointName, endpoint := range config.Endpoints {
		if envRegex.MatchString(endpoint.ApiKey) {
			matches := envRegex.FindStringSubmatch(endpoint.ApiKey)
			if len(matches) > 1 {
				envVar := matches[1]
				envValue := os.Getenv(envVar)
				if envValue == "" {
					fmt.Printf("⚠️  Warning: API key environment variable '%s' for endpoint '%s' is not set\n", envVar, endpointName)
				} else {
					fmt.Printf("✓ Loaded API key from environment variable '%s' for endpoint '%s'\n", envVar, endpointName)
				}
				endpoint.ApiKey = envValue
				config.Endpoints[endpointName] = endpoint
			}
		}
	}
}
