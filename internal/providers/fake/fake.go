package fake

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sync"
	"time"

	"github.com/toscodevjs/matriz/internal/providers"
)

// FakeProvider is a deterministic in-memory provider used across tests (§4 PR-2).
type FakeProvider struct {
	mu          sync.Mutex
	invocations int
	outW, outH  int
	jobs        map[string]*providers.VideoJob
	jobResults  map[string]*providers.VideoResult
	jobReqs     map[string]providers.VideoRequest
	seq         int64
}

// SetOutputSize forces every image produced by Generate or Edit to a fixed size
// regardless of the dimensions a request asks for, mirroring providers that
// answer with their own resolution instead of the one requested. Zero clears
// the override.
func (f *FakeProvider) SetOutputSize(w, h int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outW, f.outH = w, h
}

// NewFakeProvider initializes an offline FakeProvider instance.
func NewFakeProvider() *FakeProvider {
	return &FakeProvider{
		jobs:       make(map[string]*providers.VideoJob),
		jobResults: make(map[string]*providers.VideoResult),
		jobReqs:    make(map[string]providers.VideoRequest),
	}
}

// Name returns the provider identifier.
func (f *FakeProvider) Name() string {
	return "fake"
}

// Capabilities lists supported operations.
func (f *FakeProvider) Capabilities() []providers.Capability {
	return []providers.Capability{
		providers.CapabilityGenerate,
		providers.CapabilityInpaint,
		providers.CapabilityOutpaint,
		providers.CapabilityRemoveBG,
		providers.CapabilityUpscale,
		providers.CapabilityDeterminism,
		providers.CapabilityVideoDraft,
		providers.CapabilityVideoFinal,
		providers.CapabilityImageToVideo,
		providers.CapabilityTextToVideo,
	}
}

// EstimateCostUSD returns a nominal non-zero cost for testing budget tracking.
func (f *FakeProvider) EstimateCostUSD(req providers.GenerateRequest) float64 {
	count := req.Count
	if count <= 0 {
		count = 1
	}
	return 0.01 * float64(count)
}

// Generate returns deterministic PNG images based on prompt and seed.
func (f *FakeProvider) Generate(ctx context.Context, req providers.GenerateRequest) (*providers.Result, error) {
	f.mu.Lock()
	f.invocations++
	f.mu.Unlock()

	count := req.Count
	if count <= 0 {
		count = 1
	}

	var seed int64
	if req.Seed != nil {
		seed = *req.Seed
	}

	w, h := req.Width, req.Height
	if w <= 0 {
		w = 512
	}
	if h <= 0 {
		h = 512
	}

	f.mu.Lock()
	if f.outW > 0 && f.outH > 0 {
		w, h = f.outW, f.outH
	}
	f.mu.Unlock()

	images := make([][]byte, count)
	for i := 0; i < count; i++ {
		imgBytes, err := f.renderSolidImage(w, h, req.Prompt, seed+int64(i))
		if err != nil {
			return nil, fmt.Errorf("fake render failed: %w", err)
		}
		images[i] = imgBytes
	}

	return &providers.Result{
		Images:   images,
		MIMEType: "image/png",
		Seed:     seed,
		Model:    "fake-nano-banana",
		CostUSD:  0.01 * float64(count),
	}, nil
}

// Edit produces deterministic mock results for inpainting, outpainting, and background removal.
func (f *FakeProvider) Edit(ctx context.Context, req providers.EditRequest) (*providers.Result, error) {
	f.mu.Lock()
	f.invocations++
	f.mu.Unlock()

	var seed int64
	if req.Seed != nil {
		seed = *req.Seed
	}

	w, h := 512, 512
	f.mu.Lock()
	if f.outW > 0 && f.outH > 0 {
		w, h = f.outW, f.outH
	}
	f.mu.Unlock()

	imgBytes, err := f.renderSolidImage(w, h, req.Prompt, seed)
	if err != nil {
		return nil, fmt.Errorf("fake edit render failed: %w", err)
	}

	return &providers.Result{
		Images:   [][]byte{imgBytes},
		MIMEType: "image/png",
		Seed:     seed,
		Model:    "fake-nano-banana",
		CostUSD:  0.01,
	}, nil
}

// EstimateVideoCostUSD estimates the video generation cost based on duration.
func (f *FakeProvider) EstimateVideoCostUSD(req providers.VideoRequest) float64 {
	dur := req.DurationSec
	if dur <= 0 {
		dur = 5.0
	}
	return 0.05 * dur
}

// StartVideo initializes a mock video generation job.
func (f *FakeProvider) StartVideo(ctx context.Context, req providers.VideoRequest) (*providers.VideoJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invocations++
	f.seq++

	dur := req.DurationSec
	if dur <= 0 {
		dur = 5.0
	}
	fps := req.FPS
	if fps <= 0 {
		fps = 24
	}

	jobID := fmt.Sprintf("fake-job-%d", f.seq)
	job := &providers.VideoJob{
		ID:               jobID,
		Provider:         "fake",
		Model:            "fake-veo-turbo",
		Status:           providers.VideoJobProcessing,
		ProgressPct:      50,
		CreatedAt:        time.Now().UTC(),
		EstimatedCostUSD: f.EstimateVideoCostUSD(req),
	}

	var seed int64
	if req.Seed != nil {
		seed = *req.Seed
	}

	posterBytes, _ := f.renderSolidImage(512, 512, req.Prompt, seed)

	res := &providers.VideoResult{
		VideoBytes: []byte("fake-mp4-stream-bytes"),
		PosterPNG:  posterBytes,
		MIMEType:   "video/mp4",
		Duration:   dur,
		FPS:        fps,
		Width:      1280,
		Height:     720,
		Seed:       seed,
		Model:      "fake-veo-turbo",
		CostUSD:    job.EstimatedCostUSD,
	}

	f.jobs[jobID] = job
	f.jobResults[jobID] = res
	f.jobReqs[jobID] = req
	return job, nil
}

// PollVideo returns the current status and result for an asynchronous video job.
func (f *FakeProvider) PollVideo(ctx context.Context, jobID string) (*providers.VideoJob, *providers.VideoResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	job, ok := f.jobs[jobID]
	if !ok {
		return nil, nil, fmt.Errorf("job %q not found", jobID)
	}

	// In test mode, transition from Processing to Completed on poll
	job.Status = providers.VideoJobCompleted
	job.ProgressPct = 100

	res := f.jobResults[jobID]
	return job, res, nil
}

// CancelVideo cancels an in-flight video generation job.
func (f *FakeProvider) CancelVideo(ctx context.Context, jobID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	job, ok := f.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %q not found", jobID)
	}

	job.Status = providers.VideoJobCancelled
	return nil
}

// InvocationCount returns the number of times Generate or Edit was called.
func (f *FakeProvider) InvocationCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.invocations
}

func (f *FakeProvider) renderSolidImage(w, h int, prompt string, seed int64) ([]byte, error) {
	hasher := sha256.New()
	hasher.Write([]byte(prompt))
	_ = binary.Write(hasher, binary.LittleEndian, seed)
	hash := hasher.Sum(nil)

	r := hash[0]
	g := hash[1]
	b := hash[2]

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := color.RGBA{R: r, G: g, B: b, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
