package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// ProbeMCPClients discovers MCP configuration in standard client paths.
func ProbeMCPClients(ctx context.Context) CheckResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{
			Name:    "MCP Client Integration",
			Status:  StatusWarn,
			Message: "Could not determine user home directory",
		}
	}

	var detected []string

	// Check Claude Desktop config
	claudePath := filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	if data, err := os.ReadFile(claudePath); err == nil {
		if strings.Contains(string(data), "matriz") {
			detected = append(detected, "Claude Desktop (configured)")
		} else {
			detected = append(detected, "Claude Desktop (config present, matriz not added)")
		}
	}

	// Check Cursor config
	cursorPath := filepath.Join(home, ".cursor", "mcp.json")
	if data, err := os.ReadFile(cursorPath); err == nil {
		if strings.Contains(string(data), "matriz") {
			detected = append(detected, "Cursor (configured)")
		}
	}

	if len(detected) == 0 {
		return CheckResult{
			Name:    "MCP Client Integration",
			Status:  StatusWarn,
			Message: "No MCP clients detected with Matriz configuration. Register binary in claude_desktop_config.json or .cursor/mcp.json.",
		}
	}

	return CheckResult{
		Name:    "MCP Client Integration",
		Status:  StatusPass,
		Message: strings.Join(detected, ", "),
	}
}
