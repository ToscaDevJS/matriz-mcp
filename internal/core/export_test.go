package core_test

import (
	"strings"
	"testing"

	"github.com/toscodevjs/matriz/internal/core"
)

// TestT05_ExportWeb_WidthsCappedAtOriginal verifies T-05:
// ExportWeb on a 900px wide original generates only widths [420, 768] and none > 900.
func TestT05_ExportWeb_WidthsCappedAtOriginal(t *testing.T) {
	tempDir := t.TempDir()
	src := generateTestPattern(900, 450)

	opts := core.ExportOptions{
		ProjectRoot: tempDir,
		AssetRef:    core.AssetRef("assets/hero.png"),
		Widths:      core.DefaultExportWidths, // [420, 768, 1024, 1440, 1920]
		Formats:     []core.ImageFormat{core.FormatWebP},
		Quality:     80,
	}

	res, err := core.ExportWeb(src, opts)
	if err != nil {
		t.Fatalf("ExportWeb failed: %v", err)
	}

	if len(res.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(res.Variants))
	}

	if res.Variants[0].Width != 420 || res.Variants[1].Width != 768 {
		t.Fatalf("expected widths [420, 768], got [%d, %d]", res.Variants[0].Width, res.Variants[1].Width)
	}
}

// TestT06_ExportWeb_SrcsetFormat verifies T-06:
// The returned srcset has format "<path> <n>w" separated by ", " and widths in ascending order.
func TestT06_ExportWeb_SrcsetFormat(t *testing.T) {
	tempDir := t.TempDir()
	src := generateTestPattern(1200, 600)

	opts := core.ExportOptions{
		ProjectRoot: tempDir,
		AssetRef:    core.AssetRef("assets/hero.png"),
		Widths:      core.DefaultExportWidths, // [420, 768, 1024] (< 1200)
		Formats:     []core.ImageFormat{core.FormatWebP},
		Quality:     80,
	}

	res, err := core.ExportWeb(src, opts)
	if err != nil {
		t.Fatalf("ExportWeb failed: %v", err)
	}

	expectedSrcset := "assets/hero-420w.webp 420w, assets/hero-768w.webp 768w, assets/hero-1024w.webp 1024w"
	if res.Srcset[core.FormatWebP] != expectedSrcset {
		t.Errorf("expected srcset %q, got %q", expectedSrcset, res.Srcset[core.FormatWebP])
	}

	// Verify all parts are in strictly ascending width order
	parts := strings.Split(res.Srcset[core.FormatWebP], ", ")
	for _, part := range parts {
		pieces := strings.Fields(part)
		if len(pieces) != 2 {
			t.Fatalf("malformed srcset entry %q", part)
		}
		if _, err := core.ResolveRef(tempDir, core.AssetRef(pieces[0])); err != nil {
			t.Errorf("invalid ref in srcset %q: %v", pieces[0], err)
		}
		if !strings.HasSuffix(pieces[1], "w") {
			t.Fatalf("missing 'w' descriptor in %q", pieces[1])
		}
	}
}
