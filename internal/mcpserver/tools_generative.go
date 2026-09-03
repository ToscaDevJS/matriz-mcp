package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/providers"
)

type GenerateDraftsIn struct {
	Prompt         string `json:"prompt" jsonschema:"what to generate"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Count          int    `json:"count" jsonschema:"number of drafts, 1..4"`
	AspectRatio    string `json:"aspect_ratio" jsonschema:"e.g. 16:9, 1:1, 21:9"`
	Slot           string `json:"slot,omitempty" jsonschema:"manifest slot id this image is for; fills dimensions automatically"`
	Seed           *int64 `json:"seed,omitempty" jsonschema:"omit for random; the response always reports the seed used"`
}

type GenerateDraftsOut struct {
	Drafts     []core.Asset `json:"drafts"`
	Seeds      []int64      `json:"seeds"`
	CostUSD    float64      `json:"cost_usd"`
	BudgetLeft float64      `json:"budget_left_usd"`
}

func handleGenerateDrafts(cfg *config.Config, reg *providers.Registry, guard *budget.Guard) mcp.ToolHandlerFor[GenerateDraftsIn, GenerateDraftsOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in GenerateDraftsIn) (*mcp.CallToolResult, GenerateDraftsOut, error) {
		provider, err := reg.Get(cfg.Provider)
		if err != nil {
			return toolError(err, "Check MATRIZ_PROVIDER configuration."), GenerateDraftsOut{}, nil
		}

		count := in.Count
		if count <= 0 {
			count = 1
		} else if count > 4 {
			count = 4
		}

		w, h := parseAspectRatio(in.AspectRatio, cfg.DraftMaxEdge)

		genReq := providers.GenerateRequest{
			Prompt:         in.Prompt,
			NegativePrompt: in.NegativePrompt,
			Width:          w,
			Height:         h,
			Count:          count,
			Seed:           in.Seed,
			Model:          cfg.ModelDraft,
		}

		// Pre-flight budget guard check (§3.2 & §5.6)
		estimatedCost := provider.EstimateCostUSD(genReq)
		if err := guard.Reserve(estimatedCost); err != nil {
			return toolError(err, "Deterministic operations via img_transform are free and do not require budget."), GenerateDraftsOut{}, nil
		}

		result, err := provider.Generate(ctx, genReq)
		if err != nil {
			return toolError(err, "Provider generation failed."), GenerateDraftsOut{}, nil
		}

		guard.Commit(result.CostUSD)

		draftsDir := filepath.Join(cfg.ProjectRoot, "assets", "drafts")
		_ = os.MkdirAll(draftsDir, 0755)

		var assets []core.Asset
		var seeds []int64
		var contentList []mcp.Content

		now := time.Now().UnixNano()

		for i, imgBytes := range result.Images {
			fileName := fmt.Sprintf("draft-%d-%d.png", now, i+1)
			relRef := core.AssetRef(filepath.Join("assets", "drafts", fileName))
			absPath, _ := core.ResolveRef(cfg.ProjectRoot, relRef)

			_ = os.WriteFile(absPath, imgBytes, 0644)

			// Decode before recording anything: both the sidecar and the Asset
			// handed back to the model must describe the image that was
			// produced, not the size that was requested.
			var dims core.Dimensions
			var thumb *mcp.ImageContent
			decoded, _, err := image.Decode(bytes.NewReader(imgBytes))
			if err == nil {
				dims = core.Dimensions{
					Width:  decoded.Bounds().Dx(),
					Height: decoded.Bounds().Dy(),
				}
				thumb, _ = thumbnailContent(decoded, 512)
			}

			sidecar := providers.BuildSidecar(relRef, provider.Name(), result, genReq, dims)
			_ = core.WriteSidecar(absPath, sidecar)

			if thumb != nil {
				contentList = append(contentList, thumb)
			}

			assets = append(assets, core.Asset{
				Ref:      relRef,
				Origin:   core.OriginGenerated,
				MIMEType: result.MIMEType,
				Dims:     dims,
				Bytes:    int64(len(imgBytes)),
			})
			seeds = append(seeds, result.Seed)
		}

		out := GenerateDraftsOut{
			Drafts:     assets,
			Seeds:      seeds,
			CostUSD:    result.CostUSD,
			BudgetLeft: guard.BudgetLeft(),
		}

		return &mcp.CallToolResult{
			Content: contentList,
		}, out, nil
	}
}

type RefineIn struct {
	Ref       string `json:"ref" jsonschema:"source asset reference to refine"`
	Operation string `json:"operation" jsonschema:"inpaint, outpaint, or remove_background"`
	Prompt    string `json:"prompt" jsonschema:"description of the desired edit"`
	Mask      string `json:"mask,omitempty" jsonschema:"optional mask asset reference for inpainting"`
	Output    string `json:"output" jsonschema:"project-relative path to write refined result"`
	Seed      *int64 `json:"seed,omitempty" jsonschema:"optional seed"`
}

type RefineOut struct {
	Asset      core.Asset `json:"asset"`
	CostUSD    float64    `json:"cost_usd"`
	BudgetLeft float64    `json:"budget_left_usd"`
}

func handleRefine(cfg *config.Config, reg *providers.Registry, guard *budget.Guard) mcp.ToolHandlerFor[RefineIn, RefineOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in RefineIn) (*mcp.CallToolResult, RefineOut, error) {
		provider, err := reg.Get(cfg.Provider)
		if err != nil {
			return toolError(err, "Check MATRIZ_PROVIDER configuration."), RefineOut{}, nil
		}

		srcRef := core.AssetRef(in.Ref)
		srcPath, err := core.ResolveRef(cfg.ProjectRoot, srcRef)
		if err != nil {
			return toolError(err, "Check input ref path."), RefineOut{}, nil
		}

		srcBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return toolError(err, "Source asset could not be read."), RefineOut{}, nil
		}

		var maskBytes []byte
		if in.Mask != "" {
			maskPath, err := core.ResolveRef(cfg.ProjectRoot, core.AssetRef(in.Mask))
			if err == nil {
				maskBytes, _ = os.ReadFile(maskPath)
			}
		}

		editReq := providers.EditRequest{
			Source:    srcBytes,
			Mask:      maskBytes,
			Prompt:    in.Prompt,
			Operation: providers.Capability(in.Operation),
			Seed:      in.Seed,
			Model:     cfg.ModelFinal,
		}

		estimatedCost := provider.EstimateCostUSD(providers.GenerateRequest{
			Model: cfg.ModelFinal,
			Count: 1,
		})

		if err := guard.Reserve(estimatedCost); err != nil {
			return toolError(err, "Deterministic operations via img_transform are free and do not require budget."), RefineOut{}, nil
		}

		result, err := provider.Edit(ctx, editReq)
		if err != nil {
			return toolError(err, "Provider edit failed."), RefineOut{}, nil
		}

		guard.Commit(result.CostUSD)

		// A provider may answer without image bytes. Indexing Images[0] here
		// would panic, and a panic in a stdio handler takes the server down
		// instead of reaching the model (hard rule 7.9).
		if len(result.Images) == 0 {
			return toolError(
				fmt.Errorf("provider %s returned no image for operation %q", provider.Name(), in.Operation),
				"The provider accepted the request but produced no image. Try a simpler prompt, or a different operation.",
			), RefineOut{}, nil
		}

		outRef := core.AssetRef(in.Output)
		if outRef == "" {
			outRef = core.AssetRef(fmt.Sprintf("assets/refined-%d.png", time.Now().UnixNano()))
		}
		outPath, err := core.ResolveRef(cfg.ProjectRoot, outRef)
		if err != nil {
			return toolError(err, "Invalid output path."), RefineOut{}, nil
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return toolError(err, "Failed to create the destination directory."), RefineOut{}, nil
		}
		if err := os.WriteFile(outPath, result.Images[0], 0644); err != nil {
			return toolError(err, "Failed to write the refined image."), RefineOut{}, nil
		}

		// Decode before recording anything: an undecodable result must not take
		// the handler down, and both the sidecar and the Asset describe the
		// image that was produced.
		var dims core.Dimensions
		var thumb *mcp.ImageContent
		if decoded, _, decErr := image.Decode(bytes.NewReader(result.Images[0])); decErr == nil {
			dims = core.Dimensions{
				Width:  decoded.Bounds().Dx(),
				Height: decoded.Bounds().Dy(),
			}
			thumb, _ = thumbnailContent(decoded, 512)
		}

		// §5.4: every produced file writes its sidecar next to it. Refinement is
		// the operation that spends money; without this its provenance -- the
		// provider, model, prompt, seed and cost -- was lost the moment it ran.
		sidecar := providers.BuildEditSidecar(outRef, provider.Name(), result, editReq, srcRef, dims)
		_ = core.WriteSidecar(outPath, sidecar)

		asset := core.Asset{
			Ref:      outRef,
			Origin:   core.OriginGenerated,
			MIMEType: result.MIMEType,
			Dims:     dims,
			Bytes:    int64(len(result.Images[0])),
		}

		var content []mcp.Content
		if thumb != nil {
			content = append(content, thumb)
		}

		return &mcp.CallToolResult{
				Content: content,
			}, RefineOut{
				Asset:      asset,
				CostUSD:    result.CostUSD,
				BudgetLeft: guard.BudgetLeft(),
			}, nil
	}
}

type UpscaleIn struct {
	Ref    string `json:"ref" jsonschema:"source draft asset reference to upscale"`
	Prompt string `json:"prompt,omitempty" jsonschema:"optional enhancement prompt or fine detail instructions"`
	Output string `json:"output" jsonschema:"project-relative path to write upscaled result"`
	Seed   *int64 `json:"seed,omitempty" jsonschema:"optional seed"`
}

type UpscaleOut struct {
	Asset      core.Asset `json:"asset"`
	CostUSD    float64    `json:"cost_usd"`
	BudgetLeft float64    `json:"budget_left_usd"`
}

func handleUpscale(cfg *config.Config, reg *providers.Registry, guard *budget.Guard) mcp.ToolHandlerFor[UpscaleIn, UpscaleOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in UpscaleIn) (*mcp.CallToolResult, UpscaleOut, error) {
		provider, err := reg.Get(cfg.Provider)
		if err != nil {
			return toolError(err, "Check MATRIZ_PROVIDER configuration."), UpscaleOut{}, nil
		}

		srcRef := core.AssetRef(in.Ref)
		srcPath, err := core.ResolveRef(cfg.ProjectRoot, srcRef)
		if err != nil {
			return toolError(err, "Check input ref path."), UpscaleOut{}, nil
		}

		srcBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return toolError(err, "Source asset could not be read."), UpscaleOut{}, nil
		}

		prompt := in.Prompt
		if prompt == "" {
			prompt = "Enhance image resolution and visual fidelity to pro quality, preserving composition and details."
		}

		editReq := providers.EditRequest{
			Source:    srcBytes,
			Prompt:    prompt,
			Operation: providers.CapabilityUpscale,
			Seed:      in.Seed,
			Model:     cfg.ModelFinal,
		}

		estimatedCost := provider.EstimateCostUSD(providers.GenerateRequest{
			Model: cfg.ModelFinal,
			Count: 1,
		})

		if err := guard.Reserve(estimatedCost); err != nil {
			return toolError(err, "Deterministic operations via img_transform are free and do not require budget."), UpscaleOut{}, nil
		}

		result, err := provider.Edit(ctx, editReq)
		if err != nil {
			return toolError(err, "Provider upscale failed."), UpscaleOut{}, nil
		}

		guard.Commit(result.CostUSD)

		if len(result.Images) == 0 {
			return toolError(
				fmt.Errorf("provider %s returned no image for upscale", provider.Name()),
				"The provider accepted the request but produced no image. Try a different prompt.",
			), UpscaleOut{}, nil
		}

		outRef := core.AssetRef(in.Output)
		if outRef == "" {
			outRef = core.AssetRef(fmt.Sprintf("assets/upscaled-%d.png", time.Now().UnixNano()))
		}
		outPath, err := core.ResolveRef(cfg.ProjectRoot, outRef)
		if err != nil {
			return toolError(err, "Invalid output path."), UpscaleOut{}, nil
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return toolError(err, "Failed to create the destination directory."), UpscaleOut{}, nil
		}
		if err := os.WriteFile(outPath, result.Images[0], 0644); err != nil {
			return toolError(err, "Failed to write the upscaled image."), UpscaleOut{}, nil
		}

		var dims core.Dimensions
		var thumb *mcp.ImageContent
		if decoded, _, decErr := image.Decode(bytes.NewReader(result.Images[0])); decErr == nil {
			dims = core.Dimensions{
				Width:  decoded.Bounds().Dx(),
				Height: decoded.Bounds().Dy(),
			}
			thumb, _ = thumbnailContent(decoded, 512)
		}

		sidecar := providers.BuildEditSidecar(outRef, provider.Name(), result, editReq, srcRef, dims)
		_ = core.WriteSidecar(outPath, sidecar)

		asset := core.Asset{
			Ref:      outRef,
			Origin:   core.OriginGenerated,
			MIMEType: result.MIMEType,
			Dims:     dims,
			Bytes:    int64(len(result.Images[0])),
		}

		var content []mcp.Content
		if thumb != nil {
			content = append(content, thumb)
		}

		return &mcp.CallToolResult{
			Content: content,
		}, UpscaleOut{
			Asset:      asset,
			CostUSD:    result.CostUSD,
			BudgetLeft: guard.BudgetLeft(),
		}, nil
	}
}

func parseAspectRatio(ratio string, maxEdge int) (int, int) {
	if maxEdge <= 0 {
		maxEdge = 768
	}
	switch ratio {
	case "16:9":
		return maxEdge, (maxEdge * 9) / 16
	case "21:9":
		return maxEdge, (maxEdge * 9) / 21
	case "4:3":
		return maxEdge, (maxEdge * 3) / 4
	case "1:1":
		return maxEdge, maxEdge
	case "9:16":
		return (maxEdge * 9) / 16, maxEdge
	default:
		return maxEdge, (maxEdge * 9) / 16
	}
}
