package mcpserver_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/jobs"
	"github.com/toscodevjs/matriz/internal/mcpserver"
	"github.com/toscodevjs/matriz/internal/providers"
	"github.com/toscodevjs/matriz/internal/providers/fake"
)

func setupVideoTestServer(t *testing.T, customGuard *budget.Guard) (*config.Config, *providers.Registry, *budget.Guard, *jobs.Engine) {
	tempDir := t.TempDir()
	if eval, err := filepath.EvalSymlinks(tempDir); err == nil {
		tempDir = eval
	}

	fp := fake.NewFakeProvider()
	reg := providers.NewRegistry()
	reg.Register(fp)

	guard := customGuard
	if guard == nil {
		guard = budget.NewGuard(5.00, 20)
	}

	cfg := &config.Config{
		ProjectRoot:     tempDir,
		Provider:        "fake",
		ModelDraft:      "fake-draft",
		ModelFinal:      "fake-final",
		ModelVideoDraft: "fake-veo-turbo",
		ModelVideoFinal: "fake-veo-pro",
		DraftMaxEdge:    768,
	}

	engine := jobs.NewEngine(tempDir, reg, guard)
	return cfg, reg, guard, engine
}

func TestVideoTools_ToolDefinitions(t *testing.T) {
	defs := mcpserver.GetToolDefinitions()

	var hasGen, hasStatus, hasCancel bool
	for _, tool := range defs {
		switch tool.Name {
		case "video_generate":
			hasGen = true
			if !strings.HasPrefix(tool.Description, "COSTS MONEY") {
				t.Errorf("video_generate description must start with COSTS MONEY, got: %s", tool.Description)
			}
		case "video_status":
			hasStatus = true
			if !strings.HasPrefix(tool.Description, "FREE") {
				t.Errorf("video_status description must start with FREE, got: %s", tool.Description)
			}
		case "video_cancel":
			hasCancel = true
			if !strings.HasPrefix(tool.Description, "FREE") {
				t.Errorf("video_cancel description must start with FREE, got: %s", tool.Description)
			}
		}
	}

	if !hasGen {
		t.Errorf("expected video_generate in tool definitions")
	}
	if !hasStatus {
		t.Errorf("expected video_status in tool definitions")
	}
	if !hasCancel {
		t.Errorf("expected video_cancel in tool definitions")
	}
}

func TestVideoGenerate_TextToVideo_EndToEnd(t *testing.T) {
	cfg, reg, guard, engine := setupVideoTestServer(t, nil)

	in := mcpserver.VideoGenerateIn{
		Prompt:      "cinematic drone shot over ocean at sunset",
		DurationSec: 5.0,
		AspectRatio: "16:9",
		ModelTier:   "draft",
	}

	res, err := mcpserver.CallVideoGenerate(context.Background(), cfg, reg, guard, engine, in)
	if err != nil {
		t.Fatalf("CallVideoGenerate failed: %v", err)
	}

	out, ok := res.StructuredContent.(mcpserver.VideoGenerateOut)
	if !ok {
		t.Fatalf("expected StructuredContent to be VideoGenerateOut, got %T", res.StructuredContent)
	}

	if out.JobID == "" {
		t.Fatalf("expected non-empty JobID")
	}
	if out.Status != "processing" {
		t.Errorf("expected status 'processing', got %q", out.Status)
	}

	// Budget hold was recorded
	if guard.ReservedUSD() == 0 {
		t.Errorf("expected guard to hold reservation for video job")
	}

	// Wait up to 3 seconds for background worker to complete the mock job
	var statusOut mcpserver.VideoStatusOut
	for i := 0; i < 15; i++ {
		time.Sleep(200 * time.Millisecond)
		statusRes, err := mcpserver.CallVideoStatus(context.Background(), cfg, engine, mcpserver.VideoStatusIn{
			JobID:       out.JobID,
			WaitSeconds: 1,
		})
		if err != nil {
			t.Fatalf("CallVideoStatus failed: %v", err)
		}
		statusOut = statusRes.StructuredContent.(mcpserver.VideoStatusOut)
		if statusOut.Status == "completed" {
			// Assert poster thumbnail content is present
			if len(statusRes.Content) == 0 {
				t.Errorf("expected thumbnail poster ImageContent in completed CallVideoStatus result")
			}
			break
		}
	}

	if statusOut.Status != "completed" {
		t.Fatalf("expected job to complete, got status %s", statusOut.Status)
	}
	if statusOut.Asset == nil || statusOut.Asset.Ref == "" {
		t.Fatalf("expected created Asset in output")
	}

	// Check sidecar
	sidecarPath := filepath.Join(cfg.ProjectRoot, string(statusOut.Asset.Ref)+".meta.json")
	sidecar, err := core.ReadSidecar(sidecarPath)
	if err != nil {
		t.Fatalf("failed to read created video sidecar: %v", err)
	}
	if sidecar.Origin != core.OriginGenerated {
		t.Errorf("expected origin 'generated', got %q", sidecar.Origin)
	}
}

