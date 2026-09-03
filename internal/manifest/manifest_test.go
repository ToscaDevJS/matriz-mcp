package manifest_test

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/manifest"
)

// TestT16_Scan_ReconstructsInventory verifies T-16:
// scan reconstructs the inventory from disk and matches the written manifest.
func TestT16_Scan_ReconstructsInventory(t *testing.T) {
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(assetsDir, 0755)

	// Create test image
	imgPath := filepath.Join(assetsDir, "hero.png")
	createPNG(t, imgPath, 800, 400)

	// Create test sidecar
	sidecar := &core.Sidecar{
		Schema:    core.SidecarSchema,
		Ref:       core.AssetRef("assets/hero.png"),
		Origin:    core.OriginGenerated,
		CreatedAt: time.Now().UTC(),
		Model:     "gemini-3.1-flash-lite-image",
	}
	_ = core.WriteSidecar(imgPath, sidecar)

	// Perform scan
	m, err := manifest.ScanProject(tempDir, "salon-demo")
	if err != nil {
		t.Fatalf("ScanProject failed: %v", err)
	}

	if len(m.Assets) != 1 {
		t.Fatalf("expected 1 asset in scanned manifest, got %d", len(m.Assets))
	}

	asset := m.Assets[0]
	if asset.Ref != "assets/hero.png" {
		t.Errorf("expected asset ref assets/hero.png, got %s", asset.Ref)
	}
	if asset.Origin != core.OriginGenerated {
		t.Errorf("expected origin generated, got %s", asset.Origin)
	}
	if asset.Dims.Width != 800 || asset.Dims.Height != 400 {
		t.Errorf("expected dims 800x400, got %dx%d", asset.Dims.Width, asset.Dims.Height)
	}
}

// TestT17_Manifest_MissingOrigin_FailsValidation verifies T-17:
// An asset without an origin field fails manifest validation.
func TestT17_Manifest_MissingOrigin_FailsValidation(t *testing.T) {
	m := &manifest.Manifest{
		Schema:  manifest.ManifestSchema,
		Project: "barber-shop",
		Assets: []core.Asset{
			{
				Ref:      "assets/chair.png",
				Origin:   "", // missing origin!
				MIMEType: "image/png",
				Dims:     core.Dimensions{Width: 500, Height: 500},
			},
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatalf("expected validation error when asset is missing origin")
	}
}

func createPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create png file: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
}

func TestScanProject_DiscoversVideoAssets(t *testing.T) {
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets", "videos")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets dir: %v", err)
	}

	videoPath := filepath.Join(assetsDir, "hero.mp4")
	if err := os.WriteFile(videoPath, []byte("dummy-mp4-data"), 0644); err != nil {
		t.Fatalf("failed to write dummy video: %v", err)
	}

	sidecar := &core.Sidecar{
		Schema:    core.SidecarSchema,
		Ref:       core.AssetRef("assets/videos/hero.mp4"),
		Origin:    core.OriginGenerated,
		CreatedAt: time.Now().UTC(),
		Params: map[string]any{
			"width":        1920,
			"height":       1080,
			"duration_sec": 6.5,
		},
	}
	if err := core.WriteSidecar(videoPath, sidecar); err != nil {
		t.Fatalf("failed to write sidecar: %v", err)
	}

	m, err := manifest.ScanProject(tempDir, "test-project")
	if err != nil {
		t.Fatalf("ScanProject failed: %v", err)
	}

	if len(m.Assets) != 1 {
		t.Fatalf("expected 1 asset in manifest, got %d", len(m.Assets))
	}

	asset := m.Assets[0]
	if asset.Ref != "assets/videos/hero.mp4" {
		t.Errorf("expected ref assets/videos/hero.mp4, got %s", asset.Ref)
	}
	if asset.MIMEType != "video/mp4" {
		t.Errorf("expected MIMEType video/mp4, got %s", asset.MIMEType)
	}
	if asset.Dims.Width != 1920 || asset.Dims.Height != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", asset.Dims.Width, asset.Dims.Height)
	}
	if asset.Duration != 6.5 {
		t.Errorf("expected duration 6.5, got %f", asset.Duration)
	}
}
