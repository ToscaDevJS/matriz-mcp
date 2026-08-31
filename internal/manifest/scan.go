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

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		cfg, _, err := image.DecodeConfig(f)
		width, height := 0, 0
		if err == nil {
			width = cfg.Width
			height = cfg.Height
		}

		fi, _ := d.Info()
		var fileBytes int64
		if fi != nil {
			fileBytes = fi.Size()
		}

		origin := core.OriginClient
		sidecar, err := core.ReadSidecar(path)
		if err == nil && sidecar.Origin != "" {
			origin = sidecar.Origin
		}

		scannedAssets = append(scannedAssets, Asset{
			Ref:      core.AssetRef(relPath),
			Origin:   origin,
			MIMEType: format.MIMEType(),
			Dims: core.Dimensions{
				Width:  width,
				Height: height,
			},
			Bytes: fileBytes,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	manifest.Assets = scannedAssets
	return manifest, nil
}
