package core

import (
	"image"

	"github.com/disintegration/imaging"
)

// CropBox specifies a rectangular region in pixels.
type CropBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// AdjustOptions specifies color correction adjustments (-100 to 100).
type AdjustOptions struct {
	Brightness float64 `json:"brightness,omitempty"`
	Contrast   float64 `json:"contrast,omitempty"`
	Saturation float64 `json:"saturation,omitempty"`
}

// Crop extracts a sub-region from the source image.
func Crop(img image.Image, box CropBox) image.Image {
	rect := image.Rect(box.X, box.Y, box.X+box.Width, box.Y+box.Height)
	return imaging.Crop(img, rect)
}

// Resize scales the image. If width or height is 0, aspect ratio is preserved.
func Resize(img image.Image, width, height int) image.Image {
	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// Adjust applies brightness, contrast, and saturation adjustments in sequence.
// Input percentages are in range -100..100 where 0 is neutral.
func Adjust(img image.Image, opts *AdjustOptions) image.Image {
	if opts == nil {
		return img
	}

	result := img
	if opts.Brightness != 0 {
		result = imaging.AdjustBrightness(result, opts.Brightness)
	}
	if opts.Contrast != 0 {
		result = imaging.AdjustContrast(result, opts.Contrast)
	}
	if opts.Saturation != 0 {
		result = imaging.AdjustSaturation(result, opts.Saturation)
	}
	return result
}

// Rotate rotates the image clockwise by the specified degrees.
func Rotate(img image.Image, angle float64) image.Image {
	return imaging.Rotate(img, angle, image.Transparent)
}

// Sharpen applies an unsharp mask filter to sharpen the image.
func Sharpen(img image.Image, sigma float64) image.Image {
	return imaging.Sharpen(img, sigma)
}
