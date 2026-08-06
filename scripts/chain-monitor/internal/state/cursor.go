package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// CursorFile persists the last successfully processed block height.
type CursorFile struct {
	path string
	mu   sync.Mutex
}

type cursorData struct {
	Height uint64 `json:"height"`
}

// NewCursorFile creates a cursor store at path.
func NewCursorFile(path string) *CursorFile {
	return &CursorFile{path: path}
}

// Path returns the cursor file path.
func (c *CursorFile) Path() string {
	return c.path
}

// Load returns the stored height and whether a cursor file existed.
func (c *CursorFile) Load() (height uint64, exists bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("read cursor: %w", err)
	}

	var cd cursorData
	if err := json.Unmarshal(data, &cd); err != nil {
		return 0, false, fmt.Errorf("parse cursor: %w", err)
	}
	return cd.Height, true, nil
}

// Save writes the last processed height atomically.
func (c *CursorFile) Save(height uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("create cursor dir: %w", err)
	}

	payload, err := json.MarshalIndent(cursorData{Height: height}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cursor: %w", err)
	}
	payload = append(payload, '\n')

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return fmt.Errorf("write cursor tmp: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename cursor: %w", err)
	}
	return nil
}
