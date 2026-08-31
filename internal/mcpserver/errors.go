package mcpserver

import (
	"errors"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/core"
)

// toolError maps internal domain and IO errors into actionable MCP tool error responses (§5.7).
// Tool errors are NEVER returned as Go protocol errors so the LLM can see them and self-correct.
func toolError(err error, fallbackSuggestion string) *mcp.CallToolResult {
	if err == nil {
		return nil
	}

	var msg string
	if errors.Is(err, core.ErrInvalidAssetRef) {
		msg = fmt.Sprintf("Invalid asset ref: %v.\nMake sure paths are project-relative and do not escape the project root.", err)
	} else if os.IsNotExist(err) || errors.Is(err, core.ErrAssetNotFound) {
		msg = fmt.Sprintf("Asset not found: %v.\nCall the matriz://project/manifest resource to see available assets.", err)
	} else if errors.Is(err, core.ErrAspectRatioMismatch) {
		msg = fmt.Sprintf("Aspect ratio mismatch: %v.\nUse img_transform with a crop box, or regenerate with the expected aspect_ratio.", err)
	} else if errors.Is(err, core.ErrProviderUnsupported) {
		msg = fmt.Sprintf("Provider does not support operation: %v.\nCheck supported capabilities with img_list_models or switch provider with MATRIZ_PROVIDER.", err)
	} else if errors.Is(err, core.ErrBudgetExhausted) {
		msg = fmt.Sprintf("%v", err)
	} else {
		if fallbackSuggestion != "" {
			msg = fmt.Sprintf("%v.\n%s", err, fallbackSuggestion)
		} else {
			msg = fmt.Sprintf("%v.\nCheck inputs and verify project configuration.", err)
		}
	}

	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: msg,
			},
		},
	}
}
