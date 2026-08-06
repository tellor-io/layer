package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Kind constants for non-event rules.
const (
	KindEvent         = "" // or "event"
	KindBlockInterval = "block_interval"
	KindRPCUnhealthy  = "rpc_unhealthy"
	KindIngestLag     = "ingest_lag"
	KindSchedule      = "schedule"
)

// Config is the top-level monitor configuration.
type Config struct {
	NodeName    string             `yaml:"node_name"`
	RPC         RPCConfig          `yaml:"rpc"`
	State       StateConfig        `yaml:"state"`
	Health      HealthConfig       `yaml:"health"`
	StartFrom   StartFromConfig    `yaml:"start_from"`
	Enrichment  EnrichmentConfig   `yaml:"enrichment"`
	Power       PowerConfig        `yaml:"power"`
	Channels    map[string]Channel `yaml:"channels"`
	Defaults    DefaultsConfig     `yaml:"defaults"`
	Rules       []RuleConfig       `yaml:"rules"`
	DryRun      bool               `yaml:"dry_run"`
	WatchEvents []string           `yaml:"watch_events"`
}

type RPCConfig struct {
	URLs        []string      `yaml:"urls"`
	Timeout     time.Duration `yaml:"timeout"`
	MinInterval time.Duration `yaml:"min_interval"`
}

type StateConfig struct {
	CursorPath           string `yaml:"cursor_path"`
	ValsetTimestampsPath string `yaml:"valset_timestamps_path"`
}

type HealthConfig struct {
	Listen string `yaml:"listen"`
}

type StartFromConfig struct {
	Height uint64 `yaml:"height"`
	Tip    bool   `yaml:"tip"`
}

type EnrichmentConfig struct {
	QueryIDsMap    string        `yaml:"query_ids_map"`
	ReloadInterval time.Duration `yaml:"reload_interval"`
}

type PowerConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
}

type Channel struct {
	WebhookURL string `yaml:"webhook_url"`
}

