package daemons

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/pelletier/go-toml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type TelemetryConfig struct {
	ServiceName             string     `mapstructure:"service-name" toml:"service-name"`
	Enabled                 bool       `mapstructure:"enabled" toml:"enabled"`
	EnableHostname          bool       `mapstructure:"enable-hostname" toml:"enable-hostname"`
	EnableHostnameLabel     bool       `mapstructure:"enable-hostname-label" toml:"enable-hostname-label"`
	EnableServiceLabel      bool       `mapstructure:"enable-service-label" toml:"enable-service-label"`
	PrometheusRetentionTime int64      `mapstructure:"prometheus-retention-time" toml:"prometheus-retention-time"`
	GlobalLabels            [][]string `mapstructure:"global-labels" toml:"global-labels"`
	MetricsSink             string     `mapstructure:"metrics-sink" toml:"metrics-sink"`
	StatsdAddr              string     `mapstructure:"statsd-addr" toml:"statsd-addr"`
	DatadogHostname         string     `mapstructure:"datadog-hostname" toml:"datadog-hostname"`
}

// ConfigWrapper is used to unmarshal the telemetry section from app.toml
type ConfigWrapper struct {
	Telemetry TelemetryConfig `toml:"telemetry"`
}

// RegisterTelemetryIfEnabled reads the telemetry config from app.toml and initializes telemetry if enabled.
// prometheusPort: the port to serve Prometheus metrics on. If 0, defaults to 26661.
func RegisterTelemetryIfEnabled(logger log.Logger, homePath string, prometheusPort int) {
	logger.Info("Registering telemetry if enabled")
	configPath := filepath.Join(homePath, "config", "app.toml")
	logger.Info("configPath", "path", configPath)
	file, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error("failed to read app.toml for telemetry", "error", err)
		panic(fmt.Sprintf("failed to read app.toml for telemetry: %v", err))
	}
	tomlTree, err := toml.LoadBytes(file)
	if err != nil {
		logger.Error("failed to parse app.toml for telemetry", "error", err)
		panic(fmt.Sprintf("failed to parse app.toml for telemetry: %v", err))
	}
	telemetryTree := tomlTree.Get("telemetry")
	if telemetryTree == nil {
		// No telemetry section
		return
	}

	// Unmarshal the entire config file to get the telemetry section
	var configWrapper ConfigWrapper
	if err := toml.Unmarshal(file, &configWrapper); err != nil {
		logger.Error("failed to unmarshal telemetry config", "error", err)
		panic(fmt.Sprintf("failed to unmarshal telemetry config: %v", err))
	}
	telemetryConfig := configWrapper.Telemetry
	logger.Info("Telemetry config", "config", telemetryConfig)
	if telemetryConfig.Enabled {
		logger.Info("Telemetry is enabled")
		cosmosConfig := telemetry.Config{
			ServiceName:             telemetryConfig.ServiceName,
			Enabled:                 telemetryConfig.Enabled,
			EnableHostname:          telemetryConfig.EnableHostname,
			EnableHostnameLabel:     telemetryConfig.EnableHostnameLabel,
			EnableServiceLabel:      telemetryConfig.EnableServiceLabel,
			PrometheusRetentionTime: telemetryConfig.PrometheusRetentionTime,
			GlobalLabels:            telemetryConfig.GlobalLabels,
			MetricsSink:             telemetryConfig.MetricsSink,
			StatsdAddr:              telemetryConfig.StatsdAddr,
			DatadogHostname:         telemetryConfig.DatadogHostname,
		}
		_, err := telemetry.New(cosmosConfig)
		if err != nil {
			logger.Error("failed to initialize telemetry", "error", err)
			panic(fmt.Sprintf("failed to initialize telemetry: %v", err))
		}

		// Start Prometheus HTTP server if PrometheusRetentionTime is set
		if telemetryConfig.PrometheusRetentionTime > 0 {
			port := prometheusPort
			if port == 0 {
				port = 26661 // default port
			}
			logger.Info("Starting Prometheus metrics HTTP server", "port", port)
			go func() {
				mux := http.NewServeMux()
				mux.Handle("/metrics", promhttp.Handler())
				addr := ":" + strconv.Itoa(port)
				logger.Info("Starting Prometheus metrics HTTP server", "port", port)
				if err := http.ListenAndServe(addr, mux); err != nil {
					logger.Error("Prometheus HTTP server failed", "error", err)
					panic(fmt.Sprintf("Prometheus HTTP server failed: %v", err))
				}
			}()
		} else {
			logger.Info("Telemetry is disabled")
		}
	}
}
