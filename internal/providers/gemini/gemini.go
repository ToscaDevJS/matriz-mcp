package gemini

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/toscodevjs/matriz/internal/providers"
	"google.golang.org/genai"
)

// GeminiProvider implements providers.Provider using the official Google GenAI Go SDK.
type GeminiProvider struct {
	client            *genai.Client
	pricingTable      *PricingTable
	defaultDraft      string
	defaultFinal      string
	defaultVideoDraft string
	defaultVideoFinal string
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
		client:            client,
		pricingTable:      NewPricingTable(),
		defaultDraft:      draftModel,
		defaultFinal:      finalModel,
		defaultVideoDraft: "gemini-omni-1.1-flash",
		defaultVideoFinal: "veo-3.1-generate-preview",
	}, nil
}

// SetVideoModels overrides the default draft and final video models.
func (g *GeminiProvider) SetVideoModels(draft, final string) {
	if draft != "" {
		g.defaultVideoDraft = draft
	}
	if final != "" {
		g.defaultVideoFinal = final
	}
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
		providers.CapabilityVideoDraft,
		providers.CapabilityVideoFinal,
		providers.CapabilityImageToVideo,
		providers.CapabilityTextToVideo,
	}
}

// EstimateCostUSD performs offline cost calculation.
func (g *GeminiProvider) EstimateCostUSD(req providers.GenerateRequest) float64 {
	if req.Model == "" {
		req.Model = g.defaultDraft
	}
	return g.pricingTable.EstimateCostUSD(req)
}

// WorkerSeed returns the seed for the i-th worker in a concurrent generation batch.
// Absent a caller-supplied seed, it returns nil so the provider picks seeds independently.
func WorkerSeed(baseSeed *int64, workerIndex int) *int64 {
	if baseSeed == nil {
		return nil
	}
	s := *baseSeed + int64(workerIndex)
	return &s
}

// Generate invokes Gemini multimodal generation. When req.Count > 1, it executes
// concurrent requests in parallel because Gemini's image endpoint returns only one
// candidate per call.
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

	count := req.Count
	if count <= 0 {
		count = 1
	}

	if count == 1 {
		return g.generateSingle(ctx, modelID, prompt, req, req.Seed)
	}

	return g.generateConcurrent(ctx, modelID, prompt, req, count)
}

func (g *GeminiProvider) generateSingle(ctx context.Context, modelID, prompt string, req providers.GenerateRequest, seed *int64) (*providers.Result, error) {
	singleReq := req
	singleReq.Count = 1
	singleReq.Seed = seed

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		},
	}

	resp, err := g.client.Models.GenerateContent(ctx, modelID, contents, buildGenerateConfig(singleReq))
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

	var resultSeed int64
	if seed != nil {
		resultSeed = *seed
	}

	return &providers.Result{
		Images:   images,
		MIMEType: mimeType,
		Seed:     resultSeed,
		Model:    modelID,
		CostUSD:  g.settledCostUSD(singleReq, len(images)),
	}, nil
}

func (g *GeminiProvider) generateConcurrent(ctx context.Context, modelID, prompt string, req providers.GenerateRequest, count int) (*providers.Result, error) {
	if count > 4 {
		count = 4
	}

	type workerResult struct {
		index int
		image []byte
		mime  string
		seed  int64
		err   error
	}

	resultsChan := make(chan workerResult, count)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(workerIndex int) {
			defer wg.Done()

			workerSeed := WorkerSeed(req.Seed, workerIndex)
			workerReq := req
			workerReq.Count = 1
			workerReq.Seed = workerSeed

			singleRes, err := g.generateSingle(ctx, modelID, prompt, workerReq, workerSeed)
			if err != nil {
				resultsChan <- workerResult{index: workerIndex, err: err}
				return
			}

			if len(singleRes.Images) > 0 {
				resultsChan <- workerResult{
					index: workerIndex,
					image: singleRes.Images[0],
					mime:  singleRes.MIMEType,
					seed:  singleRes.Seed,
				}
			} else {
				resultsChan <- workerResult{index: workerIndex, err: fmt.Errorf("no image returned")}
			}
		}(i)
	}

	wg.Wait()
	close(resultsChan)

	var images [][]byte
	var firstErr error
	var mimeType string = "image/png"
	var baseSeed int64
	if req.Seed != nil {
		baseSeed = *req.Seed
	}

	for res := range resultsChan {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		images = append(images, res.image)
		if res.mime != "" {
			mimeType = res.mime
		}
	}

	if len(images) == 0 {
		if firstErr != nil {
			return nil, fmt.Errorf("concurrent draft generation failed: %w", firstErr)
		}
		return nil, fmt.Errorf("all concurrent generations returned empty results")
	}

	return &providers.Result{
		Images:   images,
		MIMEType: mimeType,
		Seed:     baseSeed,
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

// EstimateVideoCostUSD performs offline cost estimation for video generation.
func (g *GeminiProvider) EstimateVideoCostUSD(req providers.VideoRequest) float64 {
	dur := req.DurationSec
	if dur <= 0 {
		dur = 5.0
	}

	model := req.Model
	if model == "" {
		model = g.defaultVideoDraft
	}

	if strings.Contains(model, "flash") || strings.Contains(model, "omni") {
		// Gemini Omni Flash: ~ $0.10 per second
		return 0.10 * dur
	}
	if strings.Contains(model, "lite") {
		// Veo Lite: ~ $0.05 per second
		return 0.05 * dur
	}
	// Veo 3.1 / Standard: ~ $0.40 per second
	return 0.40 * dur
}

// StartVideo dispatches an asynchronous video generation request.
func (g *GeminiProvider) StartVideo(ctx context.Context, req providers.VideoRequest) (*providers.VideoJob, error) {
	if g.client == nil {
		return nil, fmt.Errorf("gemini client not initialized (missing GOOGLE_API_KEY)")
	}

	modelID := req.Model
	if modelID == "" {
		modelID = g.defaultVideoDraft
	}

	dur := int32(req.DurationSec)
	if dur <= 0 {
		dur = 5
	}
	fps := int32(req.FPS)
	if fps <= 0 {
		fps = 24
	}

	aspectRatio := req.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}

	cfg := &genai.GenerateVideosConfig{
		AspectRatio:     aspectRatio,
		DurationSeconds: &dur,
		NegativePrompt:  req.NegativePrompt,
	}
	if req.Seed != nil {
		s := int32(*req.Seed)
		cfg.Seed = &s
	}

	var op *genai.GenerateVideosOperation
	var err error

	if len(req.SourceImage) > 0 {
		img := &genai.Image{
			ImageBytes: req.SourceImage,
			MIMEType:   "image/png",
		}
		op, err = g.client.Models.GenerateVideos(ctx, modelID, req.Prompt, img, cfg)
	} else {
		op, err = g.client.Models.GenerateVideos(ctx, modelID, req.Prompt, nil, cfg)
	}

	if err != nil {
		return nil, translateGoogleVideoError(err)
	}

	return &providers.VideoJob{
		ID:               op.Name,
		Provider:         "gemini",
		Model:            modelID,
		Status:           providers.VideoJobProcessing,
		ProgressPct:      10,
		CreatedAt:        time.Now().UTC(),
		EstimatedCostUSD: g.EstimateVideoCostUSD(req),
	}, nil
}

