package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/api"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/enrich"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/health"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/ingest"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/metrics"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/notify"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/power"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rpc"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rules"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/state"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (required)")
	dryRunFlag := flag.Bool("dry-run", false, "log alerts instead of sending to Discord")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn, error")
	logFormat := flag.String("log-format", "text", "log format: text or json")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "usage: chain-monitor -config=<path> [-dry-run] [-log-level=info] [-log-format=text]")
		os.Exit(2)
	}

	log, err := newLogger(*logLevel, *logFormat)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load config failed", "err", err)
		os.Exit(1)
	}
	if *dryRunFlag {
		cfg.DryRun = true
	}

	client, err := rpc.NewClient(cfg.RPC.URLs, cfg.RPC.Timeout, cfg.RPC.MinInterval)
	if err != nil {
		log.Error("create rpc client failed", "err", err)
		os.Exit(1)
	}

	queryMap, err := enrich.NewQueryIDMap(cfg.Enrichment.QueryIDsMap, log)
	if err != nil {
		log.Error("load query ids map failed", "err", err)
		os.Exit(1)
	}

	reporterMap, err := enrich.NewReporterMap(cfg.Enrichment.ReportersMap, log)
	if err != nil {
		log.Error("load reporters map failed", "err", err)
		os.Exit(1)
	}

	important := enrich.ParseImportantReporters(os.Getenv("IMPORTANT_REPORTERS"))
	apiURL := strings.TrimSpace(cfg.API.URL)
	if apiURL == "" {
		apiURL = strings.TrimSpace(os.Getenv("LAYER_API_URL"))
	}
	oracleClient, err := api.NewClient(apiURL, cfg.RPC.Timeout)
	if err != nil {
		log.Error("create api client failed", "err", err)
		os.Exit(1)
	}
	if len(important) > 0 {
		log.Info("loaded IMPORTANT_REPORTERS", "count", len(important))
		base := ""
		if oracleClient != nil {
			base = oracleClient.BaseURL()
		}
		if base == "" {
			log.Warn("IMPORTANT_REPORTERS is set but api.url / LAYER_API_URL is empty; per-aggregate missing-reporter checks will be skipped")
		} else if api.LooksLikeTendermint(base) {
			log.Warn("IMPORTANT_REPORTERS is set but REST base looks like Tendermint (:26657); set api.url or LAYER_API_URL to LCD (e.g. :1317)",
				"api_url", base)
		} else {
			log.Info("REST base for oracle queries", "api_url", base)
		}
	}

	reg := metrics.New()
	powerCache := power.NewCache(client, cfg.Power.RefreshInterval, reg, log)
	valset := state.NewValsetStore(cfg.State.ValsetTimestampsPath)

	cursor := state.NewCursorFile(cfg.State.CursorPath)
	tracker := health.NewTracker(cfg.NodeName)
	healthServer := health.NewServer(cfg.Health.Listen, tracker, reg, log)
	engineOpts := rules.EngineOpts{
		QueryIDs:  queryMap,
		Reporters: reporterMap,
		Important: important,
		Power:     powerCache,
		Valset:    valset,
		Log:       log,
	}
	if oracleClient != nil {
		engineOpts.Oracle = oracleClient
	}
	engine := rules.NewEngine(cfg, engineOpts)

	var sender notify.Sender
	if cfg.DryRun {
		sender = notify.NewDryRunSender(log)
		log.Info("dry-run enabled; alerts will be logged only")
	} else {
		sender = notify.NewDiscordSender(15*time.Second, log)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	stopReload := make(chan struct{})
	defer close(stopReload)
	queryMap.StartReloader(stopReload, cfg.Enrichment.ReloadInterval)
	reporterMap.StartReloader(stopReload, cfg.Enrichment.ReloadInterval)

	go powerCache.Start(ctx)

	errCh := make(chan error, 2)

	go func() {
		if err := healthServer.Start(); err != nil {
			errCh <- fmt.Errorf("health server: %w", err)
		}
	}()

	poller := ingest.NewPoller(client, cursor, tracker, engine, sender, reg, log, ingest.Config{
		NodeName:     cfg.NodeName,
		StartHeight:  cfg.StartFrom.Height,
		Channels:     cfg.Channels,
		LogRateLimit: cfg.Defaults.LogRateLimit,
	})

	go ingest.RunSchedules(ctx, engine, poller.Dispatcher(), log)

	go func() {
		errCh <- poller.Run(ctx)
	}()

	log.Info("chain-monitor started",
		"node", cfg.NodeName,
		"rpc_urls", cfg.RPC.URLs,
		"cursor", cfg.State.CursorPath,
		"health", cfg.Health.Listen,
		"rules", len(cfg.Rules),
		"dry_run", cfg.DryRun,
		"power_refresh", cfg.Power.RefreshInterval,
		"log_level", *logLevel,
		"log_format", *logFormat,
	)

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && ctx.Err() == nil {
			log.Error("fatal error", "err", err)
			cancel()
			shutdownHealth(healthServer, log)
			os.Exit(1)
		}
	}

	cancel()
	shutdownHealth(healthServer, log)
	log.Info("shutdown complete")
}

func newLogger(level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "info", "":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid -log-level %q (want debug|info|warn|error)", level)
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text", "":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		return nil, fmt.Errorf("invalid -log-format %q (want text|json)", format)
	}
	return slog.New(handler), nil
}

func shutdownHealth(s *health.Server, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Warn("health server shutdown", "err", err)
	}
}
