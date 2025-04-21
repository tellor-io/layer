package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func calculateFileChecksum(filename string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, return empty checksum
		return "", nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type ModuleStateWriter struct {
	encoder       *json.Encoder
	file          *os.File
	first         bool
	tempFilename  string
	finalFilename string
}

func NewModuleStateWriter(filename string) (*ModuleStateWriter, error) {
	// Create a temporary file first
	tempFile := filename + ".temp"

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(tempFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Create or truncate the temporary file
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write the opening structure
	if _, err := file.Write([]byte("{\n")); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write opening structure: %w", err)
	}

	return &ModuleStateWriter{
		encoder:       json.NewEncoder(file),
		file:          file,
		first:         true,
		tempFilename:  tempFile,
		finalFilename: filename,
	}, nil
}

// For array fields
func (w *ModuleStateWriter) StartArraySection(name string, afterItem bool) error {
	if !w.first {
		if afterItem {
			if _, err := w.file.Write([]byte("\n")); err != nil {
				return err
			}
		} else {
			if _, err := w.file.Write([]byte(",\n")); err != nil {
				return err
			}
		}
	}
	w.first = false

	// Write the field name and opening bracket with proper formatting
	_, err := w.file.Write([]byte(fmt.Sprintf("    \"%s\": [", name)))
	return err
}

func (w *ModuleStateWriter) WriteArrayItem(item interface{}) error {
	// Add newline and indentation for array items
	if _, err := w.file.Write([]byte("\n        ")); err != nil {
		return err
	}

	// Encode the item
	if err := w.encoder.Encode(item); err != nil {
		return err
	}

	// Remove the newline that Encode added
	if _, err := w.file.Seek(-1, io.SeekCurrent); err != nil {
		return err
	}

	if _, err := w.file.Write([]byte(",")); err != nil {
		return err
	}

	return nil
}

func (w *ModuleStateWriter) EndArraySection(numItems int) error {
	// Move back one character to remove the trailing comma
	if numItems > 0 {
		if _, err := w.file.Seek(-1, io.SeekCurrent); err != nil {
			return err
		}
	}
	// Add newline before closing bracket
	if numItems == 0 {
		_, err := w.file.Write([]byte("]"))
		return err
	} else {
		_, err := w.file.Write([]byte("\n    ]"))
		return err
	}
}

// For single value fields
func (w *ModuleStateWriter) WriteValue(name string, value interface{}) error {
	if !w.first {
		if _, err := w.file.Write([]byte(",\n")); err != nil {
			return err
		}
	}
	w.first = false

	// Write the field name with proper indentation
	if _, err := w.file.Write([]byte(fmt.Sprintf("    \"%s\": ", name))); err != nil {
		return err
	}

	// Encode the value
	w.encoder.Encode(value)

	// Remove the newline that Encode added
	if _, err := w.file.Seek(-1, io.SeekCurrent); err != nil {
		return err
	}

	if _, err := w.file.Write([]byte(",")); err != nil {
		return err
	}

	return nil
}

func (w *ModuleStateWriter) Close() {

	w.file.Write([]byte("\n}"))
	// Only close the file if it hasn't been closed yet
	if w.file != nil {
		// Flush any buffered data to disk
		if err := w.file.Sync(); err != nil {
			panic(err)
		}
		// Close the file
		if err := w.file.Close(); err != nil {
			panic(err)
		}
		w.file = nil
	}

	// Calculate checksum of the temporary file
	checksum, err := calculateFileChecksum(w.tempFilename)
	if err != nil {
		panic(err)
	}

	// Read the entire temporary file
	content, err := os.ReadFile(w.tempFilename)
	if err != nil {
		panic(err)
	}

	// Create or truncate the final file
	finalFile, err := os.OpenFile(w.finalFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer finalFile.Close()

	// Remove the final closing brace from the content
	content = content[:len(content)-2]

	// Write the original content without the final brace
	if _, err := finalFile.Write(content); err != nil {
		panic(err)
	}

	// Add the checksum and close the JSON object
	if _, err := finalFile.Write([]byte(fmt.Sprintf(",\n    \"checksum\": \"%s\"\n}", checksum))); err != nil {
		panic(err)
	}

	// Remove the temporary file
	if err := os.Remove(w.tempFilename); err != nil {
		panic(err)
	}
}
