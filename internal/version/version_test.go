package version_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/toscodevjs/matriz/internal/version"
)

func TestVersion_String(t *testing.T) {
	info := version.GetInfo()
	if info.Version == "" {
		t.Fatalf("expected non-empty Version")
	}

	str := info.String()
	if !strings.Contains(str, "matriz") || !strings.Contains(str, info.Version) {
		t.Errorf("expected string to contain matriz and version, got %q", str)
	}
}

func TestVersion_JSON(t *testing.T) {
	info := version.GetInfo()
	jsonStr, err := info.JSON()
	if err != nil {
		t.Fatalf("JSON formatting failed: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}

	if parsed["version"] != info.Version {
		t.Errorf("expected version %q, got %q", info.Version, parsed["version"])
	}
	if parsed["go_version"] == "" {
		t.Errorf("expected non-empty go_version")
	}
	if parsed["platform"] == "" {
		t.Errorf("expected non-empty platform")
	}
}
