package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toscodevjs/matriz/internal/manifest"
)

// ProbeProject validates project structure, matriz.json schema, and assets directory.
func ProbeProject(ctx context.Context, projectRoot string) CheckResult {
	if projectRoot == "" {
		projectRoot = "."
	}

	if _, err := os.Stat(projectRoot); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Project Structure",
			Status:  StatusFail,
			Message: fmt.Sprintf("Project root %q does not exist", projectRoot),
		}
	}

	manifestPath := filepath.Join(projectRoot, "matriz.json")
	m, err := manifest.ReadManifest(projectRoot)
	if err != nil {
		assetsDir := filepath.Join(projectRoot, "assets")
		if _, errAssets := os.Stat(assetsDir); os.IsNotExist(errAssets) {
			return CheckResult{
				Name:    "Project Structure",
				Status:  StatusWarn,
				Message: "No matriz.json or assets/ folder detected in project root. Run scanner or initialize project.",
			}
		}
		return CheckResult{
			Name:    "Project Structure",
			Status:  StatusWarn,
			Message: "assets/ folder exists but matriz.json is missing or unparsed.",
		}
	}

	assetsCount := len(m.Assets)
	slotsCount := len(m.Slots)

	return CheckResult{
		Name:    "Project Structure",
		Status:  StatusPass,
		Message: fmt.Sprintf("Valid matriz.json found (%s)", manifestPath),
		Details: fmt.Sprintf("Project: %s, Slots: %d, Assets: %d", m.Project, slotsCount, assetsCount),
	}
}
