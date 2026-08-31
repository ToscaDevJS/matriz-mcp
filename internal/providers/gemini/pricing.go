package gemini

import (
	"strings"

	"github.com/toscodevjs/matriz/internal/providers"
)

// PricingTable maintains model and resolution token-to-USD mappings (§5.15).
type PricingTable struct {
	// prices maps model prefix/name to unit cost in USD per image
	prices map[string]map[string]float64
	// worstCaseCostUSD is used when an unknown model is requested (hard rule 7.12)
	worstCaseCostUSD float64
}

// NewPricingTable initializes verified pricing data as of 2026-08-31.
func NewPricingTable() *PricingTable {
	return &PricingTable{
		prices: map[string]map[string]float64{
			"flash-lite": {
				"1k": 0.0336,
				"2k": 0.0336,
				"4k": 0.0672,
			},
			"pro": {
				"1k": 0.134,
				"2k": 0.134,
				"4k": 0.240,
			},
		},
		worstCaseCostUSD: 0.240,
	}
}

// WorstCaseCostUSD returns the highest known per-image cost multiplied by count.
func (p *PricingTable) WorstCaseCostUSD(count int) float64 {
	if count <= 0 {
		count = 1
	}
	return p.worstCaseCostUSD * float64(count)
}

// EstimateCostUSD calculates pre-flight cost without network access.
func (p *PricingTable) EstimateCostUSD(req providers.GenerateRequest) float64 {
	count := req.Count
	if count <= 0 {
		count = 1
	}

	modelLower := strings.ToLower(req.Model)

	// Determine resolution tier
	maxEdge := req.Width
	if req.Height > maxEdge {
		maxEdge = req.Height
	}

	resTier := "1k"
	if maxEdge > 2048 {
		resTier = "4k"
	} else if maxEdge > 1024 {
		resTier = "2k"
	}

	var modelTier string
	if strings.Contains(modelLower, "lite") || strings.Contains(modelLower, "flash") {
		modelTier = "flash-lite"
	} else if strings.Contains(modelLower, "pro") {
		modelTier = "pro"
	} else if modelLower == "" {
		// Default to flash-lite for empty model
		modelTier = "flash-lite"
	}

	if tierMap, exists := p.prices[modelTier]; exists {
		if cost, ok := tierMap[resTier]; ok {
			return cost * float64(count)
		}
	}

	// Hard rule 7.12 & 5.15: Fallback to worst-case price, never zero
	return p.WorstCaseCostUSD(count)
}