func TestVideoGenerate_ImageToVideo_DerivedFrom(t *testing.T) {
	cfg, reg, guard, engine := setupVideoTestServer(t, nil)

	createTestPNG(t, cfg.ProjectRoot, "assets/hero.png", 512, 512)

	in := mcpserver.VideoGenerateIn{
		Ref:         "assets/hero.png",
		Prompt:      "animate gentle breathing and wind",
		DurationSec: 5.0,
		Output:      "assets/videos/hero-animated.mp4",
	}

	res, err := mcpserver.CallVideoGenerate(context.Background(), cfg, reg, guard, engine, in)
	if err != nil {
		t.Fatalf("CallVideoGenerate failed: %v", err)
	}

	out := res.StructuredContent.(mcpserver.VideoGenerateOut)

	// Wait for job completion
	var statusOut mcpserver.VideoStatusOut
	for i := 0; i < 15; i++ {
		time.Sleep(200 * time.Millisecond)
		statusRes, err := mcpserver.CallVideoStatus(context.Background(), cfg, engine, mcpserver.VideoStatusIn{
			JobID:       out.JobID,
			WaitSeconds: 1,
		})
		if err != nil {
			t.Fatalf("CallVideoStatus failed: %v", err)
		}
		statusOut = statusRes.StructuredContent.(mcpserver.VideoStatusOut)
		if statusOut.Status == "completed" {
			break
		}
	}

	if statusOut.Status != "completed" {
		t.Fatalf("expected job to complete, got %s", statusOut.Status)
	}

	// Verify sidecar preserves DerivedFrom
	sidecarPath := filepath.Join(cfg.ProjectRoot, "assets/videos/hero-animated.mp4.meta.json")
	sidecar, err := core.ReadSidecar(sidecarPath)
	if err != nil {
		t.Fatalf("failed to read sidecar: %v", err)
	}
	if sidecar.DerivedFrom == nil || *sidecar.DerivedFrom != "assets/hero.png" {
		t.Errorf("expected DerivedFrom='assets/hero.png', got %v", sidecar.DerivedFrom)
	}
}

func TestVideoCancel_ReleasesHold(t *testing.T) {
	cfg, reg, guard, engine := setupVideoTestServer(t, nil)

	in := mcpserver.VideoGenerateIn{
		Prompt:      "abstract waves",
		DurationSec: 5.0,
	}

	res, err := mcpserver.CallVideoGenerate(context.Background(), cfg, reg, guard, engine, in)
	if err != nil {
		t.Fatalf("CallVideoGenerate failed: %v", err)
	}

	out := res.StructuredContent.(mcpserver.VideoGenerateOut)

	cancelRes, err := mcpserver.CallVideoCancel(context.Background(), engine, mcpserver.VideoCancelIn{
		JobID: out.JobID,
	})
	if err != nil {
		t.Fatalf("CallVideoCancel failed: %v", err)
	}

	cancelOut := cancelRes.StructuredContent.(mcpserver.VideoCancelOut)
	if cancelOut.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got %s", cancelOut.Status)
	}

	if guard.ReservedUSD() != 0 {
		t.Errorf("expected budget hold to be 0 after cancellation, got %.2f", guard.ReservedUSD())
	}
}
