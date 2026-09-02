package providers_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/providers"
	"github.com/toscodevjs/matriz/internal/providers/fake"
	"github.com/toscodevjs/matriz/internal/providers/gemini"
)

// TestT08_FakeProvider_DeterministicWithSeed verifies T-08:
// fakeProvider.Generate with the same seed returns identical bytes across two calls and counts invocations.
func TestT08_FakeProvider_DeterministicWithSeed(t *testing.T) {
	fp := fake.NewFakeProvider()
	seed := int64(12345)

	req := providers.GenerateRequest{
		Prompt: "a golden retriever in a salon",
		Width:  512,
		Height: 512,
		Count:  1,
		Seed:   &seed,
	}

	res1, err := fp.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("first generate failed: %v", err)
	}

	res2, err := fp.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("second generate failed: %v", err)
	}

	if fp.InvocationCount() != 2 {
		t.Fatalf("expected 2 invocations, got %d", fp.InvocationCount())
	}

	if len(res1.Images) != 1 || len(res2.Images) != 1 {
		t.Fatalf("expected 1 image in each result")
	}

	if !bytes.Equal(res1.Images[0], res2.Images[0]) {
		t.Fatalf("images generated with same seed must be identical")
	}
}

// TestT10b_EstimateCostUSD_UnknownModelFallback verifies T-10b:
// EstimateCostUSD of an unlisted model returns the worst-case known price, never 0.
func TestT10b_EstimateCostUSD_UnknownModelFallback(t *testing.T) {
	pricing := gemini.NewPricingTable()

	req := providers.GenerateRequest{
		Model:  "unknown-future-gemini-ultra",
		Width:  1920,
		Height: 1080,
		Count:  1,
	}

	cost := pricing.EstimateCostUSD(req)
	if cost <= 0 {
		t.Fatalf("expected worst-case cost > 0 for unknown model, got %f", cost)
	}

	worstCase := pricing.WorstCaseCostUSD(1)
	if cost != worstCase {
		t.Errorf("expected cost %f to equal worst case %f", cost, worstCase)
	}
}

// TestT10c_EstimateCostUSD_NoNetwork verifies T-10c:
// EstimateCostUSD does not perform any network calls.
func TestT10c_EstimateCostUSD_NoNetwork(t *testing.T) {
	// Custom transport that panics if any HTTP call is attempted
	originalTransport := http.DefaultTransport
	http.DefaultTransport = &failingTransport{t: t}
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	pricing := gemini.NewPricingTable()
	req := providers.GenerateRequest{
		Model:  "gemini-3.1-flash-lite-image",
		Width:  1024,
		Height: 1024,
		Count:  2,
	}

	cost := pricing.EstimateCostUSD(req)
	if cost <= 0 {
		t.Fatalf("expected positive estimated cost, got %f", cost)
	}
}

type failingTransport struct {
	t *testing.T
}

func (f *failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.t.Fatalf("network access prohibited in EstimateCostUSD: %s", req.URL)
	return nil, nil
}

// TestT10d_ResultWithoutSeed_ProducesUnseededSidecar verifies T-10d:
// A generative result without seed produces a sidecar with seed=0 and seeded=false.
func TestT10d_ResultWithoutSeed_ProducesUnseededSidecar(t *testing.T) {
	res := &providers.Result{
		Images:   [][]byte{[]byte("fake-image-bytes")},
		MIMEType: "image/png",
		Seed:     0, // no seed provided
		Model:    "gemini-3.1-flash-lite-image",
		CostUSD:  0.0336,
	}

	sidecar := providers.BuildSidecar(core.AssetRef("assets/gen.png"), "gemini", res, providers.GenerateRequest{
		Prompt: "a vintage barber chair",
		Width:  768,
		Height: 768,
		Seed:   nil,
	}, core.Dimensions{Width: 768, Height: 768})

	if sidecar.Seed != 0 {
		t.Errorf("expected seed 0, got %d", sidecar.Seed)
	}
	if seeded, ok := sidecar.Params["seeded"].(bool); !ok || seeded {
		t.Errorf("expected params.seeded to be false, got %v", sidecar.Params["seeded"])
	}
}

