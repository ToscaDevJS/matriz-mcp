package manifest

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/toscodevjs/matriz/internal/core"
)

// ScanProject inspects the project's assets directory and builds an updated Manifest.
func ScanProject(projectRoot, projectName string) (*Manifest, error) {
	manifest, err := ReadManifest(projectRoot)
	if err != nil {
		if projectName == "" {
			projectName = filepath.Base(projectRoot)
		}
		manifest = &Manifest{
			Schema:  ManifestSchema,
			Project: projectName,
			Slots:   make([]Slot, 0),
			Assets:  make([]Asset, 0),
		}
	}

	assetsDir := filepath.Join(projectRoot, "assets")
	if _, err := os.Stat(assetsDir); os.IsNotExist(err) {
		return manifest, nil
	}

	var scannedAssets []Asset

	err = filepath.WalkDir(assetsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".meta.json") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		format, err := core.ParseFormat(ext)
		if err != nil {
			return nil // Skip non-image files
		}

		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil
		}

		origin := core.OriginClient
		sidecar, err := core.ReadSidecar(path)
		if err == nil && sidecar.Origin != "" {
			origin = sidecar.Origin
		}

		width, height := 0, 0
		var duration float64

		if format == core.FormatMP4 || format == core.FormatWebM {
			if err == nil && sidecar != nil {
				if w, ok := sidecar.Params["width"].(float64); ok {
					width = int(w)
				} else if w, ok := sidecar.Params["width"].(int); ok {
					width = w
				}
				if h, ok := sidecar.Params["height"].(float64); ok {
					height = int(h)
				} else if h, ok := sidecar.Params["height"].(int); ok {
					height = h
				}
				if dur, ok := sidecar.Params["duration_sec"].(float64); ok {
					duration = dur
				}
			}
		} else {
			f, err := os.Open(path)
			if err == nil {
				cfg, _, err := image.DecodeConfig(f)
				if err == nil {
					width = cfg.Width
					height = cfg.Height
				}
				f.Close()
			}
		}

		fi, _ := d.Info()
		var fileBytes int64
		if fi != nil {
			fileBytes = fi.Size()
		}

		scannedAssets = append(scannedAssets, Asset{
			Ref:      core.AssetRef(relPath),
			Origin:   origin,
			MIMEType: format.MIMEType(),
			Dims: core.Dimensions{
				Width:  width,
				Height: height,
			},
			Bytes:    fileBytes,
			Duration: duration,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	manifest.Assets = scannedAssets
	return manifest, nil
}
