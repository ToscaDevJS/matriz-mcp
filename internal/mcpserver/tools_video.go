package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/config"
	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/jobs"
	"github.com/toscodevjs/matriz/internal/providers"
)

// VideoGenerateIn defines the parameters for video generation.
type VideoGenerateIn struct {
	Prompt         string  `json:"prompt" jsonschema:"detailed visual motion prompt"`
	Ref            string  `json:"ref,omitempty" jsonschema:"optional project-relative path to anchor image asset for Image-to-Video"`
	NegativePrompt string  `json:"negative_prompt,omitempty" jsonschema:"elements or styles to avoid"`
	DurationSec    float64 `json:"duration_seconds,omitempty" jsonschema:"video duration in seconds (4.0 to 10.0),default=5.0"`
	AspectRatio    string  `json:"aspect_ratio,omitempty" jsonschema:"aspect ratio (16:9, 9:16, 1:1),default=16:9"`
	ModelTier      string  `json:"model_tier,omitempty" jsonschema:"model tier ('draft' for rapid preview, 'final' for cinematic quality),enum=draft,enum=final,default=draft"`
	Output         string  `json:"output,omitempty" jsonschema:"optional project-relative destination path for the video"`
	Seed           *int64  `json:"seed,omitempty" jsonschema:"optional deterministic seed"`
}

// VideoGenerateOut defines the response returned immediately upon job creation.
type VideoGenerateOut struct {
	JobID             string  `json:"job_id"`
	Status            string  `json:"status"` // "processing"
	EstimatedDuration int     `json:"estimated_duration_seconds"`
	PollIntervalSec   int     `json:"poll_interval_seconds"`
	ReservedCostUSD   float64 `json:"reserved_cost_usd"`
	BudgetLeftUSD     float64 `json:"budget_left_usd"`
	Directive         string  `json:"directive"`
}

// VideoStatusIn defines the parameters for querying an active or completed job.
type VideoStatusIn struct {
	JobID       string `json:"job_id" jsonschema:"unique job identifier returned by video_generate"`
	WaitSeconds int    `json:"wait_seconds,omitempty" jsonschema:"seconds to hold response if still processing (0 to 10),default=5"`
}

// VideoStatusOut defines the response returned for a status query.
type VideoStatusOut struct {
	JobID         string      `json:"job_id"`
	Status        string      `json:"status"` // "processing" | "completed" | "failed" | "cancelled"
	ProgressPct   int         `json:"progress_percent"`
	ETASeconds    int         `json:"eta_seconds,omitempty"`
	ActualCostUSD float64     `json:"actual_cost_usd,omitempty"`
	Asset         *core.Asset `json:"asset,omitempty"`
	Directive     string      `json:"directive,omitempty"`
	Error         string      `json:"error,omitempty"`
}

// VideoCancelIn defines parameters for cancelling a job.
type VideoCancelIn struct {
	JobID string `json:"job_id" jsonschema:"unique job identifier to cancel"`
}

// VideoCancelOut defines response for job cancellation.
type VideoCancelOut struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"` // "cancelled"
	Message string `json:"message"`
}

func handleVideoGenerate(cfg *config.Config, reg *providers.Registry, guard *budget.Guard, engine *jobs.Engine) mcp.ToolHandlerFor[VideoGenerateIn, VideoGenerateOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in VideoGenerateIn) (*mcp.CallToolResult, VideoGenerateOut, error) {
		vp, err := reg.GetVideo(cfg.Provider)
		if err != nil {
			return toolError(err, "Configured provider does not support video capabilities."), VideoGenerateOut{}, nil
		}

		// Resolve duration and aspect ratio defaults
		dur := in.DurationSec
		if dur <= 0 {
			dur = 5.0
		}
		ar := in.AspectRatio
		if ar == "" {
			ar = "16:9"
		}

		// Model selection
		modelID := cfg.ModelVideoDraft
		if in.ModelTier == "final" {
			modelID = cfg.ModelVideoFinal
		}

		videoReq := providers.VideoRequest{
			Prompt:         in.Prompt,
			NegativePrompt: in.NegativePrompt,
			AspectRatio:    ar,
			DurationSec:    dur,
			FPS:            24,
			Seed:           in.Seed,
			Model:          modelID,
		}

		// Image-to-Video resolution
		if in.Ref != "" {
			srcRef := core.AssetRef(in.Ref)
			srcPath, err := core.ResolveRef(cfg.ProjectRoot, srcRef)
			if err != nil {
				return toolError(err, "Invalid source ref path for Image-to-Video."), VideoGenerateOut{}, nil
			}
			imgBytes, err := os.ReadFile(srcPath)
			if err != nil {
				return toolError(err, "Source asset could not be read."), VideoGenerateOut{}, nil
			}
			videoReq.SourceImage = imgBytes
			videoReq.SourceRef = &srcRef
		}

		// Destination path
		outRef := core.AssetRef(in.Output)
		if outRef == "" {
			outRef = core.AssetRef(fmt.Sprintf("assets/videos/video-%d.mp4", time.Now().UnixNano()))
		}
		if _, err := core.ResolveRef(cfg.ProjectRoot, outRef); err != nil {
			return toolError(err, "Invalid destination output path."), VideoGenerateOut{}, nil
		}

		// Two-phase budget reservation
		estCost := vp.EstimateVideoCostUSD(videoReq)
		ticketID, err := guard.ReserveTicket(estCost)
		if err != nil {
			return toolError(err, "Session budget exhausted for video generation."), VideoGenerateOut{}, nil
		}

		// Dispatch to engine
		jobRec, err := engine.StartJob(ctx, vp, videoReq, outRef, ticketID)
		if err != nil {
			_ = guard.ReleaseTicket(ticketID)
			return toolError(err, "Failed to start video generation job."), VideoGenerateOut{}, nil
		}

		out := VideoGenerateOut{
			JobID:             jobRec.ID,
			Status:            string(jobRec.Status),
			EstimatedDuration: jobRec.ETASeconds,
			PollIntervalSec:   10,
			ReservedCostUSD:   estCost,
			BudgetLeftUSD:     guard.BudgetLeft(),
			Directive: fmt.Sprintf("Job %s started. Video generation takes ~%ds. DO NOT poll in a tight loop. "+
				"Check back with video_status after 10-15 seconds or continue other work.", jobRec.ID, jobRec.ETASeconds),
		}

		return &mcp.CallToolResult{}, out, nil
	}
}

