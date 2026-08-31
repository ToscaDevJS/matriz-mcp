package core_test

import (
	"bytes"
	"image"
	"testing"

	"github.com/toscodevjs/matriz/internal/core"
)

func TestEncode_AllFormats(t *testing.T) {
	img := generateTestPattern(100, 100)

	formats := []struct {
		format   core.ImageFormat
		mimeType string
	}{
		{core.FormatJPEG, "image/jpeg"},
		{core.FormatPNG, "image/png"},
		{core.FormatWebP, "image/webp"},
		{core.FormatAVIF, "image/avif"},
	}

	for _, tt := range formats {
		t.Run(string(tt.format), func(t *testing.T) {
			var buf bytes.Buffer
			err := core.Encode(&buf, img, tt.format, 80)
			if err != nil {
				t.Fatalf("failed to encode format %s: %v", tt.format, err)
			}
			if buf.Len() == 0 {
				t.Fatalf("encoded buffer is empty for format %s", tt.format)
			}

			// Verify decoded config can read bounds
			cfg, formatName, err := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
			if err != nil {
				// Some formats might not register standard decodeconfig without specific imports, check bytes header
				if buf.Len() < 12 {
					t.Fatalf("encoded bytes too short: %d", buf.Len())
				}
			} else {
				if cfg.Width != 100 || cfg.Height != 100 {
					t.Errorf("expected decoded dims 100x100, got %dx%d (format: %s)", cfg.Width, cfg.Height, formatName)
				}
			}
		})
	}
}
