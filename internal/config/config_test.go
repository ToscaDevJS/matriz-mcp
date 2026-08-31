package config_test

import (
	"os"
	"testing"

	"github.com/toscodevjs/matriz/internal/config"
)

func TestConfig_LoadDefaults(t *testing.T) {
	// Clear any overrides
	os.Unsetenv("MATRIZ_PROVIDER")
	os.Unsetenv("MATRIZ_BUDGET_USD")
	os.Unsetenv("MATRIZ_MAX_GENERATIVE_CALLS")

	cfg := config.LoadFromEnv()
	if cfg.Provider != "gemini" {
		t.Errorf("expected default provider gemini, got %q", cfg.Provider)
	}
	if cfg.BudgetUSD != 2.00 {
		t.Errorf("expected default budget 2.00, got %f", cfg.BudgetUSD)
	}
	if cfg.MaxGenerativeCalls != 20 {
		t.Errorf("expected default max calls 20, got %d", cfg.MaxGenerativeCalls)
	}
}

func TestConfig_LoadCustom(t *testing.T) {
	t.Setenv("MATRIZ_PROVIDER", "custom")
	t.Setenv("MATRIZ_BUDGET_USD", "5.50")
	t.Setenv("MATRIZ_MAX_GENERATIVE_CALLS", "42")
	t.Setenv("MATRIZ_DRAFT_MAX_EDGE", "1024")

	cfg := config.LoadFromEnv()
	if cfg.Provider != "custom" {
		t.Errorf("expected custom provider, got %q", cfg.Provider)
	}
	if cfg.BudgetUSD != 5.50 {
		t.Errorf("expected budget 5.50, got %f", cfg.BudgetUSD)
	}
	if cfg.MaxGenerativeCalls != 42 {
		t.Errorf("expected max calls 42, got %d", cfg.MaxGenerativeCalls)
	}
	if cfg.DraftMaxEdge != 1024 {
		t.Errorf("expected draft edge 1024, got %d", cfg.DraftMaxEdge)
	}
}
