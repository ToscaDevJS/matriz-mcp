package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/providers"
	"github.com/toscodevjs/matriz/internal/version"
)

// ServerContext bundles dependencies for tool handlers.
type ServerContext struct {
	Config *config.Config
	Reg    *providers.Registry
	Guard  *budget.Guard
}

// Tool definitions with exact descriptions and hints (§5.5).
var (
	destrFalse     = false
	openWorldTrue  = true
	openWorldFalse = false

	ToolListModels = &mcp.Tool{
		Name:        "img_list_models",
		Description: "FREE. List active provider, model IDs, and current budget consumption.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &openWorldFalse,
			Title:         "List Models and Budget",
		},
	}

	ToolTransform = &mcp.Tool{
		Name: "img_transform",
		Description: "FREE and instant, no model call. Crop, resize, rotate, sharpen, " +
			"or adjust brightness/contrast/saturation of an existing asset. " +
			"Always prefer this over img_refine for anything a filter can do.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &destrFalse,
			IdempotentHint:  true,
			OpenWorldHint:   &openWorldFalse,
			Title:           "Transform Image",
		},
	}

	ToolGenerateDrafts = &mcp.Tool{
		Name: "img_generate_drafts",
		Description: "COSTS MONEY and takes seconds. Generates N new draft images from " +
			"a text prompt. Returns low-resolution previews you can look at. " +
			"Do NOT use for cropping, resizing, format conversion or colour " +
			"adjustment — use img_transform for those.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &destrFalse,
			IdempotentHint:  false,
			OpenWorldHint:   &openWorldTrue,
			Title:           "Generate Draft Images",
		},
	}

	ToolRefine = &mcp.Tool{
		Name: "img_refine",
		Description: "COSTS MONEY and takes seconds. Inpaint, outpaint or remove background " +
			"using generative AI models. Do NOT use for standard crops or filters — use img_transform for those.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &destrFalse,
			IdempotentHint:  false,
			OpenWorldHint:   &openWorldTrue,
			Title:           "Refine and Edit Image",
		},
	}

	ToolUpscale = &mcp.Tool{
		Name: "img_upscale",
		Description: "COSTS MONEY and takes seconds. Elevates a draft image to pro high-resolution quality " +
			"using the final generative model. Do NOT use for standard crops or filters — use img_transform for those.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &destrFalse,
			IdempotentHint:  false,
			OpenWorldHint:   &openWorldTrue,
			Title:           "Upscale Draft to Pro Quality",
		},
	}
)

// GetToolDefinitions returns all tool definitions for assertions and introspection.
func GetToolDefinitions() []*mcp.Tool {
	return []*mcp.Tool{
		ToolListModels,
		ToolTransform,
		ToolGenerateDrafts,
		ToolRefine,
		ToolUpscale,
	}
}

// NewServer builds and registers all MCP tools and resources on a new MCP server.
func NewServer(cfg *config.Config, reg *providers.Registry, guard *budget.Guard) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "matriz",
		Version: version.Version,
	}, nil)

	RegisterTools(srv, cfg, reg, guard)
	RegisterResources(srv, cfg)
	return srv
}

// RegisterTools registers all image management tools.
func RegisterTools(srv *mcp.Server, cfg *config.Config, reg *providers.Registry, guard *budget.Guard) {
	mcp.AddTool(srv, ToolListModels, handleListModels(cfg, reg, guard))
	mcp.AddTool(srv, ToolTransform, handleTransform(cfg))
	mcp.AddTool(srv, ToolGenerateDrafts, handleGenerateDrafts(cfg, reg, guard))
	mcp.AddTool(srv, ToolRefine, handleRefine(cfg, reg, guard))
	mcp.AddTool(srv, ToolUpscale, handleUpscale(cfg, reg, guard))
}

// CallTransform executes img_transform handler directly for testing.
func CallTransform(ctx context.Context, cfg *config.Config, in TransformIn) (*mcp.CallToolResult, error) {
	handler := handleTransform(cfg)
	res, out, err := handler(ctx, &mcp.CallToolRequest{}, in)
	if err != nil {
		return res, err
	}
	if res == nil {
		res = &mcp.CallToolResult{}
	}
	// The handler returns content and structured output separately; the real
	// SDK merges them. Attach it here too, otherwise callers can only assert on
	// the thumbnail and the tool's actual result is invisible.
	if res.StructuredContent == nil {
		res.StructuredContent = out
	}
	return res, nil
}

// CallGenerateDrafts executes img_generate_drafts handler directly for testing.
func CallGenerateDrafts(ctx context.Context, cfg *config.Config, reg *providers.Registry, guard *budget.Guard, in GenerateDraftsIn) (*mcp.CallToolResult, error) {
	handler := handleGenerateDrafts(cfg, reg, guard)
	res, out, err := handler(ctx, &mcp.CallToolRequest{}, in)
	if err != nil {
		return res, err
	}
	if res == nil {
		res = &mcp.CallToolResult{}
	}
	// The handler returns content and structured output separately; the real
	// SDK merges them. Attach it here too, otherwise callers can only assert on
	// the thumbnail and the tool's actual result is invisible.
	if res.StructuredContent == nil {
		res.StructuredContent = out
	}
	return res, nil
}

// CallRefine executes img_refine handler directly for testing.
func CallRefine(ctx context.Context, cfg *config.Config, reg *providers.Registry, guard *budget.Guard, in RefineIn) (*mcp.CallToolResult, error) {
	handler := handleRefine(cfg, reg, guard)
	res, out, err := handler(ctx, &mcp.CallToolRequest{}, in)
	if err != nil {
		return res, err
	}
	if res == nil {
		res = &mcp.CallToolResult{}
	}
	if res.StructuredContent == nil {
		res.StructuredContent = out
	}
	return res, nil
}

// CallUpscale executes img_upscale handler directly for testing.
func CallUpscale(ctx context.Context, cfg *config.Config, reg *providers.Registry, guard *budget.Guard, in UpscaleIn) (*mcp.CallToolResult, error) {
	handler := handleUpscale(cfg, reg, guard)
	res, out, err := handler(ctx, &mcp.CallToolRequest{}, in)
	if err != nil {
		return res, err
	}
	if res == nil {
		res = &mcp.CallToolResult{}
	}
	if res.StructuredContent == nil {
		res.StructuredContent = out
	}
	return res, nil
}

