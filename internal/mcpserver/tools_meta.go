package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/providers"
)

type ListModelsIn struct{}

type ListModelsOut struct {
	ActiveProvider     string   `json:"active_provider"`
	AvailableProviders []string `json:"available_providers"`
	DraftModel         string   `json:"draft_model"`
	FinalModel         string   `json:"final_model"`
	BudgetSpentUSD     float64  `json:"budget_spent_usd"`
	BudgetLimitUSD     float64  `json:"budget_limit_usd"`
	BudgetLeftUSD      float64  `json:"budget_left_usd"`
	CallsCompleted     int      `json:"calls_completed"`
	MaxCallsAllowed    int      `json:"max_calls_allowed"`
}

func handleListModels(cfg *config.Config, reg *providers.Registry, guard *budget.Guard) mcp.ToolHandlerFor[ListModelsIn, ListModelsOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ListModelsIn) (*mcp.CallToolResult, ListModelsOut, error) {
		out := ListModelsOut{
			ActiveProvider:     cfg.Provider,
			AvailableProviders: reg.List(),
			DraftModel:         cfg.ModelDraft,
			FinalModel:         cfg.ModelFinal,
			BudgetSpentUSD:     guard.SpentUSD(),
			BudgetLimitUSD:     guard.LimitUSD(),
			BudgetLeftUSD:      guard.BudgetLeft(),
			CallsCompleted:     guard.Calls(),
			MaxCallsAllowed:    guard.MaxCalls(),
		}
		return nil, out, nil
	}
}
