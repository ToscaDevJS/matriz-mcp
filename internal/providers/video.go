package providers

import (
	"context"
	"time"

	"github.com/toscodevjs/matriz/internal/core"
)

// VideoJobStatus describes the execution state of an asynchronous video job.
type VideoJobStatus string

const (
	VideoJobPending    VideoJobStatus = "pending"
	VideoJobProcessing VideoJobStatus = "processing"
	VideoJobCompleted  VideoJobStatus = "completed"
	VideoJobFailed     VideoJobStatus = "failed"
	VideoJobCancelled  VideoJobStatus = "cancelled"
)

// VideoRequest carries input parameters for Text-to-Video and Image-to-Video tasks.
type VideoRequest struct {
	Prompt         string
	NegativePrompt string
	SourceImage    []byte         // nil for Text-to-Video; populated for Image-to-Video
	SourceRef      *core.AssetRef // Provenance tracking for DerivedFrom
	Width          int
	Height         int
	AspectRatio    string         // "16:9", "9:16", "1:1"
	DurationSec    float64        // Duration in seconds (e.g. 4.0, 5.0, 8.0)
	FPS            int            // 24 or 30
	Seed           *int64
	Model          string
}

// VideoJob represents the metadata and progress of an active or finished video generation job.
type VideoJob struct {
	ID               string         `json:"id"`
	Provider         string         `json:"provider"`
	Model            string         `json:"model"`
	Status           VideoJobStatus `json:"status"`
	ProgressPct      int            `json:"progress_percent"`
	CreatedAt        time.Time      `json:"created_at"`
	EstimatedCostUSD float64        `json:"estimated_cost_usd"`
	Error            string         `json:"error,omitempty"`
}

// VideoResult contains the rendered video bytes, optional poster thumbnail, and metrics.
type VideoResult struct {
	VideoBytes []byte
	PosterPNG  []byte // Extracted or provided poster keyframe (max edge <= 512px)
	MIMEType   string // "video/mp4" or "video/webm"
	Duration   float64
	FPS        int
	Width      int
	Height     int
	Seed       int64
	Model      string
	CostUSD    float64
}

// VideoProvider orchestrates asynchronous video generation workflows.
type VideoProvider interface {
	Name() string
	Capabilities() []Capability
	EstimateVideoCostUSD(req VideoRequest) float64
	StartVideo(ctx context.Context, req VideoRequest) (*VideoJob, error)
	PollVideo(ctx context.Context, jobID string) (*VideoJob, *VideoResult, error)
	CancelVideo(ctx context.Context, jobID string) error
}

// BuildVideoSidecar constructs a sidecar struct conforming to matriz.sidecar/v1.
func BuildVideoSidecar(ref core.AssetRef, providerName string, res *VideoResult, req VideoRequest) *core.Sidecar {
	seeded := (req.Seed != nil && res.Seed != 0)
	params := map[string]any{
		"seeded":       seeded,
		"duration_sec": res.Duration,
	}

	if res.Width > 0 && res.Height > 0 {
		params["width"] = res.Width
		params["height"] = res.Height
	}
	if res.FPS > 0 {
		params["fps"] = res.FPS
	}
	if req.AspectRatio != "" {
		params["aspect_ratio"] = req.AspectRatio
	}

	sidecar := &core.Sidecar{
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

	if req.SourceRef != nil && *req.SourceRef != "" {
		derived := *req.SourceRef
		sidecar.DerivedFrom = &derived
	}

	return sidecar
}