type DefaultsConfig struct {
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`     // Discord delivery
	LogRateLimit LogRateLimitConfig `yaml:"log_rate_limit"` // journald / log agents
}

type RateLimitConfig struct {
	Max      int           `yaml:"max"`
	Window   time.Duration `yaml:"window"`
	Cooldown time.Duration `yaml:"cooldown"`
}

// LogRateLimitConfig caps how often full alert bodies are written to logs.
// Max 0 means unlimited (default): every rule match is logged even when Discord
// is deduped or rate-limited. Set Max > 0 as a safety valve against log blow-up.
type LogRateLimitConfig struct {
	Max    int           `yaml:"max"`
	Window time.Duration `yaml:"window"`
}

type RuleConfig struct {
	ID          string           `yaml:"id"`
	Channel     string           `yaml:"channel"`
	Match       MatchConfig      `yaml:"match"`
	When        WhenConfig       `yaml:"when"`
	Embed       EmbedConfig      `yaml:"embed"`
	Enrich      []string         `yaml:"enrich"`
	RateLimit   *RateLimitConfig `yaml:"rate_limit"`
	DedupeAttrs []string         `yaml:"dedupe_attrs"`
	SideEffects []SideEffect     `yaml:"side_effects"`
}

type MatchConfig struct {
	Kind      string   `yaml:"kind"` // empty/event | block_interval | rpc_unhealthy | ingest_lag | schedule
	EventType string   `yaml:"event_type"`
	AttrExists []string `yaml:"attr_exists"`
}

type WhenConfig struct {
	AttrUintLtRatio *AttrUintLtRatio `yaml:"attr_uint_lt_ratio"`
	MaxInterval     time.Duration    `yaml:"max_interval"`
	FailThreshold   int              `yaml:"fail_threshold"`
	CheckEvery      time.Duration    `yaml:"check_every"`
	MaxLag          uint64           `yaml:"max_lag"`
	DailyAt         string           `yaml:"daily_at"` // "15:04" local time
	Lookback        time.Duration    `yaml:"lookback"`
}

type AttrUintLtRatio struct {
	Attr    string  `yaml:"attr"`
	Ratio   float64 `yaml:"ratio"`
	Against string  `yaml:"against"` // validator_power
}

type SideEffect struct {
	RecordAttrAsTimestamp string `yaml:"record_attr_as_timestamp"`
}

type EmbedConfig struct {
	Content          string        `yaml:"content"`
	Title            string        `yaml:"title"`
	Color            uint32        `yaml:"color"`
	Fields           []FieldConfig `yaml:"fields"`
	IncludeRemaining bool          `yaml:"include_remaining"`
}

type FieldConfig struct {
	Attr   string `yaml:"attr"`
	Name   string `yaml:"name"`
	Inline bool   `yaml:"inline"`
	Format string `yaml:"format"` // raw | code | amount_trb
}

// Load reads and validates a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.RPC.Timeout == 0 {
		c.RPC.Timeout = 30 * time.Second
	}
	if c.RPC.MinInterval == 0 {
		c.RPC.MinInterval = 250 * time.Millisecond
	}
	if c.State.CursorPath == "" {
		c.State.CursorPath = "./data/cursor.json"
	}
	if c.Health.Listen == "" {
		c.Health.Listen = ":8080"
	}
	if c.StartFrom.Height == 0 {
		c.StartFrom.Tip = true
	}
	if c.Enrichment.ReloadInterval == 0 {
		c.Enrichment.ReloadInterval = time.Hour
	}
	if c.Power.RefreshInterval == 0 {
		c.Power.RefreshInterval = 10 * time.Minute
	}
	if c.Defaults.RateLimit.Max == 0 {
		c.Defaults.RateLimit.Max = 10
	}
	if c.Defaults.RateLimit.Window == 0 {
		c.Defaults.RateLimit.Window = 10 * time.Minute
	}
	if c.Defaults.RateLimit.Cooldown == 0 {
		c.Defaults.RateLimit.Cooldown = 2 * time.Hour
	}
	// LogRateLimit.Max defaults to 0 (unlimited). Only fill window if a cap is set.
	if c.Defaults.LogRateLimit.Max > 0 && c.Defaults.LogRateLimit.Window == 0 {
		c.Defaults.LogRateLimit.Window = time.Minute
	}
	if c.Channels == nil {
		c.Channels = map[string]Channel{}
	}
	for i := range c.Rules {
		r := &c.Rules[i]
		kind := strings.ToLower(strings.TrimSpace(r.Match.Kind))
		if kind == "event" {
			kind = KindEvent
			r.Match.Kind = ""
		}
		switch kind {
		case KindRPCUnhealthy:
			if r.When.FailThreshold == 0 {
				r.When.FailThreshold = 3
			}
			if r.When.CheckEvery == 0 {
				r.When.CheckEvery = 30 * time.Second
			}
		case KindIngestLag:
			if r.When.MaxLag == 0 {
				r.When.MaxLag = 50
			}
			if r.When.CheckEvery == 0 {
				r.When.CheckEvery = 30 * time.Second
			}
		case KindSchedule:
			if r.When.DailyAt == "" {
				r.When.DailyAt = "09:00"
			}
			if r.When.Lookback == 0 {
				r.When.Lookback = 14 * 24 * time.Hour
			}
		case KindBlockInterval:
			if r.When.MaxInterval == 0 {
				r.When.MaxInterval = 5 * time.Minute
			}
		}
		if r.When.AttrUintLtRatio != nil && r.When.AttrUintLtRatio.Against == "" {
			r.When.AttrUintLtRatio.Against = "validator_power"
		}
	}
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.NodeName) == "" {
		return fmt.Errorf("node_name is required")
	}
	if len(c.RPC.URLs) == 0 {
		return fmt.Errorf("rpc.urls must contain at least one URL")
	}
	for i, u := range c.RPC.URLs {
		u = strings.TrimSpace(u)
		if u == "" {
			return fmt.Errorf("rpc.urls[%d] is empty", i)
		}
		c.RPC.URLs[i] = normalizeRPCURL(u)
	}
	if c.RPC.Timeout <= 0 {
		return fmt.Errorf("rpc.timeout must be positive")
	}
	if c.RPC.MinInterval < 0 {
		return fmt.Errorf("rpc.min_interval must be non-negative")
	}
	if strings.TrimSpace(c.State.CursorPath) == "" {
		return fmt.Errorf("state.cursor_path is required")
	}
	if c.StartFrom.Height > 0 && c.StartFrom.Tip {
		return fmt.Errorf("start_from: set either height or tip, not both")
	}
	if c.StartFrom.Height == 0 && !c.StartFrom.Tip {
		return fmt.Errorf("start_from: set tip: true or a height")
	}

	ruleIDs := make(map[string]struct{}, len(c.Rules))
	for i, rule := range c.Rules {
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("rules[%d].id is required", i)
		}
		if _, dup := ruleIDs[rule.ID]; dup {
			return fmt.Errorf("rules: duplicate id %q", rule.ID)
		}
		ruleIDs[rule.ID] = struct{}{}

		if strings.TrimSpace(rule.Channel) == "" {
			return fmt.Errorf("rules[%d] (%s): channel is required", i, rule.ID)
		}
		ch, ok := c.Channels[rule.Channel]
		if !ok {
			return fmt.Errorf("rules[%d] (%s): unknown channel %q", i, rule.ID, rule.Channel)
		}
		if !c.DryRun && strings.TrimSpace(ch.WebhookURL) == "" {
			return fmt.Errorf("channels.%s.webhook_url is required (or set dry_run: true)", rule.Channel)
		}
		if strings.TrimSpace(rule.Embed.Title) == "" {
			return fmt.Errorf("rules[%d] (%s): embed.title is required", i, rule.ID)
		}

		kind := strings.ToLower(strings.TrimSpace(rule.Match.Kind))
		switch kind {
		case KindEvent, "event":
			if strings.TrimSpace(rule.Match.EventType) == "" {
				return fmt.Errorf("rules[%d] (%s): match.event_type is required for event rules", i, rule.ID)
			}
		case KindBlockInterval:
			if rule.When.MaxInterval <= 0 {
				return fmt.Errorf("rules[%d] (%s): when.max_interval must be positive", i, rule.ID)
			}
		case KindRPCUnhealthy:
			if rule.When.FailThreshold <= 0 {
				return fmt.Errorf("rules[%d] (%s): when.fail_threshold must be positive", i, rule.ID)
			}
		case KindIngestLag:
			if rule.When.MaxLag == 0 {
				return fmt.Errorf("rules[%d] (%s): when.max_lag must be positive", i, rule.ID)
			}
		case KindSchedule:
			if _, _, err := ParseDailyAt(rule.When.DailyAt); err != nil {
				return fmt.Errorf("rules[%d] (%s): when.daily_at: %w", i, rule.ID, err)
			}
		default:
			return fmt.Errorf("rules[%d] (%s): unknown match.kind %q", i, rule.ID, rule.Match.Kind)
		}

		if rule.When.AttrUintLtRatio != nil {
			w := rule.When.AttrUintLtRatio
			if w.Attr == "" {
				return fmt.Errorf("rules[%d] (%s): when.attr_uint_lt_ratio.attr is required", i, rule.ID)
			}
			if w.Ratio <= 0 || w.Ratio > 1 {
				return fmt.Errorf("rules[%d] (%s): when.attr_uint_lt_ratio.ratio must be in (0,1]", i, rule.ID)
			}
			if w.Against != "validator_power" {
				return fmt.Errorf("rules[%d] (%s): when.attr_uint_lt_ratio.against must be validator_power", i, rule.ID)
			}
		}

		for j, f := range rule.Embed.Fields {
			if strings.TrimSpace(f.Attr) == "" || strings.TrimSpace(f.Name) == "" {
				return fmt.Errorf("rules[%d] (%s): embed.fields[%d] needs attr and name", i, rule.ID, j)
			}
			switch strings.ToLower(f.Format) {
			case "", "raw", "code", "amount_trb":
			default:
				return fmt.Errorf("rules[%d] (%s): unknown field format %q", i, rule.ID, f.Format)
			}
		}
		for _, e := range rule.Enrich {
			if e != "asset_pair" {
				return fmt.Errorf("rules[%d] (%s): unknown enrich %q", i, rule.ID, e)
			}
		}
		for _, se := range rule.SideEffects {
			if se.RecordAttrAsTimestamp == "" {
				return fmt.Errorf("rules[%d] (%s): side_effects entry needs record_attr_as_timestamp", i, rule.ID)
			}
			if c.State.ValsetTimestampsPath == "" {
				return fmt.Errorf("rules[%d] (%s): state.valset_timestamps_path required for side_effects", i, rule.ID)
			}
		}
		if rule.RateLimit != nil {
			if rule.RateLimit.Max < 0 || rule.RateLimit.Window < 0 || rule.RateLimit.Cooldown < 0 {
				return fmt.Errorf("rules[%d] (%s): rate_limit values must be non-negative", i, rule.ID)
			}
		}
	}
	return nil
}

// ParseDailyAt parses "HH:MM" into hour and minute.
func ParseDailyAt(s string) (hour, min int, err error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("want HH:MM, got %q", s)
	}
	h, err := parseIntRange(parts[0], 0, 23)
	if err != nil {
		return 0, 0, fmt.Errorf("hour: %w", err)
	}
	m, err := parseIntRange(parts[1], 0, 59)
	if err != nil {
		return 0, 0, fmt.Errorf("minute: %w", err)
	}
	return h, m, nil
}

func parseIntRange(s string, lo, hi int) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number %q", s)
		}
		n = n*10 + int(c-'0')
	}
	if n < lo || n > hi {
		return 0, fmt.Errorf("%d out of range [%d,%d]", n, lo, hi)
	}
	return n, nil
}

// EventTypes returns event types referenced by event rules (plus watch_events).
func (c *Config) EventTypes() map[string]struct{} {
	set := make(map[string]struct{})
	for _, r := range c.Rules {
		kind := strings.ToLower(strings.TrimSpace(r.Match.Kind))
		if (kind == KindEvent || kind == "event") && r.Match.EventType != "" {
			set[r.Match.EventType] = struct{}{}
		}
	}
	for _, e := range c.WatchEvents {
		e = strings.TrimSpace(e)
		if e != "" {
			set[e] = struct{}{}
		}
	}
	return set
}

// RateLimitFor returns the effective rate limit for a rule.
func (c *Config) RateLimitFor(rule RuleConfig) RateLimitConfig {
	if rule.RateLimit != nil {
		rl := *rule.RateLimit
		if rl.Max == 0 {
			rl.Max = c.Defaults.RateLimit.Max
		}
		if rl.Window == 0 {
			rl.Window = c.Defaults.RateLimit.Window
		}
		if rl.Cooldown == 0 {
			rl.Cooldown = c.Defaults.RateLimit.Cooldown
		}
		return rl
	}
	return c.Defaults.RateLimit
}

// IsEventRule reports whether the rule matches chain events.
func (r RuleConfig) IsEventRule() bool {
	kind := strings.ToLower(strings.TrimSpace(r.Match.Kind))
	return kind == KindEvent || kind == "event"
}

func normalizeRPCURL(u string) string {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	if strings.Contains(u, "localhost") || strings.HasPrefix(u, "127.0.0.1") {
		return "http://" + u
	}
	return "https://" + u
}
