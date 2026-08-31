package main

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/doctor"
	"github.com/toscodevjs/matriz/internal/manifest"
	"github.com/toscodevjs/matriz/internal/mcpserver"
	"github.com/toscodevjs/matriz/internal/providers"
	"github.com/toscodevjs/matriz/internal/providers/fake"
	"github.com/toscodevjs/matriz/internal/providers/gemini"
	"github.com/toscodevjs/matriz/internal/tui"
	"github.com/toscodevjs/matriz/internal/version"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	cmd := args[0]

	switch cmd {
	case "version", "--version", "-v":
		jsonOutput := hasFlag(args[1:], "--json", "-j")
		info := version.GetInfo()
		if jsonOutput {
			data, err := info.JSON()
			if err != nil {
				fmt.Fprintf(stderr, "Error generating JSON version: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, data)
		} else {
			fmt.Fprintln(stdout, info.String())
		}
		return 0

	case "doctor":
		jsonOutput := hasFlag(args[1:], "--json", "-j")
		cfg := config.LoadFromEnv()
		report := doctor.Run(context.Background(), cfg)
		if jsonOutput {
			data, err := report.JSON()
			if err != nil {
				fmt.Fprintf(stderr, "Error generating JSON doctor report: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, data)
		} else {
			fmt.Fprint(stdout, report.Format())
		}
		if !report.Healthy {
			return 1
		}
		return 0

	case "mcp":
		cfg := config.LoadFromEnv()
		guard := budget.NewGuard(cfg.BudgetUSD, cfg.MaxGenerativeCalls)
		reg := providers.NewRegistry()
		reg.Register(fake.NewFakeProvider())

		if cfg.GoogleAPIKey != "" {
			geminiProv, err := gemini.NewGeminiProvider(context.Background(), cfg.GoogleAPIKey, cfg.ModelDraft, cfg.ModelFinal)
			if err == nil {
				reg.Register(geminiProv)
			}
		}

		server := mcpserver.NewServer(cfg, reg, guard)
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(stderr, "MCP server error: %v\n", err)
			return 1
		}
		return 0

	case "tui":
		cfg := config.LoadFromEnv()
		m, err := manifest.ScanProject(cfg.ProjectRoot, "")
		if err != nil {
			fmt.Fprintf(stderr, "Error scanning manifest: %v\n", err)
			return 1
		}
		p := tea.NewProgram(tui.NewModel(m, cfg.ProjectRoot), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(stderr, "Error running TUI: %v\n", err)
			return 1
		}
		return 0

	case "help", "--help", "-h":
		printHelp(stdout)
		return 0

	default:
		fmt.Fprintf(stderr, "Unknown command: %q\nRun 'matriz help' for usage.\n", cmd)
		return 1
	}
}

func hasFlag(args []string, flags ...string) bool {
	for _, a := range args {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `MATRIZ — Agentic image curation, transformation & generation MCP suite

Usage:
  matriz <command> [flags]

Available Commands:
  doctor     Run diagnostic health checks on codecs, config, and MCP integrations
  version    Display version and build information (--json supported)
  mcp        Start the stdio Model Context Protocol (MCP) server
  tui        Launch the interactive terminal curation and preview UI
  help       Show this help message

Flags:
  -v, --version   Show version
  -h, --help      Show help
  -j, --json      Output in JSON format (doctor, version)`)
}
