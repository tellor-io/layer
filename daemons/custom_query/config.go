package customquery

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml"
	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
)

type EndpointTemplate struct {
	URLTemplate string            `toml:"url_template"`
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
	Handler string
	Reader  *contractreader.Reader
}

type RpcHandler struct {
	Handler  string
	Reader   *rpcreader.Reader
	Invert   bool
	UsdViaID int
}
type QueryConfig struct {
	ID                string            `toml:"id"`
	AggregationMethod string            `toml:"aggregation_method"`
	MinResponses      int               `toml:"min_responses"`
	ResponseType      string            `toml:"response_type"`
	Endpoints         []EndpointConfig  `toml:"endpoints"`
	BuiltEndpoints    []BuiltEndpoint   `toml:"built_endpoints"`
	ContractReaders   []ContractHandler `toml:"-"`
	RpcReaders        []RpcHandler      `toml:"-"`
}

type EndpointConfig struct {
	EndpointType string            `toml:"endpoint_type"`
	ResponsePath []string          `toml:"response_path"`
	Params       map[string]string `toml:"params"`

	// Contract-specific fields
	Handler string `toml:"handler"`
	Chain   string `toml:"chain"`
	// cosmosis
	Invert   bool `toml:"invert"`
	UsdViaID int  `toml:"usd_via_id"`
}
type BuiltEndpoint struct {
	URL          string
	Method       string
	Timeout      int
	ResponsePath []string
	EndpointID   string
	Headers      map[string]string
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
	processApiKeys(&config)
	// for each query in the query map, build the endpoints
	for _, query := range config.Queries {
		result := make([]BuiltEndpoint, 0, len(query.Endpoints))
		contractReaders := make([]ContractHandler, 0)
		rpcReaders := make([]RpcHandler, 0)
		for _, endpoint := range query.Endpoints {
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
					Handler: endpoint.Handler,
					Reader:  contractReader,
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
			if endpoint.EndpointType == "cosmos" {
				rpcReader, err := rpcreader.NewReader(url, processedHeaders, endpoint.ResponsePath, 3)
				if err != nil {
					return nil, fmt.Errorf("failed to create RPC reader for endpoint %s in query %s: %w", endpoint.EndpointType, query.ID, err)
				}
				rpcReaders = append(rpcReaders, RpcHandler{
					Handler:  endpoint.Handler,
					Reader:   rpcReader,
					Invert:   endpoint.Invert,
					UsdViaID: endpoint.UsdViaID,
				})
				continue
			}
			result = append(result, BuiltEndpoint{
				URL:          url,
				Method:       template.Method,
				Timeout:      template.Timeout,
				ResponsePath: endpoint.ResponsePath,
				EndpointID:   endpoint.EndpointType,
				Headers:      processedHeaders,
			})
		}
		queryMap[query.ID] = QueryConfig{
			ID:                query.ID,
			AggregationMethod: query.AggregationMethod,
			MinResponses:      query.MinResponses,
			ResponseType:      query.ResponseType,
			BuiltEndpoints:    result,
			ContractReaders:   contractReaders,
			RpcReaders:        rpcReaders,
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
