package core_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/toscodevjs/matriz/internal/core"
)

// TestT07_Sidecar_WriteAndRead verifies T-07:
// Writing and reading a sidecar preserves all fields; missing schema is an error.
func TestT07_Sidecar_WriteAndRead(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "hero-01.png")

	derivedFrom := core.AssetRef("assets/original.jpg")
	now := time.Now().UTC().Truncate(time.Second)

	originalSidecar := &core.Sidecar{
		Schema:         core.SidecarSchema,
		Ref:            core.AssetRef("assets/hero-01.png"),
		Origin:         core.OriginGenerated,
		CreatedAt:      now,
		Provider:       "gemini",
		Model:          "gemini-3.1-flash-lite-image",
		Prompt:         "modern hair salon interior, warm lighting",
		NegativePrompt: "blurry, low quality",
		Seed:           42,
		Params:         map[string]any{"width": float64(1920), "height": float64(1080)},
		CostUSD:        0.0336,
		DerivedFrom:    &derivedFrom,
	}

	// Write sidecar
	if err := core.WriteSidecar(imagePath, originalSidecar); err != nil {
		t.Fatalf("failed to write sidecar: %v", err)
	}

	// Read sidecar back
	readSidecar, err := core.ReadSidecar(imagePath)
	if err != nil {
		t.Fatalf("failed to read sidecar: %v", err)
	}

	if readSidecar.Schema != originalSidecar.Schema {
		t.Errorf("schema mismatch: expected %q, got %q", originalSidecar.Schema, readSidecar.Schema)
	}
	if readSidecar.Ref != originalSidecar.Ref {
		t.Errorf("ref mismatch: expected %q, got %q", originalSidecar.Ref, readSidecar.Ref)
	}
	if readSidecar.Origin != originalSidecar.Origin {
		t.Errorf("origin mismatch: expected %q, got %q", originalSidecar.Origin, readSidecar.Origin)
	}
	if readSidecar.Provider != originalSidecar.Provider {
		t.Errorf("provider mismatch: expected %q, got %q", originalSidecar.Provider, readSidecar.Provider)
	}
	if readSidecar.Model != originalSidecar.Model {
		t.Errorf("model mismatch: expected %q, got %q", originalSidecar.Model, readSidecar.Model)
	}
	if readSidecar.Prompt != originalSidecar.Prompt {
		t.Errorf("prompt mismatch: expected %q, got %q", originalSidecar.Prompt, readSidecar.Prompt)
	}
	if readSidecar.NegativePrompt != originalSidecar.NegativePrompt {
		t.Errorf("negative prompt mismatch: expected %q, got %q", originalSidecar.NegativePrompt, readSidecar.NegativePrompt)
	}
	if readSidecar.Seed != originalSidecar.Seed {
		t.Errorf("seed mismatch: expected %d, got %d", originalSidecar.Seed, readSidecar.Seed)
	}
	if readSidecar.CostUSD != originalSidecar.CostUSD {
		t.Errorf("cost mismatch: expected %f, got %f", originalSidecar.CostUSD, readSidecar.CostUSD)
	}
	if readSidecar.DerivedFrom == nil || *readSidecar.DerivedFrom != derivedFrom {
		t.Errorf("derived_from mismatch")
	}

	// Test invalid / missing schema
	invalidSidecar := *originalSidecar
	invalidSidecar.Schema = ""
	invalidPath := filepath.Join(tempDir, "invalid.png")
	if err := core.WriteSidecar(invalidPath, &invalidSidecar); err != nil {
		t.Fatalf("unexpected error on write: %v", err)
	}
	if _, err := core.ReadSidecar(invalidPath); err == nil {
		t.Fatalf("expected error reading sidecar with missing schema")
	}
}
