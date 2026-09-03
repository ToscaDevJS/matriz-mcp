package mcpserver

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/core"
)

type TransformIn struct {
	Ref        string        `json:"ref" jsonschema:"project-relative path of the source asset"`
	Crop       *core.CropBox `json:"crop,omitempty" jsonschema:"optional crop box in pixels"`
	Width      int           `json:"width,omitempty" jsonschema:"target width in px; 0 keeps source width"`
	Height     int           `json:"height,omitempty" jsonschema:"target height in px; 0 preserves aspect ratio"`
	Brightness float64       `json:"brightness,omitempty" jsonschema:"-100..100, 0 is no change"`
	Contrast   float64       `json:"contrast,omitempty" jsonschema:"-100..100, 0 is no change"`
	Saturation float64       `json:"saturation,omitempty" jsonschema:"-100..100, 0 is no change"`
	Rotate     float64       `json:"rotate,omitempty" jsonschema:"rotation angle in degrees clockwise"`
	Sharpen    float64       `json:"sharpen,omitempty" jsonschema:"sharpen sigma factor; 0 is no sharpening"`
	Output     string        `json:"output" jsonschema:"project-relative path to write the result"`
}

type TransformOut struct {
	Asset      core.Asset `json:"asset"`
	ThumbnailW int        `json:"thumbnail_width"`
	Note       string     `json:"note" jsonschema:"human-readable summary of what was applied"`
}

func handleTransform(cfg *config.Config) mcp.ToolHandlerFor[TransformIn, TransformOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in TransformIn) (*mcp.CallToolResult, TransformOut, error) {
		srcRef := core.AssetRef(in.Ref)
		srcPath, err := core.ResolveRef(cfg.ProjectRoot, srcRef)
		if err != nil {
			return toolError(err, "Verify that the ref path stays inside the project root."), TransformOut{}, nil
		}

		f, err := os.Open(srcPath)
		if err != nil {
			return toolError(fmt.Errorf("source asset %q does not exist: %w", in.Ref, core.ErrAssetNotFound), "Call matriz://project/manifest to list existing assets."), TransformOut{}, nil
		}
		defer f.Close()

		srcImg, formatName, err := image.Decode(f)
		if err != nil {
			return toolError(fmt.Errorf("failed to decode image %s: %w", in.Ref, err), "Make sure the file is a valid PNG, JPEG, or WebP image."), TransformOut{}, nil
		}

		resultImg := srcImg
		var operations []string

		if in.Crop != nil {
			resultImg = core.Crop(resultImg, *in.Crop)
			operations = append(operations, fmt.Sprintf("crop(%dx%d at %d,%d)", in.Crop.Width, in.Crop.Height, in.Crop.X, in.Crop.Y))
		}

		if in.Width > 0 || in.Height > 0 {
			resultImg = core.Resize(resultImg, in.Width, in.Height)
			operations = append(operations, fmt.Sprintf("resize(%dx%d)", resultImg.Bounds().Dx(), resultImg.Bounds().Dy()))
		}

		if in.Brightness != 0 || in.Contrast != 0 || in.Saturation != 0 {
			resultImg = core.Adjust(resultImg, &core.AdjustOptions{
				Brightness: in.Brightness,
				Contrast:   in.Contrast,
				Saturation: in.Saturation,
			})
			operations = append(operations, fmt.Sprintf("adjust(B:%.1f,C:%.1f,S:%.1f)", in.Brightness, in.Contrast, in.Saturation))
		}

		if in.Rotate != 0 {
			resultImg = core.Rotate(resultImg, in.Rotate)
			operations = append(operations, fmt.Sprintf("rotate(%.1f°)", in.Rotate))
		}

		if in.Sharpen > 0 {
			resultImg = core.Sharpen(resultImg, in.Sharpen)
			operations = append(operations, fmt.Sprintf("sharpen(%.1f)", in.Sharpen))
		}

		outRefStr := in.Output
		if outRefStr == "" {
			outRefStr = in.Ref
		}
		outRef := core.AssetRef(outRefStr)
		outPath, err := core.ResolveRef(cfg.ProjectRoot, outRef)
		if err != nil {
			return toolError(err, "Verify that output path stays inside the project root."), TransformOut{}, nil
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return toolError(err, "Failed to create destination directory."), TransformOut{}, nil
		}

		outFormat, err := core.ParseFormat(filepath.Ext(outPath))
		if err != nil {
			if filepath.Ext(outPath) != "" {
				return toolError(err, "Supported output formats are .webp, .png, and .jpg / .jpeg."), TransformOut{}, nil
			}
			outFormat, _ = core.ParseFormat(formatName)
			if outFormat == "" {
				outFormat = core.FormatPNG
			}
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return toolError(err, "Failed to create output file."), TransformOut{}, nil
		}
		if err := core.Encode(outFile, resultImg, outFormat, 85); err != nil {
			outFile.Close()
			return toolError(err, "Failed to encode output image."), TransformOut{}, nil
		}
		outFile.Close()

		fi, _ := os.Stat(outPath)
		var fileBytes int64
		if fi != nil {
			fileBytes = fi.Size()
		}

		// Write sidecar
		note := strings.Join(operations, ", ")
		if note == "" {
			note = "format conversion or copy"
		}
		sidecar := &core.Sidecar{
			Schema:      core.SidecarSchema,
			Ref:         outRef,
			Origin:      core.OriginDerived,
			CreatedAt:   time.Now().UTC(),
			DerivedFrom: &srcRef,
			Params: map[string]any{
				"operation": note,
				"width":     resultImg.Bounds().Dx(),
				"height":    resultImg.Bounds().Dy(),
			},
		}
		_ = core.WriteSidecar(outPath, sidecar)

		// Build thumbnail preview (512px max edge)
		thumb, err := thumbnailContent(resultImg, 512)
		if err != nil {
			return toolError(err, "Failed to generate thumbnail."), TransformOut{}, nil
		}

		asset := core.Asset{
			Ref:      outRef,
			Origin:   core.OriginDerived,
			MIMEType: outFormat.MIMEType(),
			Dims: core.Dimensions{
				Width:  resultImg.Bounds().Dx(),
				Height: resultImg.Bounds().Dy(),
			},
			Bytes: fileBytes,
		}

		out := TransformOut{
			Asset:      asset,
			ThumbnailW: len(thumb.Data),
			Note:       note,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{thumb},
		}, out, nil
	}
}
