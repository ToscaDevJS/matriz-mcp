package providers

import (
	"context"
	"time"

	"github.com/toscodevjs/matriz/internal/core"
)

// Capability describes what a generative provider can do. The core uses this
// to route requests and refuse unsupported operations before spending money.
type Capability string

const (
	CapabilityGenerate    Capability = "generate"
	CapabilityInpaint     Capability = "inpaint"
	CapabilityOutpaint    Capability = "outpaint"
	CapabilityRemoveBG    Capability = "remove_background"
	CapabilityUpscale     Capability = "upscale"
	CapabilityDeterminism Capability = "seeded" // same seed + params => same image
)

// GenerateRequest is provider-agnostic. Provider implementations translate it
// into their own wire format. Never add a provider-specific field here.
type GenerateRequest struct {
	Prompt         string
	NegativePrompt string
	Width, Height  int
	Count          int    // number of drafts
	Seed           *int64 // nil => provider picks; result MUST report it back
	Model          string // empty => provider default
}

// EditRequest covers inpaint / outpaint / background removal.
type EditRequest struct {
	Source    []byte // raw image bytes
	Mask      []byte // nil for operations that do not take a mask
	Prompt    string
	Operation Capability
	Seed      *int64
	Model     string
}

// Result carries the produced images plus everything needed to reproduce them.
type Result struct {
	Images   [][]byte
	MIMEType string
	Seed     int64
	Model    string
	CostUSD  float64 // provider's own accounting; 0 if unknown
}

// Provider is the only thing the core knows about image generation.
// Adding a provider must not require touching the core or the tools.
type Provider interface {
	Name() string
	Capabilities() []Capability
	// EstimateCostUSD is called BEFORE the request and must not hit the network.
	EstimateCostUSD(req GenerateRequest) float64
	Generate(ctx context.Context, req GenerateRequest) (*Result, error)
	Edit(ctx context.Context, req EditRequest) (*Result, error)
}

// BuildSidecar constructs a sidecar struct for a generative result according to §5.4 & §5.15.
func BuildSidecar(ref core.AssetRef, providerName string, res *Result, req GenerateRequest) *core.Sidecar {
	seeded := (req.Seed != nil && res.Seed != 0)
	params := map[string]any{
		"width":  req.Width,
		"height": req.Height,
		"seeded": seeded,
	}

	return &core.Sidecar{
		Schema:         core.SidecarSchema,
		Ref:            ref,
		Origin:         core.OriginGenerated,
		CreatedAt:      time.Now().UTC(),
		Provider:       providerName,
		Model:          res.Model,
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
		Seed:           res.Seed,
		Params:         params,
		CostUSD:        res.CostUSD,
	}
}
