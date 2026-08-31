package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/version"
)

type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

// CheckResult represents the outcome of an individual diagnostic check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Report collects all diagnostic results.
type Report struct {
	Version string        `json:"version"`
	Checks  []CheckResult `json:"checks"`
	Healthy bool          `json:"healthy"`
}

// Run executes the complete diagnostic probe suite.
func Run(ctx context.Context, cfg *config.Config) *Report {
	if cfg == nil {
		cfg = config.LoadFromEnv()
	}

	checks := []CheckResult{
		ProbeCodecs(ctx),
		ProbeConfig(ctx, cfg),
		ProbeAPIKey(ctx, cfg),
		ProbeProject(ctx, cfg.ProjectRoot),
		ProbeMCPClients(ctx),
	}

	healthy := true
	for _, c := range checks {
		if c.Status == StatusFail {
			healthy = false
			break
		}
	}

	return &Report{
		Version: version.GetInfo().Version,
		Checks:  checks,
		Healthy: healthy,
	}
}

// Format returns human-readable terminal output.
func (r *Report) Format() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("MATRIZ Doctor (%s) — Diagnostic Report\n\n", r.Version))

	for _, c := range r.Checks {
		var symbol string
		switch c.Status {
		case StatusPass:
			symbol = "[✓]"
		case StatusWarn:
			symbol = "[!]"
		case StatusFail:
			symbol = "[✗]"
		}

		b.WriteString(fmt.Sprintf("%s %s: %s\n", symbol, c.Name, c.Message))
		if c.Details != "" {
			b.WriteString(fmt.Sprintf("    Details: %s\n", c.Details))
		}
	}

	b.WriteString("\n")
	if r.Healthy {
		b.WriteString("Status: Everything is healthy and ready to use!\n")
	} else {
		b.WriteString("Status: Issues detected. Please review the failed probes above.\n")
	}

	return b.String()
}

// JSON returns machine-readable JSON representation.
func (r *Report) JSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
