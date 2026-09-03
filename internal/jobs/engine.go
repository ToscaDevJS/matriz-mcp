package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/toscodevjs/matriz/internal/budget"
	"github.com/toscodevjs/matriz/internal/core"
	"github.com/toscodevjs/matriz/internal/providers"
)

// JobRecord represents the state and result of an asynchronous video job.
type JobRecord struct {
	ID               string                   `json:"id"`
	Provider         string                   `json:"provider"`
	Model            string                   `json:"model"`
	Prompt           string                   `json:"prompt"`
	NegativePrompt   string                   `json:"negative_prompt,omitempty"`
	SourceRef        *core.AssetRef           `json:"source_ref,omitempty"`
	DurationSec      float64                  `json:"duration_seconds"`
	FPS              int                      `json:"fps"`
	AspectRatio      string                   `json:"aspect_ratio"`
	OutputRef        core.AssetRef            `json:"output_ref"`
	TicketID         string                   `json:"ticket_id"`
	Status           providers.VideoJobStatus `json:"status"`
	ProgressPct      int                      `json:"progress_percent"`
	ETASeconds       int                      `json:"eta_seconds"`
	CreatedAt        time.Time                `json:"created_at"`
	EstimatedCostUSD float64                  `json:"estimated_cost_usd"`
	ActualCostUSD    float64                  `json:"actual_cost_usd,omitempty"`
	Result           *providers.VideoResult   `json:"-"`
	Asset            *core.Asset              `json:"asset,omitempty"`
	PosterBytes      []byte                   `json:"-"`
	Error            string                   `json:"error,omitempty"`

	mu       sync.RWMutex
	doneChan chan struct{}
}

// Engine manages the lifecycle of asynchronous video generation jobs.
type Engine struct {
	projectRoot string
	reg         *providers.Registry
	guard       *budget.Guard

	mu   sync.RWMutex
	jobs map[string]*JobRecord
}

// NewEngine creates a new background job engine.
func NewEngine(projectRoot string, reg *providers.Registry, guard *budget.Guard) *Engine {
	return &Engine{
		projectRoot: projectRoot,
		reg:         reg,
		guard:       guard,
		jobs:        make(map[string]*JobRecord),
	}
}

// StartJob creates and launches an asynchronous video generation workflow.
func (e *Engine) StartJob(
	ctx context.Context,
	vp providers.VideoProvider,
	req providers.VideoRequest,
	outputRef core.AssetRef,
	ticketID string,
) (*JobRecord, error) {
	vJob, err := vp.StartVideo(ctx, req)
	if err != nil {
		return nil, err
	}

	record := &JobRecord{
		ID:               vJob.ID,
		Provider:         vp.Name(),
		Model:            vJob.Model,
		Prompt:           req.Prompt,
		NegativePrompt:   req.NegativePrompt,
		SourceRef:        req.SourceRef,
		DurationSec:      req.DurationSec,
		FPS:              req.FPS,
		AspectRatio:      req.AspectRatio,
		OutputRef:        outputRef,
		TicketID:         ticketID,
		Status:           providers.VideoJobProcessing,
		ProgressPct:      vJob.ProgressPct,
		ETASeconds:       int(req.DurationSec * 12), // Rough initial ETA
		CreatedAt:        time.Now().UTC(),
		EstimatedCostUSD: vJob.EstimatedCostUSD,
		doneChan:         make(chan struct{}),
	}

	if record.ProgressPct <= 0 {
		record.ProgressPct = 10
	}
	if record.ETASeconds <= 0 {
		record.ETASeconds = 45
	}

	e.mu.Lock()
	e.jobs[record.ID] = record
	e.mu.Unlock()

	// Launch background polling worker
	go e.pollWorker(vp, record, req)

	return record, nil
}

func (e *Engine) pollWorker(vp providers.VideoProvider, rec *JobRecord, req providers.VideoRequest) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-rec.doneChan:
			return
		case <-ticker.C:
			rec.mu.RLock()
			status := rec.Status
			rec.mu.RUnlock()

			if status == providers.VideoJobCompleted || status == providers.VideoJobFailed || status == providers.VideoJobCancelled {
				return
			}

			job, res, err := vp.PollVideo(context.Background(), rec.ID)
			if err != nil {
				e.failJob(rec, err.Error())
				return
			}

			if job.Status == providers.VideoJobCompleted && res != nil {
				e.completeJob(rec, res, req)
				return
			}

			if job.Status == providers.VideoJobFailed {
				e.failJob(rec, job.Error)
				return
			}

			// Update progress
			elapsed := int(time.Since(startTime).Seconds())
			rec.mu.Lock()
			if job.ProgressPct > rec.ProgressPct {
				rec.ProgressPct = job.ProgressPct
			} else if rec.ProgressPct < 90 {
				rec.ProgressPct += 5
			}
			if rec.ETASeconds > elapsed {
				rec.ETASeconds -= 2
				if rec.ETASeconds < 5 {
					rec.ETASeconds = 5
				}
			}
			rec.mu.Unlock()
		}
	}
}

