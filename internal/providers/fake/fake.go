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

	"github.com/toscodevjs/matriz/internal/providers"
)

// FakeProvider is a deterministic in-memory provider used across tests (§4 PR-2).
type FakeProvider struct {
	mu          sync.Mutex
	invocations int
	outW, outH  int
}

// SetOutputSize forces every generated image to a fixed size regardless of the
// dimensions a request asks for, mirroring providers that answer with their own
// resolution instead of the one requested. Zero clears the override.
func (f *FakeProvider) SetOutputSize(w, h int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outW, f.outH = w, h
}

// NewFakeProvider initializes an offline FakeProvider instance.
func NewFakeProvider() *FakeProvider {
	return &FakeProvider{}
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

// Generate produces deterministic PNG image bytes derived from the prompt and seed.
func (f *FakeProvider) Generate(ctx context.Context, req providers.GenerateRequest) (*providers.Result, error) {
	f.mu.Lock()
	f.invocations++
	f.mu.Unlock()

	count := req.Count
	if count <= 0 {
		count = 1
	}

	w := req.Width
	if w <= 0 {
		w = 512
	}
	h := req.Height
	if h <= 0 {
		h = 512
	}

	f.mu.Lock()
	if f.outW > 0 && f.outH > 0 {
		w, h = f.outW, f.outH
	}
	f.mu.Unlock()

	var seed int64
	if req.Seed != nil {
		seed = *req.Seed
	} else {
		seed = int64(f.InvocationCount() * 1000)
	}

	images := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		imgBytes, err := f.renderSolidImage(w, h, req.Prompt, seed+int64(i))
		if err != nil {
			return nil, fmt.Errorf("fake render failed: %w", err)
		}
		images = append(images, imgBytes)
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

	imgBytes, err := f.renderSolidImage(512, 512, req.Prompt, seed)
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
