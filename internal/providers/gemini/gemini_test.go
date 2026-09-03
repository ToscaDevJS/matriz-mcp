package gemini

import (
	"strings"
	"testing"

	"github.com/toscodevjs/matriz/internal/providers"
)

// draftReq is the shape handleGenerateDrafts sends for a 16:9 draft batch:
// flash-lite model, 768px long edge, which prices at the "1k" tier.
func draftReq(count int) providers.GenerateRequest {
	return providers.GenerateRequest{
		Prompt: "a barber salon interior",
		Model:  "gemini-3.1-flash-lite-image",
		Width:  768,
		Height: 432,
		Count:  count,
	}
}

const flashLite1KPerImage = 0.0336

// TestSettledCostUSD_PricesImagesReturned covers the money bug: the budget guard
// must be charged for the images the provider actually returned, never for the
// number requested. Gemini answers a Count=4 request with a single candidate, so
// pricing the request instead of the response silently burns budget on images
// that were never delivered.
func TestSettledCostUSD_PricesImagesReturned(t *testing.T) {
	g := &GeminiProvider{pricingTable: NewPricingTable()}

	tests := []struct {
		name           string
		requested      int
		imagesReturned int
		want           float64
	}{
		{"four requested, one delivered", 4, 1, flashLite1KPerImage},
		{"four requested, four delivered", 4, 4, flashLite1KPerImage * 4},
		{"one requested, one delivered", 1, 1, flashLite1KPerImage},
		{"two requested, three delivered", 2, 3, flashLite1KPerImage * 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.settledCostUSD(draftReq(tt.requested), tt.imagesReturned)
			if got != tt.want {
				t.Errorf("settledCostUSD(count=%d, returned=%d) = %v, want %v",
					tt.requested, tt.imagesReturned, got, tt.want)
			}
		})
	}
}

// TestSettledCostUSD_NeverZeroForDeliveredImages guards hard rule 7.11 from the
// other side: an unknown model still prices at the worst case, so a delivered
// image is never settled at zero.
func TestSettledCostUSD_NeverZeroForDeliveredImages(t *testing.T) {
	g := &GeminiProvider{pricingTable: NewPricingTable()}

	req := draftReq(1)
	req.Model = "some-unreleased-model-id"

	got := g.settledCostUSD(req, 1)
	if got <= 0 {
		t.Errorf("settledCostUSD for an unknown model = %v, want the worst case, never zero", got)
	}
}

// TestEstimateCostUSD_StillPricesFullRequest pins the pre-flight side. Reserve
// must stay conservative and price every draft asked for, so the guard can
// refuse a batch before it spends. Only the settled cost follows reality.
func TestEstimateCostUSD_StillPricesFullRequest(t *testing.T) {
	g := &GeminiProvider{pricingTable: NewPricingTable()}

	got := g.EstimateCostUSD(draftReq(4))
	want := flashLite1KPerImage * 4
	if got != want {
		t.Errorf("EstimateCostUSD(count=4) = %v, want %v", got, want)
	}
}

// TestBuildGenerateConfig_RequestsEveryDraft covers the delivery half of the bug:
// Count never reached the wire, so Gemini was only ever asked for one candidate
// no matter how many drafts the caller wanted.
func TestBuildGenerateConfig_RequestsEveryDraft(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  int32
	}{
		{"four drafts", 4, 4},
		{"single draft", 1, 1},
		{"unset count defaults to one", 0, 1},
		{"negative count defaults to one", -3, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := buildGenerateConfig(draftReq(tt.count))
			if cfg.CandidateCount != tt.want {
				t.Errorf("CandidateCount = %d, want %d", cfg.CandidateCount, tt.want)
			}
		})
	}
}

// TestBuildGenerateConfig_KeepsImageModality is a regression guard: the config
// must keep asking for an IMAGE response, whatever else it grows.
func TestBuildGenerateConfig_KeepsImageModality(t *testing.T) {
	cfg := buildGenerateConfig(draftReq(1))

	if len(cfg.ResponseModalities) != 1 || cfg.ResponseModalities[0] != "IMAGE" {
		t.Errorf("ResponseModalities = %v, want [IMAGE]", cfg.ResponseModalities)
	}
}

