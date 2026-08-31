package doctor_test

import (
	"context"
	"testing"

	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/doctor"
	"github.com/toscodevjs/matriz/internal/manifest"
)

func TestDoctor_ProbeCodecs(t *testing.T) {
	ctx := context.Background()
	res := doctor.ProbeCodecs(ctx)

	if res.Status != doctor.StatusPass {
		t.Fatalf("expected codec probe to pass, got status %s: %s", res.Status, res.Message)
	}
}

func TestDoctor_ProbeConfig(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Provider:           "fake",
		GoogleAPIKey:       "AIzaSyDummyKeyForTesting",
		ProjectRoot:        ".",
		ModelDraft:         "gemini-3.1-flash-lite-image",
		ModelFinal:         "gemini-3-pro-image-preview",
		BudgetUSD:          5.00,
		MaxGenerativeCalls: 20,
	}

	res := doctor.ProbeConfig(ctx, cfg)
	if res.Status != doctor.StatusPass {
		t.Fatalf("expected config probe to pass, got status %s: %s", res.Status, res.Message)
	}
}

func TestDoctor_ProbeProject(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Initial empty directory
	resEmpty := doctor.ProbeProject(ctx, tempDir)
	if resEmpty.Status != doctor.StatusWarn && resEmpty.Status != doctor.StatusPass {
		t.Errorf("expected warn/pass on empty dir, got %s", resEmpty.Status)
	}

	// Create valid manifest
	m := &manifest.Manifest{
		Schema:  manifest.ManifestSchema,
		Project: "doctor-test",
		Slots:   []manifest.Slot{},
		Assets:  []manifest.Asset{},
	}
	if err := manifest.WriteManifest(tempDir, m); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	resValid := doctor.ProbeProject(ctx, tempDir)
	if resValid.Status != doctor.StatusPass {
		t.Errorf("expected pass on valid manifest, got %s: %s", resValid.Status, resValid.Message)
	}
}

func TestDoctor_Run(t *testing.T) {
	ctx := context.Background()
	cfg := config.LoadFromEnv()

	report := doctor.Run(ctx, cfg)
	if len(report.Checks) != 5 {
		t.Fatalf("expected 5 checks in doctor report, got %d", len(report.Checks))
	}
	if report.Version == "" {
		t.Errorf("expected non-empty version in report")
	}

	output := report.Format()
	if output == "" {
		t.Errorf("formatted report is empty")
	}
}
