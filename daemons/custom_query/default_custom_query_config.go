package customquery

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tmos "github.com/cometbft/cometbft/libs/os"
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
    min_responses = {{ $query.MinResponses }}
    response_type = "{{ $query.ResponseType }}"

    {{- range $idx, $endpoint := $query.Endpoints }}
        [[queries.{{ $key }}.endpoints]]
        endpoint_type = "{{ $endpoint.EndpointType }}"
        response_path = [{{ range $i, $path := $endpoint.ResponsePath }}{{if $i}}, {{end}}"{{ $path }}"{{ end }}]
        params = { {{ formatParams $endpoint.Params }} }
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

func WriteDefaultConfigToml(homeDir, localDir, file string) {
	// Write file into config folder if file does not exist.
	configFilePath := getCustomQueryConfigFilePath(homeDir, localDir, file)
	_, err := os.Stat(configFilePath)
	if err != nil {
		// if the file does not exist, create it
		if !os.IsNotExist(err) {
			panic(fmt.Sprintf("Error checking file: %v", err))
		}
		buffer := GenerateDefaultConfigTomlString()
		tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0o644)
	}
}

func getCustomQueryConfigFilePath(homeDir, localDir, file string) string {
	return filepath.Join(
		homeDir,
		localDir,
		file,
	)
}