func handleVideoStatus(cfg *config.Config, engine *jobs.Engine) mcp.ToolHandlerFor[VideoStatusIn, VideoStatusOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in VideoStatusIn) (*mcp.CallToolResult, VideoStatusOut, error) {
		waitSec := in.WaitSeconds
		if waitSec < 0 {
			waitSec = 0
		} else if waitSec > 10 {
			waitSec = 10
		}
		if waitSec == 0 && in.WaitSeconds == 0 {
			waitSec = 5 // default 5s smart wait
		}

		rec, err := engine.WaitForJob(ctx, in.JobID, time.Duration(waitSec)*time.Second)
		if err != nil {
			return toolError(err, "Job lookup failed."), VideoStatusOut{}, nil
		}

		out := VideoStatusOut{
			JobID:         rec.ID,
			Status:        string(rec.Status),
			ProgressPct:   rec.ProgressPct,
			ETASeconds:    rec.ETASeconds,
			ActualCostUSD: rec.ActualCostUSD,
			Asset:         rec.Asset,
			Error:         rec.Error,
		}

		var contentList []mcp.Content

		switch rec.Status {
		case providers.VideoJobProcessing:
			out.Directive = fmt.Sprintf("Video is still rendering (%d%%, ~%ds left). DO NOT poll immediately. "+
				"Inform the user or wait before calling video_status again.", rec.ProgressPct, rec.ETASeconds)
		case providers.VideoJobCompleted:
			out.Directive = "Video rendering complete. Poster thumbnail attached and asset saved to disk."
			if len(rec.PosterBytes) > 0 {
				if decoded, _, err := image.Decode(bytes.NewReader(rec.PosterBytes)); err == nil {
					if thumb, err := thumbnailContent(decoded, 512); err == nil && thumb != nil {
						contentList = append(contentList, thumb)
					}
				}
			}
		case providers.VideoJobFailed:
			out.Directive = fmt.Sprintf("Video rendering failed: %s. Reserved budget was released.", rec.Error)
		case providers.VideoJobCancelled:
			out.Directive = "Video job was cancelled. Reserved budget was released."
		}

		return &mcp.CallToolResult{
			Content: contentList,
		}, out, nil
	}
}

func handleVideoCancel(engine *jobs.Engine) mcp.ToolHandlerFor[VideoCancelIn, VideoCancelOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in VideoCancelIn) (*mcp.CallToolResult, VideoCancelOut, error) {
		rec, err := engine.CancelJob(ctx, in.JobID)
		if err != nil {
			return toolError(err, "Job cancellation failed."), VideoCancelOut{}, nil
		}

		return &mcp.CallToolResult{}, VideoCancelOut{
			JobID:   rec.ID,
			Status:  "cancelled",
			Message: fmt.Sprintf("Job %s cancelled. Reserved budget released.", rec.ID),
		}, nil
	}
}

// CallVideoGenerate executes video_generate handler directly for testing.
func CallVideoGenerate(ctx context.Context, cfg *config.Config, reg *providers.Registry, guard *budget.Guard, engine *jobs.Engine, in VideoGenerateIn) (*mcp.CallToolResult, error) {
	handler := handleVideoGenerate(cfg, reg, guard, engine)
	res, out, err := handler(ctx, &mcp.CallToolRequest{}, in)
	if err != nil {
		return res, err
	}
	if res == nil {
		res = &mcp.CallToolResult{}
	}
	if res.StructuredContent == nil {
		res.StructuredContent = out
	}
	return res, nil
}

// CallVideoStatus executes video_status handler directly for testing.
func CallVideoStatus(ctx context.Context, cfg *config.Config, engine *jobs.Engine, in VideoStatusIn) (*mcp.CallToolResult, error) {
	handler := handleVideoStatus(cfg, engine)
	res, out, err := handler(ctx, &mcp.CallToolRequest{}, in)
	if err != nil {
		return res, err
	}
	if res == nil {
		res = &mcp.CallToolResult{}
	}
	if res.StructuredContent == nil {
		res.StructuredContent = out
	}
	return res, nil
}

// CallVideoCancel executes video_cancel handler directly for testing.
func CallVideoCancel(ctx context.Context, engine *jobs.Engine, in VideoCancelIn) (*mcp.CallToolResult, error) {
	handler := handleVideoCancel(engine)
	res, out, err := handler(ctx, &mcp.CallToolRequest{}, in)
	if err != nil {
		return res, err
	}
	if res == nil {
		res = &mcp.CallToolResult{}
	}
	if res.StructuredContent == nil {
		res.StructuredContent = out
	}
	return res, nil
}
