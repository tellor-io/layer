package enrich

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// QueryIDMap maps query IDs / query data to human-readable asset pairs.
type QueryIDMap struct {
	mu   sync.RWMutex
	path string
	log  *slog.Logger
	data *fileData
}

type fileData struct {
	QueryIDToAssetPairMap   map[string]string `json:"queryIdToAssetPairMap"`
	QueryDataToAssetPairMap map[string]string `json:"queryDataToAssetPairMap"`
}

// NewQueryIDMap loads the map from path. Empty path disables enrichment.
func NewQueryIDMap(path string, log *slog.Logger) (*QueryIDMap, error) {
	if log == nil {
		log = slog.Default()
	}
	m := &QueryIDMap{path: path, log: log}
	if path == "" {
		return m, nil
	}
	if err := m.Reload(); err != nil {
		return nil, err
	}
	return m, nil
}

// Reload re-reads the JSON file.
func (m *QueryIDMap) Reload() error {
	if m.path == "" {
		return nil
	}
	raw, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("read query ids map: %w", err)
	}
	var data fileData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parse query ids map: %w", err)
	}
	if data.QueryIDToAssetPairMap == nil {
		data.QueryIDToAssetPairMap = map[string]string{}
	}
	if data.QueryDataToAssetPairMap == nil {
		data.QueryDataToAssetPairMap = map[string]string{}
	}
	m.mu.Lock()
	m.data = &data
	m.mu.Unlock()
	m.log.Info("loaded query ids map",
		"path", m.path,
		"query_ids", len(data.QueryIDToAssetPairMap),
		"query_data", len(data.QueryDataToAssetPairMap),
	)
	return nil
}

// AssetPair looks up an asset pair by query_id (hex, optional 0x prefix).
func (m *QueryIDMap) AssetPair(queryID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil {
		return ""
	}
	id := normalizeHex(queryID)
	if pair, ok := m.data.QueryIDToAssetPairMap[id]; ok {
		return pair
	}
	// Also try raw value in case map keys include 0x.
	if pair, ok := m.data.QueryIDToAssetPairMap[queryID]; ok {
		return pair
	}
	return ""
}

// AssetPairFromQueryData looks up by query_data hex.
func (m *QueryIDMap) AssetPairFromQueryData(queryData string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil {
		return ""
	}
	data := normalizeHex(queryData)
	if pair, ok := m.data.QueryDataToAssetPairMap[data]; ok {
		return pair
	}
	if pair, ok := m.data.QueryDataToAssetPairMap[queryData]; ok {
		return pair
	}
	return ""
}

// StartReloader reloads the map on interval until ctx is done.
func (m *QueryIDMap) StartReloader(stop <-chan struct{}, interval time.Duration) {
	if m.path == "" || interval <= 0 {
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
					m.log.Warn("reload query ids map failed", "err", err)
				}
			}
		}
	}()
}

func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return strings.ToLower(s)
}
