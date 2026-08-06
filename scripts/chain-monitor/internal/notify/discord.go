package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Field is a Discord embed field.
type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// Embed is a Discord rich embed.
type Embed struct {
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Color       uint32  `json:"color,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"` // ISO8601
	Footer      *Footer `json:"footer,omitempty"`
}

// Footer is an optional embed footer.
type Footer struct {
	Text string `json:"text,omitempty"`
}

// Message is a Discord webhook payload.
type Message struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

// FormatMessage renders a Message as plain text for structured logs / journald.
// Intended so log consumers (and humans) see the full alert body without Discord.
func FormatMessage(msg Message) string {
	var b strings.Builder
	if c := strings.TrimSpace(msg.Content); c != "" {
		b.WriteString(c)
	}
	for _, emb := range msg.Embeds {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		if emb.Title != "" {
			b.WriteString(emb.Title)
			b.WriteString("\n")
		}
		if emb.Description != "" {
			b.WriteString(emb.Description)
			b.WriteString("\n")
		}
		for _, f := range emb.Fields {
			fmt.Fprintf(&b, "%s: %s\n", f.Name, f.Value)
		}
		if emb.Footer != nil && emb.Footer.Text != "" {
			b.WriteString(emb.Footer.Text)
			b.WriteString("\n")
		}
		if emb.Timestamp != "" {
			fmt.Fprintf(&b, "timestamp: %s\n", emb.Timestamp)
		}
	}
	return strings.TrimSpace(b.String())
}

// Sender delivers alerts to a named channel.
type Sender interface {
	Send(ctx context.Context, channel, webhookURL string, msg Message) error
}

// DiscordSender posts webhook payloads with retries.
type DiscordSender struct {
	client *http.Client
	log    *slog.Logger
}

// NewDiscordSender creates a Discord webhook sender.
func NewDiscordSender(timeout time.Duration, log *slog.Logger) *DiscordSender {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if log == nil {
		log = slog.Default()
	}
	return &DiscordSender{
		client: &http.Client{Timeout: timeout},
		log:    log,
	}
}

// Send posts the message to webhookURL.
func (d *DiscordSender) Send(ctx context.Context, channel, webhookURL string, msg Message) error {
	if webhookURL == "" {
		return fmt.Errorf("empty webhook url for channel %s", channel)
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := d.post(ctx, webhookURL, payload); err != nil {
			lastErr = err
			d.log.Warn("discord send failed", "channel", channel, "attempt", attempt, "err", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (d *DiscordSender) post(ctx context.Context, webhookURL string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	// Discord rate limit
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("discord rate limited: %s", string(body))
	}
	return fmt.Errorf("discord status %d: %s", resp.StatusCode, string(body))
}

// DryRunSender logs alerts instead of sending them.
type DryRunSender struct {
	log *slog.Logger
}

// NewDryRunSender creates a logging-only sender.
func NewDryRunSender(log *slog.Logger) *DryRunSender {
	if log == nil {
		log = slog.Default()
	}
	return &DryRunSender{log: log}
}

// Send is a no-op; the dispatcher already logs the full alert text.
func (d *DryRunSender) Send(_ context.Context, channel, _ string, _ Message) error {
	d.log.Debug("dry-run: discord send skipped", "channel", channel)
	return nil
}

// RateLimiter tracks per-rule alert windows and cooldowns.
type RateLimiter struct {
	mu    sync.Mutex
	state map[string]*ruleLimitState
}

type ruleLimitState struct {
	timestamps    []time.Time
	cooldownUntil time.Time
	max           int
	window        time.Duration
	cooldown      time.Duration
}

// NewRateLimiter creates an in-memory rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{state: make(map[string]*ruleLimitState)}
}

// Decision is the result of checking whether an alert may be sent.
type Decision struct {
	Allow          bool
	EnterCooldown  bool // true when this send hits the max and starts cooldown
	InCooldown     bool
	CooldownUntil  time.Time
}

// Configure ensures a rule has limit parameters.
func (r *RateLimiter) Configure(ruleID string, max int, window, cooldown time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.state[ruleID]
	if !ok {
		st = &ruleLimitState{}
		r.state[ruleID] = st
	}
	st.max = max
	st.window = window
	st.cooldown = cooldown
}

// Check returns whether a send is allowed (does not record yet).
func (r *RateLimiter) Check(ruleID string, now time.Time) Decision {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.ensure(ruleID)
	if now.Before(st.cooldownUntil) {
		return Decision{Allow: false, InCooldown: true, CooldownUntil: st.cooldownUntil}
	}
	st.timestamps = prune(st.timestamps, now.Add(-st.window))
	if st.max > 0 && len(st.timestamps) >= st.max {
		st.cooldownUntil = now.Add(st.cooldown)
		st.timestamps = nil
		return Decision{Allow: true, EnterCooldown: true, CooldownUntil: st.cooldownUntil}
	}
	return Decision{Allow: true}
}

// Record notes a successful send.
func (r *RateLimiter) Record(ruleID string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.ensure(ruleID)
	st.timestamps = append(prune(st.timestamps, now.Add(-st.window)), now)
}

// AllowWindow returns whether an event may proceed under max/window only (no
// cooldown). When allowed and max > 0, it records the event. max <= 0 means
// unlimited and always returns true without recording.
func (r *RateLimiter) AllowWindow(ruleID string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.ensure(ruleID)
	if st.max <= 0 {
		return true
	}
	st.timestamps = prune(st.timestamps, now.Add(-st.window))
	if len(st.timestamps) >= st.max {
		return false
	}
	st.timestamps = append(st.timestamps, now)
	return true
}

func (r *RateLimiter) ensure(ruleID string) *ruleLimitState {
	st, ok := r.state[ruleID]
	if !ok {
		st = &ruleLimitState{max: 10, window: 10 * time.Minute, cooldown: 2 * time.Hour}
		r.state[ruleID] = st
	}
	return st
}

func prune(ts []time.Time, cutoff time.Time) []time.Time {
	out := ts[:0]
	for _, t := range ts {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	return out
}

// Deduper suppresses duplicate alerts within a TTL.
type Deduper struct {
	mu   sync.Mutex
	ttl  time.Duration
	seen map[string]time.Time
}

// NewDeduper creates a deduper with the given TTL.
func NewDeduper(ttl time.Duration) *Deduper {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Deduper{ttl: ttl, seen: make(map[string]time.Time)}
}

// Has reports whether key was already recorded and is still within TTL.
func (d *Deduper) Has(key string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.gc(now)
	exp, ok := d.seen[key]
	return ok && now.Before(exp)
}

// Add records a key as seen until now+ttl.
func (d *Deduper) Add(key string, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen[key] = now.Add(d.ttl)
}

// SeenOrAdd returns true if the key was already seen (duplicate); otherwise records it.
func (d *Deduper) SeenOrAdd(key string, now time.Time) bool {
	if d.Has(key, now) {
		return true
	}
	d.Add(key, now)
	return false
}

func (d *Deduper) gc(now time.Time) {
	if len(d.seen) < 1000 {
		return
	}
	for k, exp := range d.seen {
		if !now.Before(exp) {
			delete(d.seen, k)
		}
	}
}
