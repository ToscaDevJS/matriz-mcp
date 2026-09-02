package mcpserver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/mcpserver"
	"github.com/toscodevjs/matriz/internal/providers"
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

// emptyResultProvider mirrors a provider that answers without any image bytes.
// Nothing in the Provider contract forbids it, and handleRefine indexed
// result.Images[0] unconditionally.
type emptyResultProvider struct{}

func (emptyResultProvider) Name() string { return "empty" }
func (emptyResultProvider) Capabilities() []providers.Capability {
	return []providers.Capability{providers.CapabilityRemoveBG}
}
func (emptyResultProvider) EstimateCostUSD(providers.GenerateRequest) float64 { return 0.01 }
func (emptyResultProvider) Generate(context.Context, providers.GenerateRequest) (*providers.Result, error) {
	return &providers.Result{Model: "empty-model"}, nil
}
func (emptyResultProvider) Edit(context.Context, providers.EditRequest) (*providers.Result, error) {
	return &providers.Result{Model: "empty-model"}, nil
}

// TestRefine_WritesSidecar covers §5.4 end to end. img_refine wrote its output
// file and no .meta.json at all, so the one operation that spends money left no
// record of the provider, model, prompt or cost behind it.
func TestRefine_WritesSidecar(t *testing.T) {
	_, fp, tempDir, cfg, reg, guard := setupTestServerWithContext(t, nil)
	fp.SetOutputSize(1408, 768)

	createTestPNG(t, tempDir, "assets/drafts/draft-1.png", 800, 600)

	res, err := mcpserver.CallRefine(context.Background(), cfg, reg, guard, mcpserver.RefineIn{
		Ref:       "assets/drafts/draft-1.png",
		Prompt:    "remove the background",
		Operation: "remove_background",
		Output:    "assets/refined/hero.png",
	})
	if err != nil {
		t.Fatalf("CallRefine returned an error: %v", err)
	}
	if res.IsError {
		t.Fatalf("CallRefine failed: %+v", res.Content)
	}

	sidecar, err := core.ReadSidecar(filepath.Join(tempDir, "assets/refined/hero.png"))
	if err != nil {
		t.Fatalf("reading the sidecar for a refined image: %v", err)
	}

	if sidecar.Origin != core.OriginGenerated {
		t.Errorf("origin = %q, want %q", sidecar.Origin, core.OriginGenerated)
	}
	if sidecar.DerivedFrom == nil || *sidecar.DerivedFrom != core.AssetRef("assets/drafts/draft-1.png") {
		t.Errorf("derived_from = %v, want the source asset", sidecar.DerivedFrom)
	}
	if sidecar.Prompt != "remove the background" {
		t.Errorf("prompt = %q, want the instruction given", sidecar.Prompt)
	}
	if sidecar.Provider == "" || sidecar.Model == "" {
		t.Errorf("provider = %q, model = %q; both must be recorded", sidecar.Provider, sidecar.Model)
	}
	if got := sidecar.Params["operation"]; got != "remove_background" {
		t.Errorf("params.operation = %v, want %q", got, "remove_background")
	}
	if got := sidecar.Params["width"]; got != float64(1408) {
		t.Errorf("params.width = %v, want 1408 (produced)", got)
	}
}

// TestRefine_EmptyProviderResultIsAToolError guards the crash path. A provider
// answering with no images made handleRefine index Images[0] and then call
// Bounds() on a nil image. A panic in a stdio tool handler takes the server
// down; hard rule 7.9 requires an actionable tool error instead.
func TestRefine_EmptyProviderResultIsAToolError(t *testing.T) {
	_, _, tempDir, cfg, reg, guard := setupTestServerWithContext(t, nil)

	reg.Register(emptyResultProvider{})
	cfg.Provider = "empty"

	createTestPNG(t, tempDir, "assets/drafts/draft-1.png", 800, 600)

	res, err := mcpserver.CallRefine(context.Background(), cfg, reg, guard, mcpserver.RefineIn{
		Ref:       "assets/drafts/draft-1.png",
		Prompt:    "remove the background",
		Operation: "remove_background",
		Output:    "assets/refined/hero.png",
	})
	if err != nil {
		t.Fatalf("CallRefine propagated a Go error instead of a tool error: %v", err)
	}
	if !res.IsError {
		t.Fatal("want IsError=true when the provider returns no images")
	}
}
