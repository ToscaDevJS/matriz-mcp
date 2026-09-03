package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

// Injected via ldflags during build.
var (
	Version   = "v0.2.0"
	GitCommit = "dev"
	BuildDate = "unknown"
)

// Info encapsulates system and build metadata.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetInfo returns the current build and runtime version info.
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns human-readable version details.
func (i Info) String() string {
	return fmt.Sprintf("matriz %s (%s %s) %s", i.Version, i.GitCommit, i.BuildDate, i.Platform)
}

// JSON returns formatted JSON representation.
func (i Info) JSON() (string, error) {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
