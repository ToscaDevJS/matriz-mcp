package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toscodevjs/matriz/internal/core"
)

// TestT02_ResolveRef_PathTraversal verifies T-02: ResolveRef enforces path traversal security.
func TestT02_ResolveRef_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	if eval, err := filepath.EvalSymlinks(tempDir); err == nil {
		tempDir = eval
	}

	// Create valid inner file
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets dir: %v", err)
	}
	validFile := filepath.Join(assetsDir, "a.png")
	if err := os.WriteFile(validFile, []byte("fake-png"), 0644); err != nil {
		t.Fatalf("failed to write valid test file: %v", err)
	}

	// Create a file outside the root and a symlink pointing to it
	outsideDir := t.TempDir()
	secretFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}
	symlinkPath := filepath.Join(assetsDir, "escape-link.png")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Fatalf("failed to create escape symlink: %v", err)
	}

	tests := []struct {
		name      string
		ref       core.AssetRef
		wantError bool
		errSubstr string
	}{
		{
			name:      "valid asset ref",
			ref:       core.AssetRef("assets/a.png"),
			wantError: false,
		},
		{
			name:      "empty ref rejected",
			ref:       core.AssetRef(""),
			wantError: true,
			errSubstr: "paths must stay inside the project root",
		},
		{
			name:      "absolute path rejected",
			ref:       core.AssetRef("/etc/passwd"),
			wantError: true,
			errSubstr: "paths must stay inside the project root",
		},
		{
			name:      "parent traversal rejected",
			ref:       core.AssetRef("../etc/passwd"),
			wantError: true,
			errSubstr: "paths must stay inside the project root",
		},
		{
			name:      "nested traversal rejected",
			ref:       core.AssetRef("assets/../../etc/passwd"),
			wantError: true,
			errSubstr: "paths must stay inside the project root",
		},
		{
			name:      "escaping symlink rejected",
			ref:       core.AssetRef("assets/escape-link.png"),
			wantError: true,
			errSubstr: "paths must stay inside the project root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := core.ResolveRef(tempDir, tt.ref)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error for ref %q, got resolved path %q", tt.ref, resolved)
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain expected substring %q", err.Error(), tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for ref %q: %v", tt.ref, err)
				}
				expectedPath := filepath.Join(tempDir, string(tt.ref))
				if resolved != expectedPath {
					t.Errorf("expected %q, got %q", expectedPath, resolved)
				}
			}
		})
	}
}
