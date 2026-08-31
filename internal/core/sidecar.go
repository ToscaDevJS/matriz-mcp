package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SidecarPath returns the canonical sidecar file path for an image file.
func SidecarPath(imagePath string) string {
	if strings.HasSuffix(imagePath, ".meta.json") {
		return imagePath
	}
	return imagePath + ".meta.json"
}

// WriteSidecar writes metadata alongside the target image file.
func WriteSidecar(imagePath string, s *Sidecar) error {
	if s == nil {
		return fmt.Errorf("sidecar cannot be nil")
	}

	targetPath := SidecarPath(imagePath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for sidecar: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sidecar: %w", err)
	}

	return os.WriteFile(targetPath, data, 0644)
}

// ReadSidecar reads and validates the .meta.json sidecar for an image file.
func ReadSidecar(imagePath string) (*Sidecar, error) {
	targetPath := SidecarPath(imagePath)

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sidecar file: %w", err)
	}

	var s Sidecar
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sidecar JSON: %w", err)
	}

	if s.Schema == "" {
		return nil, fmt.Errorf("invalid sidecar: missing schema field")
	}

	if s.Schema != SidecarSchema {
		return nil, fmt.Errorf("unsupported sidecar schema version: %q", s.Schema)
	}

	return &s, nil
}