// TestBuildGenerateConfig_SeedOnlyWhenGiven pins hard rule 7.12: seeds are never
// invented. No seed from the caller means no seed on the wire.
func TestBuildGenerateConfig_SeedOnlyWhenGiven(t *testing.T) {
	if cfg := buildGenerateConfig(draftReq(1)); cfg.Seed != nil {
		t.Errorf("Seed = %v, want nil when the caller supplied none", *cfg.Seed)
	}

	seed := int64(42)
	req := draftReq(1)
	req.Seed = &seed

	cfg := buildGenerateConfig(req)
	if cfg.Seed == nil {
		t.Fatal("Seed = nil, want 42 when the caller supplied one")
	}
	if *cfg.Seed != 42 {
		t.Errorf("Seed = %d, want 42", *cfg.Seed)
	}
}

// TestBuildGenerateConfig_SendsAspectRatio covers the bug that produced a
// 1408x768 image for a 16:9 request: nothing about the requested shape ever
// reached the wire, so the model fell back to its own default. The MCP layer
// resolves the caller's ratio into pixels for pricing, so the provider maps
// those pixels back onto the label the API accepts.
func TestBuildGenerateConfig_SendsAspectRatio(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		want          string
	}{
		{"16:9 draft", 768, 432, "16:9"},
		{"21:9 ultrawide", 768, 329, "21:9"},
		{"square", 768, 768, "1:1"},
		{"9:16 portrait", 432, 768, "9:16"},
		{"4:3 classic", 768, 576, "4:3"},
		{"3:2 photo", 768, 512, "3:2"},
		{"unset dimensions fall back to 16:9", 0, 0, "16:9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := draftReq(1)
			req.Width, req.Height = tt.width, tt.height

			cfg := buildGenerateConfig(req)
			if cfg.ImageConfig == nil {
				t.Fatal("ImageConfig = nil, want the requested shape on the wire")
			}
			if cfg.ImageConfig.AspectRatio != tt.want {
				t.Errorf("AspectRatio for %dx%d = %q, want %q",
					tt.width, tt.height, cfg.ImageConfig.AspectRatio, tt.want)
			}
		})
	}
}

// TestBuildGenerateConfig_SendsImageSize pins the resolution tier reaching the
// API, which was also never sent.
func TestBuildGenerateConfig_SendsImageSize(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		want          string
	}{
		{"768px draft is 1K", 768, 432, "1K"},
		{"exactly 1024 is still 1K", 1024, 576, "1K"},
		{"1536px is 2K", 1536, 864, "2K"},
		{"exactly 2048 is still 2K", 2048, 1152, "2K"},
		{"3840px is 4K", 3840, 2160, "4K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := draftReq(1)
			req.Width, req.Height = tt.width, tt.height

			cfg := buildGenerateConfig(req)
			if cfg.ImageConfig == nil {
				t.Fatal("ImageConfig = nil, want the resolution tier on the wire")
			}
			if cfg.ImageConfig.ImageSize != tt.want {
				t.Errorf("ImageSize for %dx%d = %q, want %q",
					tt.width, tt.height, cfg.ImageConfig.ImageSize, tt.want)
			}
		})
	}
}

// TestImageSizeMatchesPricingTier is the drift guard. The tier we ask the API
// for and the tier we price the call at come from one function on purpose:
// requesting 2K while charging the 1K rate would under-settle the budget.
func TestImageSizeMatchesPricingTier(t *testing.T) {
	for _, dims := range [][2]int{{768, 432}, {1024, 576}, {1536, 864}, {2048, 1152}, {3840, 2160}} {
		req := draftReq(1)
		req.Width, req.Height = dims[0], dims[1]

		cfg := buildGenerateConfig(req)
		want := strings.ToUpper(resolutionTier(req.Width, req.Height))

		if cfg.ImageConfig.ImageSize != want {
			t.Errorf("for %dx%d the wire asks %q but pricing uses tier %q",
				dims[0], dims[1], cfg.ImageConfig.ImageSize, want)
		}
	}
}

func TestWorkerSeed_Offsetting(t *testing.T) {
	// Nil seed preserves nil across all workers
	if got := WorkerSeed(nil, 0); got != nil {
		t.Errorf("WorkerSeed(nil, 0) = %v, want nil", got)
	}
	if got := WorkerSeed(nil, 3); got != nil {
		t.Errorf("WorkerSeed(nil, 3) = %v, want nil", got)
	}

	// Base seed is correctly offset by worker index
	base := int64(100)
	for i := 0; i < 4; i++ {
		got := WorkerSeed(&base, i)
		if got == nil {
			t.Fatalf("WorkerSeed(&100, %d) returned nil", i)
		}
		want := int64(100 + i)
		if *got != want {
			t.Errorf("WorkerSeed(&100, %d) = %d, want %d", i, *got, want)
		}
	}
}
