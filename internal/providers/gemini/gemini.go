package gemini

import (
	"context"
	"fmt"
	"math"
	"strings"

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

	resp, err := g.client.Models.GenerateContent(ctx, modelID, contents, buildGenerateConfig(req))
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

	return &providers.Result{
		Images:   images,
		MIMEType: mimeType,
		Seed:     seed,
		Model:    modelID,
		CostUSD:  g.settledCostUSD(req, len(images)),
	}, nil
}

// buildGenerateConfig translates a provider-agnostic request into Gemini's wire
// config. Count maps to CandidateCount: a caller asking for N drafts must ask
// the model for N candidates, otherwise the request silently degrades to one.
func buildGenerateConfig(req providers.GenerateRequest) *genai.GenerateContentConfig {
	count := req.Count
	if count <= 0 {
		count = 1
	}

	cfg := &genai.GenerateContentConfig{
		ResponseModalities: []string{"IMAGE"},
		CandidateCount:     int32(count),
		// The image models take a ratio and a size tier, never a pixel size.
		// Sending neither is what made a 16:9 request come back as 1408x768.
		ImageConfig: &genai.ImageConfig{
			AspectRatio: nearestAspectRatio(req.Width, req.Height),
			ImageSize:   strings.ToUpper(resolutionTier(req.Width, req.Height)),
		},
	}

	// Hard rule 7.12: seeds are never invented. Absent a caller-supplied seed the
	// field stays unset and the provider reports back whatever it used.
	if req.Seed != nil {
		seed32 := int32(*req.Seed)
		cfg.Seed = &seed32
	}

	return cfg
}

// settledCostUSD prices a finished generation by the number of images actually
// returned. EstimateCostUSD deliberately prices the full requested count so the
// budget guard can fail closed before spending (hard rule 7.11); once the call
// has returned, charging for undelivered images would burn budget on nothing.
func (g *GeminiProvider) settledCostUSD(req providers.GenerateRequest, imagesReturned int) float64 {
	settled := req
	settled.Count = imagesReturned
	return g.pricingTable.EstimateCostUSD(settled)
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

// geminiAspectRatios lists the ratios the image models accept.
var geminiAspectRatios = []struct {
	label string
	value float64
}{
	{"1:1", 1.0},
	{"2:3", 2.0 / 3.0},
	{"3:2", 3.0 / 2.0},
	{"3:4", 3.0 / 4.0},
	{"4:3", 4.0 / 3.0},
	{"9:16", 9.0 / 16.0},
	{"16:9", 16.0 / 9.0},
	{"21:9", 21.0 / 9.0},
}

// nearestAspectRatio maps requested pixel dimensions onto the closest ratio the
// model accepts. The MCP layer resolves the caller's ratio into pixels so the
// pricing table can pick a tier, so the provider recovers the label the API
// wants. Comparison is relative, which keeps wide ratios from being pulled
// toward the narrow end of the list by raw distance.
func nearestAspectRatio(width, height int) string {
	if width <= 0 || height <= 0 {
		return "16:9"
	}

	got := float64(width) / float64(height)
	best, bestDelta := "16:9", math.Inf(1)
	for _, ar := range geminiAspectRatios {
		if delta := math.Abs(got-ar.value) / ar.value; delta < bestDelta {
			best, bestDelta = ar.label, delta
		}
	}

	return best
}
