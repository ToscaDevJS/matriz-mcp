package preview

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/toscodevjs/matriz/internal/core"
)

// GeneratePreviewHTML builds a local standalone HTML page to visually inspect an asset (§3.3).
func GeneratePreviewHTML(projectRoot string, asset core.Asset, sidecar *core.Sidecar) (string, error) {
	tmpDir := os.TempDir()
	previewPath := filepath.Join(tmpDir, fmt.Sprintf("matriz-preview-%s.html", filepath.Base(string(asset.Ref))))

	absImgPath, _ := core.ResolveRef(projectRoot, asset.Ref)

	sidecarJSON := "{}"
	if sidecar != nil {
		if data, err := json.MarshalIndent(sidecar, "", "  "); err == nil {
			sidecarJSON = string(data)
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Matriz Preview — %s</title>
	<style>
		body {
			background: #121212;
			color: #e0e0e0;
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
			margin: 0;
			padding: 24px;
			display: flex;
			flex-direction: column;
			align-items: center;
		}
		.card {
			background: #1e1e1e;
			border-radius: 8px;
			padding: 20px;
			max-width: 960px;
			width: 100%%;
			box-shadow: 0 4px 12px rgba(0,0,0,0.5);
		}
		img {
			max-width: 100%%;
			height: auto;
			border-radius: 4px;
			border: 1px solid #333;
		}
		.meta {
			margin-top: 16px;
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 12px;
		}
		.meta-item {
			background: #282828;
			padding: 10px;
			border-radius: 4px;
		}
		pre {
			background: #0d0d0d;
			padding: 12px;
			border-radius: 4px;
			overflow-x: auto;
			font-size: 12px;
		}
	</style>
</head>
<body>
	<div class="card">
		<h2>%s</h2>
		<img src="file://%s" alt="%s">
		<div class="meta">
			<div class="meta-item"><strong>Dimensions:</strong> %dx%d</div>
			<div class="meta-item"><strong>Origin:</strong> %s</div>
			<div class="meta-item"><strong>MIME Type:</strong> %s</div>
			<div class="meta-item"><strong>Size:</strong> %.2f KB</div>
		</div>
		<h3>Sidecar Metadata</h3>
		<pre><code>%s</code></pre>
	</div>
</body>
</html>`, asset.Ref, asset.Ref, absImgPath, asset.Ref, asset.Dims.Width, asset.Dims.Height, asset.Origin, asset.MIMEType, float64(asset.Bytes)/1024.0, sidecarJSON)

	if err := os.WriteFile(previewPath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write preview HTML: %w", err)
	}

	return previewPath, nil
}

// OpenBrowser launches the default browser to view the HTML preview file.
func OpenBrowser(htmlPath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", htmlPath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", htmlPath)
	default:
		cmd = exec.Command("xdg-open", htmlPath)
	}
	return cmd.Start()
}
