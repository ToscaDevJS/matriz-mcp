package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/manifest"
)

const ManifestURI = "matriz://project/manifest"

// RegisterResources attaches project resources to the MCP server (§5.14).
func RegisterResources(srv *mcp.Server, cfg *config.Config) {
	srv.AddResource(&mcp.Resource{
		URI:         ManifestURI,
		Name:        "project manifest",
		Description: "Slots, palette and full asset inventory of the current site. Read this BEFORE generating anything.",
		MIMEType:    "application/json",
	}, ManifestResourceHandler(cfg))
}

// ManifestResourceHandler returns a handler that serves the JSON project manifest.
func ManifestResourceHandler(cfg *config.Config) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		text, err := ReadManifestResource(ctx, cfg, req)
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      ManifestURI,
					MIMEType: "application/json",
					Text:     text,
				},
			},
		}, nil
	}
}

// ReadManifestResource reads and scans the manifest and returns JSON text.
func ReadManifestResource(ctx context.Context, cfg *config.Config, req *mcp.ReadResourceRequest) (string, error) {
	m, err := manifest.ScanProject(cfg.ProjectRoot, "")
	if err != nil {
		return "", fmt.Errorf("failed to scan project manifest: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format manifest JSON: %w", err)
	}

	return string(data), nil
}
