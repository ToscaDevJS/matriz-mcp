package core

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"github.com/gen2brain/avif"
	"github.com/gen2brain/webp"
)

// ImageFormat represents a supported image encoding format.
type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatJPG  ImageFormat = "jpg"
	FormatPNG  ImageFormat = "png"
	FormatWebP ImageFormat = "webp"
	FormatAVIF ImageFormat = "avif"
)

// ParseFormat normalizes a format string or file extension into an ImageFormat.
func ParseFormat(extOrFormat string) (ImageFormat, error) {
	clean := strings.ToLower(strings.TrimPrefix(extOrFormat, "."))
	switch clean {
	case "jpg", "jpeg":
		return FormatJPEG, nil
	case "png":
		return FormatPNG, nil
	case "webp":
		return FormatWebP, nil
	case "avif":
		return FormatAVIF, nil
	default:
		return "", fmt.Errorf("unsupported image format: %q", extOrFormat)
	}
}

// MIMEType returns the MIME type corresponding to the format.
func (f ImageFormat) MIMEType() string {
	switch f {
	case FormatJPEG, FormatJPG:
		return "image/jpeg"
	case FormatPNG:
		return "image/png"
	case FormatWebP:
		return "image/webp"
	case FormatAVIF:
		return "image/avif"
	default:
		return "application/octet-stream"
	}
}

// Extension returns standard extension with leading dot.
func (f ImageFormat) Extension() string {
	switch f {
	case FormatJPEG, FormatJPG:
		return ".jpg"
	case FormatPNG:
		return ".png"
	case FormatWebP:
		return ".webp"
	case FormatAVIF:
		return ".avif"
	default:
		return ""
	}
}

// Encode writes the image to the writer in the requested format with quality [1..100].
func Encode(w io.Writer, img image.Image, format ImageFormat, quality int) error {
	if quality <= 0 {
		quality = 80
	} else if quality > 100 {
		quality = 100
	}

	switch format {
	case FormatJPEG, FormatJPG:
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	case FormatPNG:
		return png.Encode(w, img)
	case FormatWebP:
		return webp.Encode(w, img, webp.Options{Quality: quality})
	case FormatAVIF:
		return avif.Encode(w, img, avif.Options{Quality: quality})
	default:
		return fmt.Errorf("unsupported encoding format: %q", format)
	}
}
