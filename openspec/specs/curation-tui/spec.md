# Curation TUI Specification

## Purpose
Provides a terminal user interface for asset inspection, curation, and local HTML preview rendering.

## Requirements

### Requirement: TUI Model Initialization & Navigation
The system MUST construct an interactive Bubbletea model from a manifest and support navigation keys (arrow keys, enter, export, quit).

#### Scenario: Model construction from test fixture
- GIVEN a valid fixture manifest in `testdata/`
- WHEN `NewModel` is initialized
- THEN the model contains the asset list without accessing the filesystem outside `testdata/`

### Requirement: Local HTML Preview Trigger
The system MUST generate a local static HTML preview file and open it in the default system browser upon pressing Enter.

#### Scenario: Preview generation
- GIVEN an active asset in the TUI
- WHEN Enter key is pressed
- THEN a temporary HTML preview document is generated linking to local image files
