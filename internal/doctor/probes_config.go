package doctor

import (
	"context"
	"fmt"

	"github.com/toscodevjs/matriz/internal/config"
)

// ProbeConfig validates current active provider and budget ceilings.
func ProbeConfig(ctx context.Context, cfg *config.Config) CheckResult {
	if cfg.Provider == "" {
		return CheckResult{
			Name:    "Configuration & Budget",
			Status:  StatusWarn,
			Message: "MATRIZ_PROVIDER not set; defaulting to gemini",
		}
	}

	details := fmt.Sprintf("Provider: %s, Draft: %s, Final: %s, Budget: $%.2f, MaxCalls: %d",
		cfg.Provider, cfg.ModelDraft, cfg.ModelFinal, cfg.BudgetUSD, cfg.MaxGenerativeCalls)

	return CheckResult{
		Name:    "Configuration & Budget",
		Status:  StatusPass,
		Message: fmt.Sprintf("Active provider %q, budget ceiling $%.2f", cfg.Provider, cfg.BudgetUSD),
		Details: details,
	}
}

// ProbeAPIKey checks for API credentials and masks the output to avoid leaking secrets.
func ProbeAPIKey(ctx context.Context, cfg *config.Config) CheckResult {
	key := cfg.GoogleAPIKey
	if key == "" {
		if cfg.Provider == "fake" {
			return CheckResult{
				Name:    "Google API Key",
				Status:  StatusPass,
				Message: "Offline fake provider active; no API key required",
			}
		}
		return CheckResult{
			Name:    "Google API Key",
			Status:  StatusWarn,
			Message: "GOOGLE_API_KEY is not set. Generative operations will fail; deterministic transforms remain available.",
		}
	}

	masked := maskKey(key)
	return CheckResult{
		Name:    "Google API Key",
		Status:  StatusPass,
		Message: fmt.Sprintf("Key detected (%s)", masked),
	}
}

func maskKey(k string) string {
	if len(k) <= 8 {
		return "********"
	}
	prefix := k[:4]
	suffix := k[len(k)-4:]
	return fmt.Sprintf("%s...%s", prefix, suffix)
}
