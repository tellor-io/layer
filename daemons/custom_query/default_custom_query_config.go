package customquery

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/pelletier/go-toml"
)

type CombinedConfig struct {
	Endpoints    map[string]*EndpointTemplate
	RPCEndpoints map[string]*RPCEndpointTemplate
	Queries      map[string]*QueryConfig
}

const (
	defaultCustomQueryTomlTemplate = `# This is a TOML config file.
[endpoints]
{{- range $key, $endpoint := .Endpoints }}
    [endpoints.{{ $key }}]
    url_template = "{{ $endpoint.URLTemplate }}"
    method = "{{ $endpoint.Method }}"
    timeout = {{ $endpoint.Timeout }}
    {{- if $endpoint.Query }}
    query = '''{{ $endpoint.Query }}'''
    {{- end }}
    {{- if $endpoint.ApiKey }}
    api_key = "{{ $endpoint.ApiKey }}"
    {{- end }}
    {{- if $endpoint.Headers }}
    headers = { {{ formatParams $endpoint.Headers }} }
    {{- end }}
{{- end }}
[rpc_endpoints]
{{- range $key, $rpcEndpoint := .RPCEndpoints }}
    [rpc_endpoints.{{ $key }}]
    urls = [{{ range $i, $url := $rpcEndpoint.URLs }}{{if $i}}, {{end}}"{{ $url }}"{{ end }}]
{{- end }}

[queries]
{{- range $key, $query := .Queries }}
    [queries.{{ $key }}]
    id = "{{ $query.ID }}"
    aggregation_method = "{{ $query.AggregationMethod }}"
	max_spread_percent = {{ tomlValue $query.MaxSpreadPercent }}
    min_responses = {{ $query.MinResponses }}
    response_type = "{{ $query.ResponseType }}"

    {{- range $idx, $endpoint := $query.Endpoints }}
        [[queries.{{ $key }}.endpoints]]
        endpoint_type = "{{ $endpoint.EndpointType }}"
        response_path = [{{ range $i, $path := $endpoint.ResponsePath }}{{if $i}}, {{end}}"{{ $path }}"{{ end }}]
        params = { {{ formatParams $endpoint.Params }} }
		{{- if $endpoint.MarketId }}
        market_id = "{{ $endpoint.MarketId }}"
		{{- end }}
		{{- if $endpoint.Handler }}
        handler = "{{ $endpoint.Handler }}"
		{{- end }}
		{{- if $endpoint.Chain }}
		chain = "{{ $endpoint.Chain }}"
		{{- end }}
		{{- if $endpoint.Invert }}
		invert = {{ $endpoint.Invert }}
		{{- end }}
		{{- if $endpoint.UsdViaID }}
		usd_via_id = {{ $endpoint.UsdViaID }}
		{{- end }}
		{{- if $endpoint.CombinedSources }}
		combined_sources = { {{ formatCombinedSources $endpoint.CombinedSources }} }
		{{- end }}
		{{- if $endpoint.CombinedConfig }}
		[queries.{{ $key }}.endpoints.combined_config]
		    {{- range $k, $v := $endpoint.CombinedConfig }}
		        {{- if hasSuffix $k "_response_path" }}
		    {{ $k }} = [{{ range $i, $path := $v }}{{if $i}}, {{end}}"{{ $path }}"{{ end }}]
		        {{- else if hasSuffix $k "_params" }}
		    {{ $k }} = { {{ formatMapStringString $v }} }
		        {{- else }}
		    {{ $k }} = {{ tomlValue $v }}
		        {{- end }}
		    {{- end }}
		{{- end }}
    {{- end }}
{{- end }}
`
)

// Helper function to format parameters as a comma-separated list
func formatParams(params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, fmt.Sprintf(`%s = "%s"`, k, v))
	}
	return strings.Join(parts, ", ")
}

// Helper function to format combined sources as a comma-separated list
func formatCombinedSources(sources map[string]string) string {
	var parts []string
	for k, v := range sources {
		parts = append(parts, fmt.Sprintf(`%s = "%s"`, k, v))
	}
	return strings.Join(parts, ", ")
}

// Helper function to format map[string]string as TOML inline table
func formatMapStringString(m any) string {
	var parts []string
	switch v := m.(type) {
	case map[string]string:
		for k, val := range v {
			parts = append(parts, fmt.Sprintf(`%s = "%s"`, k, val))
		}
	case map[string]any:
		for k, val := range v {
			parts = append(parts, fmt.Sprintf(`%s = "%v"`, k, val))
		}
	}
	return strings.Join(parts, ", ")
}

// Helper function to convert a value to TOML format
func tomlValue(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, val)
	case bool:
		return fmt.Sprintf("%t", val)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	default:
		return fmt.Sprintf(`"%v"`, val)
	}
}

func GenerateDefaultConfigTomlString() bytes.Buffer {
	// Create the combined config
	combined := CombinedConfig{
		Endpoints:    StaticEndpointTemplateConfig,
		Queries:      StaticQueriesConfig,
		RPCEndpoints: StaticRPCEndpointTemplateConfig,
	}

	// Create a template with the helper function
	tmpl := template.New("config").Funcs(template.FuncMap{
		"formatParams":          formatParams,
		"formatCombinedSources": formatCombinedSources,
		"formatMapStringString": formatMapStringString,
		"tomlValue":             tomlValue,
		"hasSuffix":             strings.HasSuffix,
	})

	tmpl, err := tmpl.Parse(defaultCustomQueryTomlTemplate)
	if err != nil {
		panic(err)
	}

	// Execute the template with the combined config
	var configToml bytes.Buffer
	err = tmpl.Execute(&configToml, combined)
	if err != nil {
		panic(err)
	}

	return configToml
}

