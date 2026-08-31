package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/manifest"
	"github.com/toscodevjs/matriz/internal/preview"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#EE6FF8")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(lipgloss.Color("#DDDDDD"))

	infoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)
)

// Model represents the state of the Matriz Curation TUI.
type Model struct {
	manifest    *manifest.Manifest
	projectRoot string
	cursor      int
	keys        KeyMap
	statusMsg   string
}

// NewModel creates an initialized TUI Model.
func NewModel(m *manifest.Manifest, projectRoot string) Model {
	if m == nil {
		m = &manifest.Manifest{
			Schema: manifest.ManifestSchema,
			Slots:  make([]manifest.Slot, 0),
			Assets: make([]core.Asset, 0),
		}
	}
	return Model{
		manifest:    m,
		projectRoot: projectRoot,
		cursor:      0,
		keys:        DefaultKeyMap,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.manifest.Assets)-1 {
				m.cursor++
			}

		case key.Matches(msg, m.keys.Enter):
			if len(m.manifest.Assets) > 0 {
				asset := m.manifest.Assets[m.cursor]
				absPath, _ := core.ResolveRef(m.projectRoot, asset.Ref)
				sidecar, _ := core.ReadSidecar(absPath)
				htmlPath, err := preview.GeneratePreviewHTML(m.projectRoot, asset, sidecar)
				if err == nil {
					_ = preview.OpenBrowser(htmlPath)
					m.statusMsg = fmt.Sprintf("Opened preview: %s", htmlPath)
				} else {
					m.statusMsg = fmt.Sprintf("Preview error: %v", err)
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf(" MATRIZ Curation TUI — %s ", m.manifest.Project)))
	b.WriteString("\n\n")

	if len(m.manifest.Assets) == 0 {
		b.WriteString("  No assets found in manifest. Add images to assets/ or scan project.\n")
	} else {
		b.WriteString("  Assets:\n")
		for i, asset := range m.manifest.Assets {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
				line := fmt.Sprintf("%s %s (%dx%d, %s, %s)", cursor, asset.Ref, asset.Dims.Width, asset.Dims.Height, asset.Origin, asset.MIMEType)
				b.WriteString(selectedItemStyle.Render(line) + "\n")
			} else {
				line := fmt.Sprintf("%s %s (%dx%d, %s)", cursor, asset.Ref, asset.Dims.Width, asset.Dims.Height, asset.Origin)
				b.WriteString(normalItemStyle.Render(line) + "\n")
			}
		}
	}

	b.WriteString("\n")
	if len(m.manifest.Assets) > 0 && m.cursor < len(m.manifest.Assets) {
		selected := m.manifest.Assets[m.cursor]
		detail := fmt.Sprintf("Selected: %s\nDimensions: %dx%d\nOrigin: %s\nSize: %.2f KB",
			selected.Ref, selected.Dims.Width, selected.Dims.Height, selected.Origin, float64(selected.Bytes)/1024.0)
		b.WriteString(infoBoxStyle.Render(detail) + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\nStatus: " + m.statusMsg + "\n")
	}

	b.WriteString("\nControls: ↑/k up • ↓/j down • enter open preview • q quit\n")

	return b.String()
}

// Assets returns the asset list in the model.
func (m Model) Assets() []core.Asset {
	return m.manifest.Assets
}

// SelectedIndex returns the current cursor index.
func (m Model) SelectedIndex() int {
	return m.cursor
}
