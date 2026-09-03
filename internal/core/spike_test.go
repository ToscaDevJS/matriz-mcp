package core_test

import (
	"image"
	"testing"

	_ "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/disintegration/imaging"
	_ "github.com/gen2brain/webp"
	_ "github.com/modelcontextprotocol/go-sdk/mcp"
	_ "google.golang.org/genai"
)

// TestT01_DependenciesResolveAndLink verifies T-01: All key dependencies compile and link on this platform.
func TestT01_DependenciesResolveAndLink(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if img.Bounds().Dx() != 10 {
		t.Fatalf("unexpected image bounds")
	}
}