// readExistingCustomQueryConfig reads and parses an existing custom query config file.
func readExistingCustomQueryConfig(homeDir, localDir, file string) (*Config, error) {
	configFilePath := getCustomQueryConfigFilePath(homeDir, localDir, file)
	tomlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing config file: %w", err)
	}

	var config Config
	if err = toml.Unmarshal(tomlFile, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal existing config: %w", err)
	}

	return &config, nil
}

// MergeCustomQueryConfig merges missing entries from static configs into existing config file.
// It preserves existing entries and only adds new ones with default values.
func MergeCustomQueryConfig(homeDir, localDir, file string) error {
	configFilePath := getCustomQueryConfigFilePath(homeDir, localDir, file)

	// Read existing config
	existingConfig, err := readExistingCustomQueryConfig(homeDir, localDir, file)
	if err != nil {
		return err
	}

	// Initialize maps if nil
	if existingConfig.Endpoints == nil {
		existingConfig.Endpoints = make(map[string]EndpointTemplate)
	}
	if existingConfig.RPCEndpoints == nil {
		existingConfig.RPCEndpoints = make(map[string]RPCEndpointTemplate)
	}
	if existingConfig.Queries == nil {
		existingConfig.Queries = make(map[string]QueryConfig)
	}

	// Merge endpoints
	mergedEndpoints := make(map[string]*EndpointTemplate)
	for key, endpoint := range existingConfig.Endpoints {
		endpoint := endpoint // Create local copy to avoid taking address of loop variable
		mergedEndpoints[key] = &endpoint
	}
	for key, defaultEndpoint := range StaticEndpointTemplateConfig {
		if _, exists := mergedEndpoints[key]; !exists {
			mergedEndpoints[key] = defaultEndpoint
		}
	}

	// Merge RPC endpoints
	mergedRPCEndpoints := make(map[string]*RPCEndpointTemplate)
	for key, rpcEndpoint := range existingConfig.RPCEndpoints {
		rpcEndpoint := rpcEndpoint // Create local copy to avoid taking address of loop variable
		mergedRPCEndpoints[key] = &rpcEndpoint
	}
	for key, defaultRPCEndpoint := range StaticRPCEndpointTemplateConfig {
		if _, exists := mergedRPCEndpoints[key]; !exists {
			mergedRPCEndpoints[key] = defaultRPCEndpoint
		}
	}

	// Merge queries
	mergedQueries := make(map[string]*QueryConfig)
	for key, query := range existingConfig.Queries {
		query := query // Create local copy to avoid taking address of loop variable
		mergedQueries[key] = &query
	}
	for key, defaultQuery := range StaticQueriesConfig {
		if _, exists := mergedQueries[key]; !exists {
			mergedQueries[key] = defaultQuery
		}
	}

	// Create combined config for template
	combined := CombinedConfig{
		Endpoints:    mergedEndpoints,
		RPCEndpoints: mergedRPCEndpoints,
		Queries:      mergedQueries,
	}

	// Generate merged TOML using template
	tmpl := template.New("config").Funcs(template.FuncMap{
		"formatParams":          formatParams,
		"formatCombinedSources": formatCombinedSources,
		"formatMapStringString": formatMapStringString,
		"tomlValue":             tomlValue,
		"hasSuffix":             strings.HasSuffix,
	})

	tmpl, err = tmpl.Parse(defaultCustomQueryTomlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var mergedToml bytes.Buffer
	if err = tmpl.Execute(&mergedToml, combined); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Validate merged config by reading it back
	tempFile := configFilePath + ".tmp"
	if err = tmos.WriteFile(tempFile, mergedToml.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Validate by attempting to read and parse it
	testConfig := Config{}
	testToml, err := os.ReadFile(tempFile)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to read temp file for validation: %w", err)
	}
	if err = toml.Unmarshal(testToml, &testConfig); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("merged config validation failed: %w", err)
	}

	// Replace original file with validated merged config
	if err = os.Rename(tempFile, configFilePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
}

func WriteDefaultConfigToml(homeDir, localDir, file string) {
	// Write file into config folder if file does not exist.
	// If the file exists, merge missing entries from static configs.
	configFilePath := getCustomQueryConfigFilePath(homeDir, localDir, file)
	configDir := filepath.Dir(configFilePath)
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		// Check if directory actually exists (might have been created by another process)
		if _, statErr := os.Stat(configDir); statErr != nil {
			panic(fmt.Sprintf("failed to create config directory %s: %v", configDir, err))
		}
	}
	_, err := os.Stat(configFilePath)
	if err != nil {
		// if the file does not exist, create it
		if !os.IsNotExist(err) {
			panic(fmt.Sprintf("Error checking file: %v", err))
		}
		buffer := GenerateDefaultConfigTomlString()
		tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0o644)
	} else {
		// File exists, merge missing entries
		if err := MergeCustomQueryConfig(homeDir, localDir, file); err != nil {
			panic(fmt.Sprintf("failed to merge custom query config: %v", err))
		}
	}
}

func getCustomQueryConfigFilePath(homeDir, localDir, file string) string {
	return filepath.Join(
		homeDir,
		localDir,
		file,
	)
}
