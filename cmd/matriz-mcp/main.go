package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/mcpserver"
	"github.com/toscodevjs/matriz/internal/providers"
	"github.com/toscodevjs/matriz/internal/providers/fake"
	"github.com/toscodevjs/matriz/internal/providers/gemini"
)

func main() {
	cfg := config.LoadFromEnv()
	guard := budget.NewGuard(cfg.BudgetUSD, cfg.MaxGenerativeCalls)

	reg := providers.NewRegistry()
	reg.Register(fake.NewFakeProvider())

	if cfg.GoogleAPIKey != "" {
		geminiProv, err := gemini.NewGeminiProvider(context.Background(), cfg.GoogleAPIKey, cfg.ModelDraft, cfg.ModelFinal)
		if err == nil {
			geminiProv.SetVideoModels(cfg.ModelVideoDraft, cfg.ModelVideoFinal)
			reg.Register(geminiProv)
		}
	}

	server := mcpserver.NewServer(cfg, reg, guard)

	// stdio only. No HTTP transport in v0.1.0.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
	_ = os.Stdout // stdout belongs to the protocol: never print to it.
}