func (e *Engine) completeJob(rec *JobRecord, res *providers.VideoResult, req providers.VideoRequest) {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	// Commit budget ticket
	if rec.TicketID != "" && e.guard != nil {
		_ = e.guard.CommitTicket(rec.TicketID, res.CostUSD)
	}

	// Write MP4 video file to destination
	outPath, err := core.ResolveRef(e.projectRoot, rec.OutputRef)
	if err == nil && len(res.VideoBytes) > 0 {
		_ = os.MkdirAll(filepath.Dir(outPath), 0755)
		_ = os.WriteFile(outPath, res.VideoBytes, 0644)
	}

	// Build and write sidecar
	sidecar := providers.BuildVideoSidecar(rec.OutputRef, rec.Provider, res, req)
	if err == nil {
		_ = core.WriteSidecar(outPath, sidecar)
	}

	rec.Status = providers.VideoJobCompleted
	rec.ProgressPct = 100
	rec.ETASeconds = 0
	rec.ActualCostUSD = res.CostUSD
	rec.Result = res
	rec.PosterBytes = res.PosterPNG

	// If no poster is provided but input image is available, use input image as poster
	if len(rec.PosterBytes) == 0 && len(req.SourceImage) > 0 {
		rec.PosterBytes = req.SourceImage
	}

	rec.Asset = &core.Asset{
		Ref:      rec.OutputRef,
		Origin:   core.OriginGenerated,
		MIMEType: res.MIMEType,
		Dims: core.Dimensions{
			Width:  res.Width,
			Height: res.Height,
		},
		Bytes:    int64(len(res.VideoBytes)),
		Duration: res.Duration,
	}

	select {
	case <-rec.doneChan:
	default:
		close(rec.doneChan)
	}
}

func (e *Engine) failJob(rec *JobRecord, errMsg string) {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	if rec.TicketID != "" && e.guard != nil {
		_ = e.guard.ReleaseTicket(rec.TicketID)
	}

	rec.Status = providers.VideoJobFailed
	rec.Error = errMsg

	select {
	case <-rec.doneChan:
	default:
		close(rec.doneChan)
	}
}

// GetJob returns a copy of the job record.
func (e *Engine) GetJob(jobID string) (*JobRecord, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rec, ok := e.jobs[jobID]
	if !ok {
		return nil, false
	}

	rec.mu.RLock()
	defer rec.mu.RUnlock()
	copied := *rec
	return &copied, true
}

// WaitForJob applies bounded smart-wait on an in-flight job.
func (e *Engine) WaitForJob(ctx context.Context, jobID string, waitDuration time.Duration) (*JobRecord, error) {
	e.mu.RLock()
	rec, ok := e.jobs[jobID]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("job %q not found", jobID)
	}

	rec.mu.RLock()
	status := rec.Status
	rec.mu.RUnlock()

	if status == providers.VideoJobCompleted || status == providers.VideoJobFailed || status == providers.VideoJobCancelled {
		rec.mu.RLock()
		defer rec.mu.RUnlock()
		cp := *rec
		return &cp, nil
	}

	if waitDuration <= 0 {
		rec.mu.RLock()
		defer rec.mu.RUnlock()
		cp := *rec
		return &cp, nil
	}

	timer := time.NewTimer(waitDuration)
	defer timer.Stop()

	select {
	case <-rec.doneChan:
	case <-timer.C:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	rec.mu.RLock()
	defer rec.mu.RUnlock()
	cp := *rec
	return &cp, nil
}

// CancelJob cancels an in-flight job and releases its budget hold.
func (e *Engine) CancelJob(ctx context.Context, jobID string) (*JobRecord, error) {
	e.mu.RLock()
	rec, ok := e.jobs[jobID]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("job %q not found", jobID)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()

	if rec.Status == providers.VideoJobCompleted || rec.Status == providers.VideoJobCancelled || rec.Status == providers.VideoJobFailed {
		cp := *rec
		return &cp, nil
	}

	if rec.TicketID != "" && e.guard != nil {
		_ = e.guard.ReleaseTicket(rec.TicketID)
	}

	rec.Status = providers.VideoJobCancelled

	// Notify provider
	if vp, err := e.reg.GetVideo(rec.Provider); err == nil {
		_ = vp.CancelVideo(ctx, jobID)
	}

	select {
	case <-rec.doneChan:
	default:
		close(rec.doneChan)
	}

	cp := *rec
	return &cp, nil
}

// ListJobs returns all jobs tracked by the engine.
func (e *Engine) ListJobs() []*JobRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	list := make([]*JobRecord, 0, len(e.jobs))
	for _, rec := range e.jobs {
		rec.mu.RLock()
		cp := *rec
		rec.mu.RUnlock()
		list = append(list, &cp)
	}
	return list
}
