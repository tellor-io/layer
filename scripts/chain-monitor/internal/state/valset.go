package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ValsetStore appends bridge validator-set timestamps and supports lookback analysis.
type ValsetStore struct {
	path string
	mu   sync.Mutex
}

type valsetFile struct {
	Timestamps []int64 `json:"timestamps"` // unix seconds
}

// NewValsetStore creates a store at path. Empty path disables persistence.
func NewValsetStore(path string) *ValsetStore {
	return &ValsetStore{path: path}
}

// Enabled reports whether a path is configured.
func (v *ValsetStore) Enabled() bool {
	return v != nil && v.path != ""
}

// Append records a timestamp string (unix seconds, unix ms, or RFC3339).
func (v *ValsetStore) Append(raw string) error {
	if !v.Enabled() {
		return nil
	}
	ts, err := parseTimestamp(raw)
	if err != nil {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := v.loadUnlocked()
	if err != nil {
		return err
	}
	unix := ts.Unix()
	for _, existing := range data.Timestamps {
		if existing == unix {
			return nil
		}
	}
	data.Timestamps = append(data.Timestamps, unix)
	sort.Slice(data.Timestamps, func(i, j int) bool {
		return data.Timestamps[i] < data.Timestamps[j]
	})
	return v.saveUnlocked(data)
}

// Recent returns timestamps within lookback of now, sorted ascending.
func (v *ValsetStore) Recent(lookback time.Duration) ([]time.Time, error) {
	if !v.Enabled() {
		return nil, nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := v.loadUnlocked()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-lookback).Unix()
	var out []time.Time
	for _, sec := range data.Timestamps {
		if sec >= cutoff {
			out = append(out, time.Unix(sec, 0).UTC())
		}
	}
	return out, nil
}

func (v *ValsetStore) loadUnlocked() (*valsetFile, error) {
	raw, err := os.ReadFile(v.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &valsetFile{}, nil
		}
		return nil, fmt.Errorf("read valset file: %w", err)
	}
	var data valsetFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parse valset file: %w", err)
	}
	if data.Timestamps == nil {
		data.Timestamps = []int64{}
	}
	return &data, nil
}

func (v *ValsetStore) saveUnlocked(data *valsetFile) error {
	if err := os.MkdirAll(filepath.Dir(v.path), 0o755); err != nil {
		return fmt.Errorf("create valset dir: %w", err)
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	tmp := v.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, v.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func parseTimestamp(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognized timestamp %q", raw)
	}
	if n > 1_000_000_000_000 { // ms
		return time.UnixMilli(n).UTC(), nil
	}
	return time.Unix(n, 0).UTC(), nil
}

// AnalyzeFrequency computes average/median gaps for a sorted timestamp list.
func AnalyzeFrequency(timestamps []time.Time) (count int, avg, median time.Duration, latest time.Time) {
	count = len(timestamps)
	if count == 0 {
		return 0, 0, 0, time.Time{}
	}
	latest = timestamps[count-1]
	if count < 2 {
		return count, 0, 0, latest
	}
	diffs := make([]time.Duration, 0, count-1)
	var sum time.Duration
	for i := 1; i < count; i++ {
		d := timestamps[i].Sub(timestamps[i-1])
		diffs = append(diffs, d)
		sum += d
	}
	avg = sum / time.Duration(len(diffs))
	sort.Slice(diffs, func(i, j int) bool { return diffs[i] < diffs[j] })
	if len(diffs)%2 == 1 {
		median = diffs[len(diffs)/2]
	} else {
		median = (diffs[len(diffs)/2-1] + diffs[len(diffs)/2]) / 2
	}
	return count, avg, median, latest
}
