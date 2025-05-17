package customquery

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml"
)

type EndpointTemplate struct {
	URLTemplate string            `toml:"url_template"`
	Method      string            `toml:"method"`
	Timeout     int               `toml:"timeout"`
	ApiKey      string            `toml:"api_key"`
	Headers     map[string]string `toml:"headers"`
}

type Config struct {
	Endpoints map[string]EndpointTemplate `toml:"endpoints"`
	Queries   map[string]QueryConfig      `toml:"queries"`
}

type QueryConfig struct {
	ID                string           `toml:"id"`
	AggregationMethod string           `toml:"aggregation_method"`
	MinResponses      int              `toml:"min_responses"`
	ResponseType      string           `toml:"response_type"`
	Endpoints         []EndpointConfig `toml:"endpoints"`
	BuiltEndpoints    []BuiltEndpoint  `toml:"built_endpoints"`
}

type EndpointConfig struct {
	EndpointType string            `toml:"endpoint_type"`
	ResponsePath []string          `toml:"response_path"`
	Params       map[string]string `toml:"params"`
}
type BuiltEndpoint struct {
	URL          string
	Method       string
	Timeout      int
	ResponsePath []string
	EndpointID   string
	Headers      map[string]string
}

func BuildQueryEndpoints(configPath string) (map[string]QueryConfig, error) {
	// Read the TOML configuration file
	// TODO: move the config file to a more appropriate location
	var config Config
	dir, _ := os.Getwd()
	tomlFile, err := os.ReadFile(filepath.Join(dir, configPath))
	if err != nil {
		panic(err)
	}

	if err = toml.Unmarshal(tomlFile, &config); err != nil {
		fmt.Println("Error unmarshalling toml file", err.Error())
		panic(err)
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
		for _, endpoint := range query.Endpoints {
			// check if the endpoint type exists in the config
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
				endpoint.ApiKey = envValue
				config.Endpoints[endpointName] = endpoint
			}
		}
	}
}
