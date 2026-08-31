package core

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultExportWidths contains the standard responsive image widths.
var DefaultExportWidths = []int{420, 768, 1024, 1440, 1920}

// ExportOptions configures responsive image generation.
type ExportOptions struct {
	ProjectRoot string
	AssetRef    AssetRef
	Widths      []int
	Formats     []ImageFormat
	Quality     int
	SizesHint   string
}

// ExportVariant represents one generated responsive image file.
type ExportVariant struct {
	Ref      AssetRef    `json:"ref"`
	Format   ImageFormat `json:"format"`
	Width    int         `json:"width"`
	Height   int         `json:"height"`
	Bytes    int64       `json:"bytes"`
	FilePath string      `json:"file_path"`
}

// ExportResult contains all generated variants and corresponding srcsets.
type ExportResult struct {
	Variants  []ExportVariant            `json:"variants"`
	Srcset    map[ImageFormat]string     `json:"srcset"`
	SizesHint string                     `json:"sizes_hint,omitempty"`
}

// ExportWeb scales and encodes an image into responsive variants according to §5.10.
func ExportWeb(src image.Image, opts ExportOptions) (*ExportResult, error) {
	if len(opts.Widths) == 0 {
		opts.Widths = DefaultExportWidths
	}
	if len(opts.Formats) == 0 {
		opts.Formats = []ImageFormat{FormatAVIF, FormatWebP}
	}
	if opts.Quality <= 0 {
		opts.Quality = 80
	}

	srcBounds := src.Bounds()
	origWidth := srcBounds.Dx()
	origHeight := srcBounds.Dy()

	// Filter widths that do not exceed original width and sort ascending
	var targetWidths []int
	for _, w := range opts.Widths {
		if w <= origWidth {
			targetWidths = append(targetWidths, w)
		}
	}
	sort.Ints(targetWidths)

	if len(targetWidths) == 0 {
		// If original is smaller than all default widths, use original width
		targetWidths = []int{origWidth}
	}

	refStr := string(opts.AssetRef)
	dir := filepath.Dir(refStr)
	ext := filepath.Ext(refStr)
	base := strings.TrimSuffix(filepath.Base(refStr), ext)

	res := &ExportResult{
		Variants:  make([]ExportVariant, 0),
		Srcset:    make(map[ImageFormat]string),
		SizesHint: opts.SizesHint,
	}

	now := time.Now().UTC()

	for _, format := range opts.Formats {
		var srcsetEntries []string

		for _, w := range targetWidths {
			scaled := src
			targetHeight := origHeight
			if w != origWidth {
				scaled = Resize(src, w, 0)
				targetHeight = scaled.Bounds().Dy()
			}

			variantFileName := fmt.Sprintf("%s-%dw%s", base, w, format.Extension())
			variantRelPath := filepath.Join(dir, variantFileName)
			if dir == "." {
				variantRelPath = variantFileName
			}
			variantRef := AssetRef(variantRelPath)

			var absPath string
			if opts.ProjectRoot != "" {
				var err error
				absPath, err = ResolveRef(opts.ProjectRoot, variantRef)
				if err != nil {
					return nil, err
				}

				if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
					return nil, fmt.Errorf("failed to create directory for variant: %w", err)
				}

				outFile, err := os.Create(absPath)
				if err != nil {
					return nil, fmt.Errorf("failed to create variant file: %w", err)
				}

				if err := Encode(outFile, scaled, format, opts.Quality); err != nil {
					outFile.Close()
					return nil, fmt.Errorf("failed to encode variant %s: %w", variantRef, err)
				}
				outFile.Close()

				fi, _ := os.Stat(absPath)
				var fileBytes int64
				if fi != nil {
					fileBytes = fi.Size()
				}

				// Write sidecar for derived variant
				sidecar := &Sidecar{
					Schema:      SidecarSchema,
					Ref:         variantRef,
					Origin:      OriginDerived,
					CreatedAt:   now,
					DerivedFrom: &opts.AssetRef,
					Params: map[string]any{
						"operation": "export_web",
						"width":     w,
						"height":    targetHeight,
						"format":    string(format),
						"quality":   opts.Quality,
					},
				}
				_ = WriteSidecar(absPath, sidecar)

				res.Variants = append(res.Variants, ExportVariant{
					Ref:      variantRef,
					Format:   format,
					Width:    w,
					Height:   targetHeight,
					Bytes:    fileBytes,
					FilePath: absPath,
				})
			} else {
				var buf bytes.Buffer
				if err := Encode(&buf, scaled, format, opts.Quality); err != nil {
					return nil, err
				}
				res.Variants = append(res.Variants, ExportVariant{
					Ref:    variantRef,
					Format: format,
					Width:  w,
					Height: targetHeight,
					Bytes:  int64(buf.Len()),
				})
			}

			srcsetEntries = append(srcsetEntries, fmt.Sprintf("%s %dw", variantRelPath, w))
		}

		res.Srcset[format] = strings.Join(srcsetEntries, ", ")
	}

	return res, nil
}
