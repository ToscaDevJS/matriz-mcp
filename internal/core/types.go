package core

import "time"

// Origin describes the provenance of an asset.
type Origin string

const (
	OriginClient    Origin = "client"    // uploaded by the client; never modified in place
	OriginGenerated Origin = "generated" // produced by a generative model
	OriginDerived   Origin = "derived"   // deterministic transform of another asset
)

// Dimensions stores the pixel width and height of an asset.
type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// AssetRef is the stable handle passed across the system. It is always a
// project-relative path, never an absolute one (hard rule 7.4).
type AssetRef string

// Asset represents a managed image asset within a project.
type Asset struct {
	Ref      AssetRef   `json:"ref"`
	Origin   Origin     `json:"origin"`
	MIMEType string     `json:"mime_type"`
	Dims     Dimensions `json:"dims"`
	Bytes    int64      `json:"bytes"`
}

// SidecarSchema is the schema identifier for metadata sidecars.
const SidecarSchema = "matriz.sidecar/v1"

// Sidecar stores reproducibility and audit metadata alongside an image.
type Sidecar struct {
	Schema         string         `json:"schema"`
	Ref            AssetRef       `json:"ref"`
	Origin         Origin         `json:"origin"`
	CreatedAt      time.Time      `json:"created_at"`
	Provider       string         `json:"provider,omitempty"`
	Model          string         `json:"model,omitempty"`
	Prompt         string         `json:"prompt,omitempty"`
	NegativePrompt string         `json:"negative_prompt,omitempty"`
	Seed           int64          `json:"seed"`
	Params         map[string]any `json:"params,omitempty"`
	CostUSD        float64        `json:"cost_usd"`
	DerivedFrom    *AssetRef      `json:"derived_from,omitempty"`
}
