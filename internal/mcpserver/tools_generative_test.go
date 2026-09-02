package mcpserver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/mcpserver"
)

// TestGenerateDrafts_ReportsProducedDimensions is the end-to-end guard for the
// reproducibility bug. Image providers do not honour a requested pixel size:
// Gemini answered a 768x432 request with a 1408x768 image. Both the Asset
// returned to the model and the sidecar written to disk must describe the file
// that actually exists, otherwise the model sizes a slot against a lie.
func TestGenerateDrafts_ReportsProducedDimensions(t *testing.T) {
	_, fp, tempDir, cfg, reg, guard := setupTestServerWithContext(t, nil)

	// Mirror a provider that ignores the requested size, as Gemini does.
	fp.SetOutputSize(1408, 768)

	res, err := mcpserver.CallGenerateDrafts(context.Background(), cfg, reg, guard, mcpserver.GenerateDraftsIn{
		Prompt:      "a barber salon interior",
		Count:       1,
		AspectRatio: "16:9", // resolves to 768x432 for pricing
	})
	if err != nil {
		t.Fatalf("CallGenerateDrafts returned an error: %v", err)
	}
	if res.IsError {
		t.Fatalf("CallGenerateDrafts failed: %+v", res.Content)
	}

	out, ok := res.StructuredContent.(mcpserver.GenerateDraftsOut)
	if !ok {
		t.Fatalf("StructuredContent is %T, want GenerateDraftsOut", res.StructuredContent)
	}
	if len(out.Drafts) != 1 {
		t.Fatalf("got %d drafts, want 1", len(out.Drafts))
	}

	draft := out.Drafts[0]
	if draft.Dims.Width != 1408 || draft.Dims.Height != 768 {
		t.Errorf("Asset.Dims = %dx%d, want 1408x768 (produced), not 768x432 (requested)",
			draft.Dims.Width, draft.Dims.Height)
	}

	sidecar, err := core.ReadSidecar(filepath.Join(tempDir, string(draft.Ref)))
	if err != nil {
		t.Fatalf("reading the sidecar: %v", err)
	}

	if got := sidecar.Params["width"]; got != float64(1408) {
		t.Errorf("sidecar params.width = %v, want 1408 (produced), not 768 (requested)", got)
	}
	if got := sidecar.Params["height"]; got != float64(768) {
		t.Errorf("sidecar params.height = %v, want 768 (produced), not 432 (requested)", got)
	}
}
