package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toscodevjs/matriz/internal/core"
)

// ManifestSchema is the schema identifier for website image manifests (§5.14).
const ManifestSchema = "matriz.manifest/v1"

// Slot defines a design placeholder for an image on the client website.
type Slot struct {
	ID          string `json:"id"`
	Usage       string `json:"usage"`
	AspectRatio string `json:"aspect_ratio"`
	MinWidth    int    `json:"min_width"`
	SizesHint   string `json:"sizes_hint,omitempty"`
	Asset       string `json:"asset,omitempty"`
	Alt         string `json:"alt,omitempty"`
}

// Dimensions re-exports core.Dimensions for JSON manifest compatibility.
type Dimensions = core.Dimensions

// Asset re-exports core.Asset for JSON manifest compatibility.
type Asset = core.Asset

// Manifest defines the full project manifest structure (§5.14).
type Manifest struct {
	Schema  string   `json:"schema"`
	Project string   `json:"project"`
	Palette []string `json:"palette,omitempty"`
	Slots   []Slot   `json:"slots"`
	Assets  []Asset  `json:"assets"`
}

// Validate checks that the manifest conforms to the specification.
func (m *Manifest) Validate() error {
	if m.Schema != ManifestSchema {
		return fmt.Errorf("invalid manifest schema: expected %q, got %q", ManifestSchema, m.Schema)
	}

	for i, a := range m.Assets {
		if a.Ref == "" {
			return fmt.Errorf("asset at index %d is missing ref", i)
		}
		if a.Origin == "" {
			return fmt.Errorf("asset %q is missing required origin field (client | generated | derived)", a.Ref)
		}
	}

	return nil
}

// ReadManifest loads and validates matriz.json from the project root.
func ReadManifest(projectRoot string) (*Manifest, error) {
	manifestPath := filepath.Join(projectRoot, "matriz.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read matriz.json: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse matriz.json: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &m, nil
}

// WriteManifest saves a validated Manifest to matriz.json in the project root.
func WriteManifest(projectRoot string, m *Manifest) error {
	if m == nil {
		return fmt.Errorf("manifest cannot be nil")
	}
	m.Schema = ManifestSchema

	if err := m.Validate(); err != nil {
		return fmt.Errorf("cannot write invalid manifest: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode manifest JSON: %w", err)
	}

	manifestPath := filepath.Join(projectRoot, "matriz.json")
	return os.WriteFile(manifestPath, data, 0644)
}
