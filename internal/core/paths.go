package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRef converts a project-relative AssetRef into an absolute filesystem
// path, and refuses anything that escapes the project root.
//
// This runs on EVERY ref that arrives from the MCP boundary, including refs
// that came back from a previous tool call. An LLM-supplied path is untrusted input.
func ResolveRef(projectRoot string, ref AssetRef) (string, error) {
	rawRef := string(ref)
	if strings.TrimSpace(rawRef) == "" {
		return "", fmt.Errorf("%w: invalid asset ref %q: paths must stay inside the project root", ErrInvalidAssetRef, rawRef)
	}

	// Reject absolute paths directly
	if filepath.IsAbs(rawRef) || strings.HasPrefix(rawRef, "/") || strings.HasPrefix(rawRef, "\\") {
		return "", fmt.Errorf("%w: invalid asset ref %q: paths must stay inside the project root", ErrInvalidAssetRef, rawRef)
	}

	// Clean and join
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("%w: failed to resolve project root: %v", ErrInvalidAssetRef, err)
	}

	// Eval symlinks on root itself if it exists
	if evalRoot, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = evalRoot
	}

	targetPath := filepath.Clean(filepath.Join(absRoot, rawRef))

	// Check if target path starts with project root
	rel, err := filepath.Rel(absRoot, targetPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: invalid asset ref %q: paths must stay inside the project root", ErrInvalidAssetRef, rawRef)
	}

	// Check for symlinks if target exists or evaluate existing ancestors
	if fi, err := os.Lstat(targetPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 || true {
			evaluated, err := filepath.EvalSymlinks(targetPath)
			if err == nil {
				evalRel, err := filepath.Rel(absRoot, evaluated)
				if err != nil || evalRel == ".." || strings.HasPrefix(evalRel, ".."+string(filepath.Separator)) {
					return "", fmt.Errorf("%w: invalid asset ref %q: paths must stay inside the project root", ErrInvalidAssetRef, rawRef)
				}
			}
		}
	} else {
		// If the leaf doesn't exist yet (e.g. for write target), evaluate closest existing parent
		parent := filepath.Dir(targetPath)
		for parent != absRoot && len(parent) >= len(absRoot) {
			if _, err := os.Stat(parent); err == nil {
				evalParent, err := filepath.EvalSymlinks(parent)
				if err == nil {
					evalRel, err := filepath.Rel(absRoot, evalParent)
					if err != nil || evalRel == ".." || strings.HasPrefix(evalRel, ".."+string(filepath.Separator)) {
						return "", fmt.Errorf("%w: invalid asset ref %q: paths must stay inside the project root", ErrInvalidAssetRef, rawRef)
					}
				}
				break
			}
			nextParent := filepath.Dir(parent)
			if nextParent == parent {
				break
			}
			parent = nextParent
		}
	}

	return targetPath, nil
}
