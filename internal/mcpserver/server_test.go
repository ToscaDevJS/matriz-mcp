package mcpserver_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/mcpserver"
	"github.com/toscodevjs/matriz/internal/providers"
	"github.com/toscodevjs/matriz/internal/providers/fake"
)

func setupTestServerWithContext(t *testing.T, customGuard *budget.Guard) (*mcp.Server, *fake.FakeProvider, string, *config.Config, *providers.Registry, *budget.Guard) {
	tempDir := t.TempDir()
	if eval, err := filepath.EvalSymlinks(tempDir); err == nil {
		tempDir = eval
	}

	fp := fake.NewFakeProvider()
	reg := providers.NewRegistry()
	reg.Register(fp)

	guard := customGuard
	if guard == nil {
		guard = budget.NewGuard(2.00, 20)
	}

	cfg := &config.Config{
		ProjectRoot:  tempDir,
		Provider:     "fake",
		ModelDraft:   "fake-draft",
		ModelFinal:   "fake-final",
		DraftMaxEdge: 768,
	}

	srv := mcpserver.NewServer(cfg, reg, guard)
	return srv, fp, tempDir, cfg, reg, guard
}

// createTestPNG writes a valid PNG fixture inside projectDir.
func createTestPNG(t *testing.T, projectDir, relPath string, w, h int) {
	absPath := filepath.Join(projectDir, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		t.Fatalf("failed to create dirs for test png: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 100, A: 255})
		}
	}

	f, err := os.Create(absPath)
	if err != nil {
		t.Fatalf("failed to create test png: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
}

// TestT11_Transform_MissingAsset_ReturnsIsError verifies T-11:
// img_transform with a non-existent ref returns CallToolResult.IsError == true, not a protocol error.
func TestT11_Transform_MissingAsset_ReturnsIsError(t *testing.T) {
	_, _, tempDir, cfg, _, _ := setupTestServerWithContext(t, nil)
	ctx := context.Background()

	res, err := mcpserver.CallTransform(ctx, cfg, mcpserver.TransformIn{
		Ref:    "assets/non_existent.png",
		Output: "assets/out.png",
	})

	if err != nil {
		t.Fatalf("tool handler returned Go protocol error instead of CallToolResult error: %v", err)
	}

	if res == nil || !res.IsError {
		t.Fatalf("expected CallToolResult.IsError == true for missing asset")
	}

	// Must have actionable error text in content
	if len(res.Content) == 0 {
		t.Fatalf("expected error content in CallToolResult")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "does not exist") {
		t.Errorf("expected actionable error message, got %v", res.Content[0])
	}
	_ = tempDir
}

// TestT12_And_T13_ThumbnailGeneratedAndSized verifies T-12 and T-13:
// Every image-producing tool returns an ImageContent in Content with max edge <= 512px.
func TestT12_And_T13_ThumbnailGeneratedAndSized(t *testing.T) {
	_, _, tempDir, cfg, _, _ := setupTestServerWithContext(t, nil)
	createTestPNG(t, tempDir, "assets/source.png", 1000, 600)
	ctx := context.Background()

	// 1. Test img_transform
	resTransform, err := mcpserver.CallTransform(ctx, cfg, mcpserver.TransformIn{
		Ref:    "assets/source.png",
		Width:  400,
		Output: "assets/derived.png",
	})
	if err != nil || resTransform.IsError {
		t.Fatalf("transform failed: %v", err)
	}

	var foundThumb bool
	for _, c := range resTransform.Content {
		if imgContent, ok := c.(*mcp.ImageContent); ok {
			foundThumb = true
			if imgContent.MIMEType != "image/png" {
				t.Errorf("expected thumbnail mime type image/png, got %s", imgContent.MIMEType)
			}
			cfgImg, _, err := image.DecodeConfig(bytes.NewReader(imgContent.Data))
			if err != nil {
				t.Fatalf("failed to decode thumbnail config: %v", err)
			}
			if cfgImg.Width > 512 || cfgImg.Height > 512 {
				t.Fatalf("thumbnail exceeds 512px max edge: %dx%d", cfgImg.Width, cfgImg.Height)
			}
		}
	}
	if !foundThumb {
		t.Fatalf("img_transform did not return an ImageContent thumbnail")
	}
}

// TestT14_GenerateDrafts_BudgetExhausted_NoProviderCall verifies T-14:
// img_generate_drafts with an exhausted guard does not call the provider (fake count stays 0).
func TestT14_GenerateDrafts_BudgetExhausted_NoProviderCall(t *testing.T) {
	exhaustedGuard := budget.NewGuard(0.00, 0) // Limit is $0, max calls 0
	_, fp, _, cfg, reg, guard := setupTestServerWithContext(t, exhaustedGuard)
	ctx := context.Background()

	res, err := mcpserver.CallGenerateDrafts(ctx, cfg, reg, guard, mcpserver.GenerateDraftsIn{
		Prompt: "luxury barber haircut",
		Count:  2,
	})

	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}

	if res == nil || !res.IsError {
		t.Fatalf("expected CallToolResult.IsError == true when budget exhausted")
	}

	if fp.InvocationCount() != 0 {
		t.Fatalf("expected 0 provider calls when budget is exhausted, got %d", fp.InvocationCount())
	}
}

// TestT15_ToolDescriptions_CostMarkers verifies T-15:
// Generative tools contain COSTS MONEY and deterministic tools contain FREE in descriptions.
func TestT15_ToolDescriptions_CostMarkers(t *testing.T) {
	tools := mcpserver.GetToolDefinitions()

	for _, tool := range tools {
		switch tool.Name {
		case "img_generate_drafts", "img_refine":
			if !strings.HasPrefix(tool.Description, "COSTS MONEY") {
				t.Errorf("tool %s description must start with COSTS MONEY, got %q", tool.Name, tool.Description)
			}
		case "img_transform":
			if !strings.HasPrefix(tool.Description, "FREE") {
				t.Errorf("tool %s description must start with FREE, got %q", tool.Name, tool.Description)
			}
		case "img_list_models":
			// Meta tool
			if !strings.HasPrefix(tool.Description, "FREE") && !strings.Contains(tool.Description, "List") {
				t.Errorf("meta tool description unexpected: %s", tool.Description)
			}
		}
	}
}

func TestTransform_RejectsAVIFOutput(t *testing.T) {
	_, _, tempDir, cfg, _, _ := setupTestServerWithContext(t, nil)
	createTestPNG(t, tempDir, "assets/source.png", 200, 200)
	ctx := context.Background()

	res, err := mcpserver.CallTransform(ctx, cfg, mcpserver.TransformIn{
		Ref:    "assets/source.png",
		Output: "assets/output.avif",
	})
	if err != nil {
		t.Fatalf("expected tool error not protocol error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected CallToolResult.IsError == true for .avif output")
	}
	if len(res.Content) == 0 {
		t.Fatalf("expected error content in result")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "AVIF encoding is disabled") {
		t.Errorf("expected error to explain AVIF is disabled, got: %v", res.Content[0])
	}
}
