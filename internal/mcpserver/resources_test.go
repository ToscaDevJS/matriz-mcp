package mcpserver_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/manifest"
	"github.com/toscodevjs/matriz/internal/mcpserver"
)

// TestT18_ManifestResource_ValidatesSchema verifies T-18:
// The resource handler for matriz://project/manifest returns valid JSON against schema matriz.manifest/v1.
func TestT18_ManifestResource_ValidatesSchema(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "matriz.json")

	initialManifest := &manifest.Manifest{
		Schema:  manifest.ManifestSchema,
		Project: "barber-test",
		Palette: []string{"#000000", "#ffffff"},
		Slots: []manifest.Slot{
			{
				ID:          "hero",
				Usage:       "landing page main hero",
				AspectRatio: "21:9",
				MinWidth:    1920,
				SizesHint:   "100vw",
				Asset:       "assets/hero.avif",
				Alt:         "Modern hair salon hero",
			},
		},
		Assets: []manifest.Asset{
			{
				Ref:      "assets/hero.avif",
				Origin:   "generated",
				MIMEType: "image/avif",
				Dims:     manifest.Dimensions{Width: 1920, Height: 823},
				Bytes:    12345,
			},
		},
	}

	data, err := json.MarshalIndent(initialManifest, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal initial manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		t.Fatalf("failed to write matriz.json: %v", err)
	}

	cfg := &config.Config{
		ProjectRoot: tempDir,
	}

	resText, err := mcpserver.ReadManifestResource(context.Background(), cfg, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "matriz://project/manifest",
		},
	})
	if err != nil {
		t.Fatalf("ReadManifestResource failed: %v", err)
	}

	var parsed manifest.Manifest
	if err := json.Unmarshal([]byte(resText), &parsed); err != nil {
		t.Fatalf("resource output failed JSON parse: %v", err)
	}

	if parsed.Schema != manifest.ManifestSchema {
		t.Errorf("expected schema %q, got %q", manifest.ManifestSchema, parsed.Schema)
	}
	if parsed.Project != "barber-test" {
		t.Errorf("expected project barber-test, got %q", parsed.Project)
	}
	if len(parsed.Slots) != 1 || parsed.Slots[0].ID != "hero" {
		t.Errorf("slots did not parse correctly")
	}
}