// TestBuildSidecar_AttributesPerImageCost covers batch cost attribution. A
// Result carries the settled cost of the WHOLE batch, but BuildSidecar writes
// one sidecar per image. Copying the batch total into each file would report
// four times the real cost of a four-draft batch, and the sidecar is the
// project's cost record (§5.4).
func TestBuildSidecar_AttributesPerImageCost(t *testing.T) {
	tests := []struct {
		name      string
		images    int
		batchCost float64
		want      float64
	}{
		{"four drafts share the batch cost", 4, 0.1344, 0.0336},
		{"single draft carries the whole cost", 1, 0.0336, 0.0336},
		{"two drafts", 2, 0.0672, 0.0336},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := make([][]byte, tt.images)
			for i := range images {
				images[i] = []byte("fake-image-bytes")
			}

			res := &providers.Result{
				Images:   images,
				MIMEType: "image/png",
				Model:    "gemini-3.1-flash-lite-image",
				CostUSD:  tt.batchCost,
			}

			sidecar := providers.BuildSidecar(
				core.AssetRef("assets/drafts/draft-1.png"), "gemini", res,
				providers.GenerateRequest{Prompt: "a barber chair", Count: tt.images},
				core.Dimensions{Width: 1408, Height: 768},
			)

			if sidecar.CostUSD != tt.want {
				t.Errorf("sidecar cost for 1 of %d images = %v, want %v",
					tt.images, sidecar.CostUSD, tt.want)
			}
		})
	}
}

// TestBuildSidecar_EmptyResultDoesNotPanic guards the degenerate case: a Result
// with no images must not divide by zero.
func TestBuildSidecar_EmptyResultDoesNotPanic(t *testing.T) {
	res := &providers.Result{
		Images:  nil,
		Model:   "gemini-3.1-flash-lite-image",
		CostUSD: 0.0336,
	}

	sidecar := providers.BuildSidecar(
		core.AssetRef("assets/drafts/draft-1.png"), "gemini", res,
		providers.GenerateRequest{Prompt: "a barber chair"},
		core.Dimensions{Width: 1408, Height: 768},
	)

	if sidecar.CostUSD != 0.0336 {
		t.Errorf("sidecar cost for an empty result = %v, want the unsplit cost 0.0336", sidecar.CostUSD)
	}
}

// TestBuildSidecar_RecordsProducedDimensions covers the reproducibility bug: the
// sidecar recorded the dimensions that were REQUESTED, not the ones the provider
// produced. Gemini answered a 768x432 request with a 1408x768 image, so every
// .meta.json on disk described a file that did not exist at that size. The
// sidecar is the reproducibility record (§5.4); it documents the result.
func TestBuildSidecar_RecordsProducedDimensions(t *testing.T) {
	res := &providers.Result{
		Images:   [][]byte{[]byte("fake-image-bytes")},
		MIMEType: "image/png",
		Model:    "gemini-3.1-flash-lite-image",
		CostUSD:  0.0336,
	}

	// Requested 768x432; the provider produced 1408x768.
	sidecar := providers.BuildSidecar(
		core.AssetRef("assets/drafts/draft-1.png"), "gemini", res,
		providers.GenerateRequest{Prompt: "a barber chair", Width: 768, Height: 432},
		core.Dimensions{Width: 1408, Height: 768},
	)

	if got := sidecar.Params["width"]; got != 1408 {
		t.Errorf("params.width = %v, want 1408 (produced), not 768 (requested)", got)
	}
	if got := sidecar.Params["height"]; got != 768 {
		t.Errorf("params.height = %v, want 768 (produced), not 432 (requested)", got)
	}
}

// TestBuildSidecar_OmitsUnknownDimensions pins the honest fallback: when the
// produced image could not be decoded, the sidecar records no dimensions rather
// than inventing the requested ones.
func TestBuildSidecar_OmitsUnknownDimensions(t *testing.T) {
	res := &providers.Result{
		Images:  [][]byte{[]byte("undecodable")},
		Model:   "gemini-3.1-flash-lite-image",
		CostUSD: 0.0336,
	}

	sidecar := providers.BuildSidecar(
		core.AssetRef("assets/drafts/draft-1.png"), "gemini", res,
		providers.GenerateRequest{Prompt: "a barber chair", Width: 768, Height: 432},
		core.Dimensions{},
	)

	if _, present := sidecar.Params["width"]; present {
		t.Errorf("params.width = %v, want the key absent when dimensions are unknown", sidecar.Params["width"])
	}
	if _, present := sidecar.Params["height"]; present {
		t.Errorf("params.height = %v, want the key absent when dimensions are unknown", sidecar.Params["height"])
	}
	if seeded, ok := sidecar.Params["seeded"].(bool); !ok || seeded {
		t.Error("params.seeded must survive even when dimensions are unknown")
	}
}
