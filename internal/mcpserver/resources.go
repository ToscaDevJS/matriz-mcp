package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/jobs"
	"github.com/toscodevjs/matriz/internal/manifest"
)

const (
	ManifestURI     = "matriz://project/manifest"
	JobsResourceURI = "matriz://jobs"
)

// RegisterResources attaches project resources to the MCP server (§5.14).
func RegisterResources(srv *mcp.Server, cfg *config.Config) {
	srv.AddResource(&mcp.Resource{
		URI:         ManifestURI,
		Name:        "project manifest",
		Description: "Slots, palette and full asset inventory of the current site. Read this BEFORE generating anything.",
		MIMEType:    "application/json",
	}, ManifestResourceHandler(cfg))
}

// RegisterResourcesWithEngine attaches manifest and active jobs resources to the MCP server.
func RegisterResourcesWithEngine(srv *mcp.Server, cfg *config.Config, engine *jobs.Engine) {
	RegisterResources(srv, cfg)

	if engine != nil {
		srv.AddResource(&mcp.Resource{
			URI:         JobsResourceURI,
			Name:        "video jobs",
			Description: "Active and recent asynchronous video generation jobs in the session.",
			MIMEType:    "application/json",
		}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			jobList := engine.ListJobs()
			data, err := json.MarshalIndent(jobList, "", "  ")
			if err != nil {
				return nil, err
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      JobsResourceURI,
						MIMEType: "application/json",
						Text:     string(data),
					},
				},
			}, nil
		})
	}
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
