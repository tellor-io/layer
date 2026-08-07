package enrich

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ReporterMap maps reporter bech32 addresses to human-readable monikers.
type ReporterMap struct {
	mu   sync.RWMutex
	path string
	log  *slog.Logger
	data map[string]string // lowercased address -> moniker
}

type reportersFileData struct {
	AddressToMonikerMap map[string]string `json:"addressToMonikerMap"`
}

// NewReporterMap loads the map from path. Empty path disables moniker lookup.
func NewReporterMap(path string, log *slog.Logger) (*ReporterMap, error) {
	if log == nil {
		log = slog.Default()
	}
	m := &ReporterMap{path: path, log: log, data: map[string]string{}}
	if path == "" {
		return m, nil
	}
	if err := m.Reload(); err != nil {
		return nil, err
	}
	return m, nil
}

// Reload re-reads the JSON file.
func (m *ReporterMap) Reload() error {
	if m.path == "" {
		return nil
	}
	raw, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("read reporters map: %w", err)
	}
	var data reportersFileData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parse reporters map: %w", err)
	}
	normalized := make(map[string]string, len(data.AddressToMonikerMap))
	for addr, moniker := range data.AddressToMonikerMap {
		addr = strings.TrimSpace(addr)
		moniker = strings.TrimSpace(moniker)
		if addr == "" || moniker == "" {
			continue
		}
		normalized[strings.ToLower(addr)] = moniker
	}
	m.mu.Lock()
	m.data = normalized
	m.mu.Unlock()
	m.log.Info("loaded reporters map", "path", m.path, "reporters", len(normalized))
	return nil
}

// Moniker returns the configured moniker for addr, or empty if unknown.
func (m *ReporterMap) Moniker(addr string) string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[strings.ToLower(strings.TrimSpace(addr))]
}

// Display returns moniker if known, otherwise the original address.
func (m *ReporterMap) Display(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if moniker := m.Moniker(addr); moniker != "" {
		return moniker
	}
	return addr
}

// StartReloader reloads the map on interval until stop is closed.
func (m *ReporterMap) StartReloader(stop <-chan struct{}, interval time.Duration) {
	if m == nil || m.path == "" || interval <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if err := m.Reload(); err != nil {
					m.log.Warn("reload reporters map failed", "err", err)
				}
			}
		}
	}()
}

// ParseImportantReporters parses a comma-separated IMPORTANT_REPORTERS env value.
// Addresses are trimmed, de-duplicated case-insensitively, and sorted stably.
func ParseImportantReporters(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	reporters := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		reporter := strings.TrimSpace(part)
		if reporter == "" {
			continue
		}
		key := strings.ToLower(reporter)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		reporters = append(reporters, reporter)
	}
	sort.Strings(reporters)
	if len(reporters) == 0 {
		return nil
	}
	return reporters
}

// MissingReporters returns display names (moniker when known) for important
// reporters that did not appear in submitted. Nil if none missing.
func MissingReporters(important, submitted []string, monikers *ReporterMap) []string {
	if len(important) == 0 {
		return nil
	}
	reported := make(map[string]struct{}, len(submitted))
	for _, r := range submitted {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		reported[strings.ToLower(r)] = struct{}{}
	}
	missing := make([]string, 0)
	for _, addr := range important {
		if _, ok := reported[strings.ToLower(addr)]; ok {
			continue
		}
		missing = append(missing, monikers.Display(addr))
	}
	if len(missing) == 0 {
		return nil
	}
	return missing
}

// FormatMissingReporters returns a comma-separated display list of important
// reporters that did not appear in submitted. Empty if none missing.
func FormatMissingReporters(important, submitted []string, monikers *ReporterMap) string {
	return strings.Join(MissingReporters(important, submitted, monikers), ", ")
}
