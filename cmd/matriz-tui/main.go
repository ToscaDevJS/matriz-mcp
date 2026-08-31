package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/manifest"
	"github.com/toscodevjs/matriz/internal/tui"
)

func main() {
	cfg := config.LoadFromEnv()

	m, err := manifest.ScanProject(cfg.ProjectRoot, "")
	if err != nil {
		log.Fatalf("failed to scan project manifest: %v", err)
	}

	p := tea.NewProgram(tui.NewModel(m, cfg.ProjectRoot), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