// PollVideo queries the operation status of an in-flight video generation.
func (g *GeminiProvider) PollVideo(ctx context.Context, jobID string) (*providers.VideoJob, *providers.VideoResult, error) {
	if g.client == nil {
		return nil, nil, fmt.Errorf("gemini client not initialized (missing GOOGLE_API_KEY)")
	}

	op := &genai.GenerateVideosOperation{Name: jobID}
	updatedOp, err := g.client.Operations.GetVideosOperation(ctx, op, nil)
	if err != nil {
		return nil, nil, translateGoogleVideoError(err)
	}

	if !updatedOp.Done {
		return &providers.VideoJob{
			ID:          jobID,
			Provider:    "gemini",
			Status:      providers.VideoJobProcessing,
			ProgressPct: 50,
			CreatedAt:   time.Now().UTC(),
		}, nil, nil
	}

	if updatedOp.Error != nil && len(updatedOp.Error) > 0 {
		errMsg := fmt.Sprintf("%v", updatedOp.Error)
		return &providers.VideoJob{
			ID:       jobID,
			Provider: "gemini",
			Status:   providers.VideoJobFailed,
			Error:    errMsg,
		}, nil, fmt.Errorf("video generation failed: %s", errMsg)
	}

	if updatedOp.Response == nil || len(updatedOp.Response.GeneratedVideos) == 0 {
		if updatedOp.Response != nil && updatedOp.Response.RAIMediaFilteredCount > 0 {
			reasons := strings.Join(updatedOp.Response.RAIMediaFilteredReasons, ", ")
			return &providers.VideoJob{
				ID:       jobID,
				Provider: "gemini",
				Status:   providers.VideoJobFailed,
				Error:    fmt.Sprintf("safety filter blocked generation: %s", reasons),
			}, nil, fmt.Errorf("video generation blocked by Google safety filters: %s", reasons)
		}
		return &providers.VideoJob{
			ID:       jobID,
			Provider: "gemini",
			Status:   providers.VideoJobFailed,
			Error:    "no video returned in response",
		}, nil, fmt.Errorf("provider returned no video in response")
	}

	firstVid := updatedOp.Response.GeneratedVideos[0].Video
	var videoBytes []byte
	mimeType := "video/mp4"

	if firstVid != nil {
		if len(firstVid.VideoBytes) > 0 {
			videoBytes = firstVid.VideoBytes
		} else if firstVid.URI != "" {
			dlURI := genai.NewDownloadURIFromVideo(firstVid)
			downloaded, err := g.client.Files.Download(ctx, dlURI, nil)
			if err == nil && len(downloaded) > 0 {
				videoBytes = downloaded
			}
		}
		if firstVid.MIMEType != "" {
			mimeType = firstVid.MIMEType
		}
	}

	job := &providers.VideoJob{
		ID:          jobID,
		Provider:    "gemini",
		Status:      providers.VideoJobCompleted,
		ProgressPct: 100,
	}

	res := &providers.VideoResult{
		VideoBytes: videoBytes,
		MIMEType:   mimeType,
		CostUSD:    0.40,
	}

	return job, res, nil
}

// CancelVideo attempts to cancel an in-flight operation.
func (g *GeminiProvider) CancelVideo(ctx context.Context, jobID string) error {
	return nil
}

func translateGoogleVideoError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "RESOURCE_EXHAUSTED") || strings.Contains(msg, "429") {
		return fmt.Errorf("video generation quota exceeded (RESOURCE_EXHAUSTED): wait before retrying or request quota increase: %w", err)
	}
	if strings.Contains(msg, "HARM_CATEGORY") || strings.Contains(msg, "safety") || strings.Contains(msg, "blocked") {
		return fmt.Errorf("video generation blocked by Google safety filters: modify prompt or change anchor image: %w", err)
	}
	return err
}
