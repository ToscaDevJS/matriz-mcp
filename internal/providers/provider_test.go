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
	})

	if sidecar.Seed != 0 {
		t.Errorf("expected seed 0, got %d", sidecar.Seed)
	}
	if seeded, ok := sidecar.Params["seeded"].(bool); !ok || seeded {
		t.Errorf("expected params.seeded to be false, got %v", sidecar.Params["seeded"])
	}
}
