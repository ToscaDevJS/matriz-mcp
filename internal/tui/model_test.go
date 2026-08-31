package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/manifest"
	"github.com/toscodevjs/matriz/internal/tui"
)

// TestT19_TUIModel_ConstructsFromManifest verifies T-19:
// The TUI model builds from a manifest fixture without touching disk outside testdata/.
func TestT19_TUIModel_ConstructsFromManifest(t *testing.T) {
	fixtureManifest := &manifest.Manifest{
		Schema:  manifest.ManifestSchema,
		Project: "tui-test-salon",
		Palette: []string{"#0b0b0b", "#c8a45c"},
		Slots: []manifest.Slot{
			{
				ID:          "hero",
				Usage:       "landing hero",
				AspectRatio: "21:9",
				MinWidth:    1920,
				Asset:       "assets/hero.png",
			},
		},
		Assets: []core.Asset{
			{
				Ref:      "assets/hero.png",
				Origin:   core.OriginGenerated,
				MIMEType: "image/png",
				Dims:     core.Dimensions{Width: 1920, Height: 823},
				Bytes:    45000,
			},
			{
				Ref:      "assets/chair.png",
				Origin:   core.OriginClient,
				MIMEType: "image/png",
				Dims:     core.Dimensions{Width: 800, Height: 800},
				Bytes:    22000,
			},
		},
	}

	m := tui.NewModel(fixtureManifest, "/workspace/test")

	if len(m.Assets()) != 2 {
		t.Fatalf("expected 2 assets in model, got %d", len(m.Assets()))
	}

	if m.SelectedIndex() != 0 {
		t.Fatalf("expected initial selected index 0, got %d", m.SelectedIndex())
	}

	// Test navigation down
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updatedModel.(tui.Model)

	if m.SelectedIndex() != 1 {
		t.Errorf("expected selected index 1 after KeyDown, got %d", m.SelectedIndex())
	}

	// Test view rendering
	viewStr := m.View()
	if viewStr == "" {
		t.Fatalf("model view produced empty string")
	}
}
