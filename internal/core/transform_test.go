package core_test

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/toscodevjs/matriz/internal/core"
)

var update = flag.Bool("update", false, "update golden test files")

// generateTestPattern creates a deterministic 1000x500 image for golden tests.
func generateTestPattern(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := uint8((x * 255) / w)
			g := uint8((y * 255) / h)
			b := uint8(((x + y) * 255) / (w + h))
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func compareOrUpdateGolden(t *testing.T, goldenName string, actualBytes []byte) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "golden", goldenName)

	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, actualBytes, 0644); err != nil {
			t.Fatalf("failed to write golden file %s: %v", goldenPath, err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s (run with -update to generate): %v", goldenPath, err)
	}

	if !bytes.Equal(expected, actualBytes) {
		t.Fatalf("output bytes do not match golden file %s", goldenPath)
	}
}

// TestT03_Resize_Golden verifies T-03: Resize of 1000x500 to width 400 matches golden file.
func TestT03_Resize_Golden(t *testing.T) {
	src := generateTestPattern(1000, 500)

	resized := core.Resize(src, 400, 0) // 0 preserves aspect ratio => 400x200
	if resized.Bounds().Dx() != 400 || resized.Bounds().Dy() != 200 {
		t.Fatalf("expected dimensions 400x200, got %dx%d", resized.Bounds().Dx(), resized.Bounds().Dy())
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		t.Fatalf("failed to encode resized image: %v", err)
	}

	compareOrUpdateGolden(t, "resize_1000x500_to_400w.png", buf.Bytes())
}

// TestT04_Adjust_Golden verifies T-04: Adjust with brightness +20 matches golden file.
func TestT04_Adjust_Golden(t *testing.T) {
	src := generateTestPattern(1000, 500)

	adjusted := core.Adjust(src, &core.AdjustOptions{
		Brightness: 20,
		Contrast:   0,
		Saturation: 0,
	})

	var buf bytes.Buffer
	if err := png.Encode(&buf, adjusted); err != nil {
		t.Fatalf("failed to encode adjusted image: %v", err)
	}

	compareOrUpdateGolden(t, "adjust_brightness_p20.png", buf.Bytes())
}

func TestTransform_CropRotateSharpen(t *testing.T) {
	src := generateTestPattern(100, 100)

	// Test Crop
	cropped := core.Crop(src, core.CropBox{X: 10, Y: 10, Width: 50, Height: 50})
	if cropped.Bounds().Dx() != 50 || cropped.Bounds().Dy() != 50 {
		t.Errorf("expected cropped size 50x50, got %dx%d", cropped.Bounds().Dx(), cropped.Bounds().Dy())
	}

	// Test Rotate
	rotated := core.Rotate(src, 90)
	if rotated.Bounds().Dx() != 100 || rotated.Bounds().Dy() != 100 {
		t.Errorf("expected rotated size 100x100, got %dx%d", rotated.Bounds().Dx(), rotated.Bounds().Dy())
	}

	// Test Sharpen
	sharpened := core.Sharpen(src, 1.5)
	if sharpened.Bounds().Dx() != 100 || sharpened.Bounds().Dy() != 100 {
		t.Errorf("expected sharpened size 100x100, got %dx%d", sharpened.Bounds().Dx(), sharpened.Bounds().Dy())
	}
}
