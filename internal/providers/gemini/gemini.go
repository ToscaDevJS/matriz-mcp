package gemini

import (
	"context"
	"fmt"

	"github.com/toscodevjs/matriz/internal/providers"
	"google.golang.org/genai"
)

// GeminiProvider implements providers.Provider using the official Google GenAI Go SDK.
type GeminiProvider struct {
	client       *genai.Client
	pricingTable *PricingTable
	defaultDraft string
	defaultFinal string
}

// NewGeminiProvider initializes a GeminiProvider with an API key and model configurations.
func NewGeminiProvider(ctx context.Context, apiKey, draftModel, finalModel string) (*GeminiProvider, error) {
	if draftModel == "" {
		draftModel = "gemini-3.1-flash-lite-image"
	}
	if finalModel == "" {
		finalModel = "gemini-3-pro-image-preview"
	}

	var client *genai.Client
	var err error
	if apiKey != "" {
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini client: %w", err)
		}
	}

	return &GeminiProvider{
		client:       client,
		pricingTable: NewPricingTable(),
		defaultDraft: draftModel,
		defaultFinal: finalModel,
	}, nil
}

// Name returns the provider identifier.
func (g *GeminiProvider) Name() string {
	return "gemini"
}

// Capabilities lists supported generative features.
func (g *GeminiProvider) Capabilities() []providers.Capability {
	return []providers.Capability{
		providers.CapabilityGenerate,
		providers.CapabilityInpaint,
		providers.CapabilityOutpaint,
		providers.CapabilityRemoveBG,
		providers.CapabilityUpscale,
	}
}

// EstimateCostUSD performs offline cost calculation.
func (g *GeminiProvider) EstimateCostUSD(req providers.GenerateRequest) float64 {
	if req.Model == "" {
		req.Model = g.defaultDraft
	}
	return g.pricingTable.EstimateCostUSD(req)
}

// Generate invokes Gemini multimodal generation.
func (g *GeminiProvider) Generate(ctx context.Context, req providers.GenerateRequest) (*providers.Result, error) {
	if g.client == nil {
		return nil, fmt.Errorf("gemini client not initialized (missing GOOGLE_API_KEY)")
	}

	modelID := req.Model
	if modelID == "" {
		modelID = g.defaultDraft
	}

	prompt := req.Prompt
	if req.NegativePrompt != "" {
		prompt = fmt.Sprintf("%s (Avoid: %s)", prompt, req.NegativePrompt)
	}

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		},
	}

	cfg := &genai.GenerateContentConfig{
		ResponseModalities: []string{"IMAGE"},
	}

	if req.Seed != nil {
		seed32 := int32(*req.Seed)
		cfg.Seed = &seed32
	}

	resp, err := g.client.Models.GenerateContent(ctx, modelID, contents, cfg)
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	var images [][]byte
	var mimeType string = "image/png"

	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if part.InlineData != nil && len(part.InlineData.Data) > 0 {
					images = append(images, part.InlineData.Data)
					if part.InlineData.MIMEType != "" {
						mimeType = part.InlineData.MIMEType
					}
				}
			}
		}
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("gemini returned no image parts in response")
	}

	var seed int64
	if req.Seed != nil {
		seed = *req.Seed
	}

	costUSD := g.pricingTable.EstimateCostUSD(req)

	return &providers.Result{
		Images:   images,
		MIMEType: mimeType,
		Seed:     seed,
		Model:    modelID,
		CostUSD:  costUSD,
	}, nil
}

// Edit invokes Gemini image editing / multimodal refinement.
func (g *GeminiProvider) Edit(ctx context.Context, req providers.EditRequest) (*providers.Result, error) {
	if g.client == nil {
		return nil, fmt.Errorf("gemini client not initialized (missing GOOGLE_API_KEY)")
	}

	modelID := req.Model
	if modelID == "" {
		modelID = g.defaultFinal
	}

	instruction := fmt.Sprintf("Perform %s: %s", req.Operation, req.Prompt)
	parts := []*genai.Part{
		genai.NewPartFromText(instruction),
		{
			InlineData: &genai.Blob{
				Data:     req.Source,
				MIMEType: "image/png",
			},
		},
	}

	if len(req.Mask) > 0 {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     req.Mask,
				MIMEType: "image/png",
			},
		})
	}

	contents := []*genai.Content{
		{
			Parts: parts,
		},
	}

	cfg := &genai.GenerateContentConfig{
		ResponseModalities: []string{"IMAGE"},
	}

	if req.Seed != nil {
		seed32 := int32(*req.Seed)
		cfg.Seed = &seed32
	}

	resp, err := g.client.Models.GenerateContent(ctx, modelID, contents, cfg)
	if err != nil {
		return nil, fmt.Errorf("gemini edit failed: %w", err)
	}

	var images [][]byte
	var mimeType string = "image/png"

	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if part.InlineData != nil && len(part.InlineData.Data) > 0 {
					images = append(images, part.InlineData.Data)
					if part.InlineData.MIMEType != "" {
						mimeType = part.InlineData.MIMEType
					}
				}
			}
		}
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("gemini returned no image parts in edit response")
	}

	var seed int64
	if req.Seed != nil {
		seed = *req.Seed
	}

	costUSD := g.pricingTable.EstimateCostUSD(providers.GenerateRequest{
		Model: modelID,
		Count: 1,
	})

	return &providers.Result{
		Images:   images,
		MIMEType: mimeType,
		Seed:     seed,
		Model:    modelID,
		CostUSD:  costUSD,
	}, nil
}
