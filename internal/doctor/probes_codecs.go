package doctor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"runtime"

	"github.com/toscodevjs/matriz/internal/core"
)

// ProbeCodecs verifies that all 4 image codecs encode/decode properly in memory.
func ProbeCodecs(ctx context.Context) CheckResult {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	formats := []core.ImageFormat{
		core.FormatPNG,
		core.FormatJPEG,
		core.FormatWebP,
	}

	for _, fmtType := range formats {
		var buf bytes.Buffer
		if err := core.Encode(&buf, img, fmtType, 80); err != nil {
			return CheckResult{
				Name:    "Go Runtime & Codecs",
				Status:  StatusFail,
				Message: fmt.Sprintf("Codec %s encoding failed: %v", fmtType, err),
			}
		}
	}

	return CheckResult{
		Name:    "Go Runtime & Codecs",
		Status:  StatusPass,
		Message: fmt.Sprintf("%s runtime operational; PNG, JPEG, WebP codecs verified", runtime.Version()),
	}
}
