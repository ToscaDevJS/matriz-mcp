package core

import "errors"

// Sentinel errors representing common domain failures.
var (
	ErrInvalidAssetRef     = errors.New("invalid asset ref")
	ErrAssetNotFound       = errors.New("asset not found")
	ErrAspectRatioMismatch = errors.New("aspect ratio mismatch")
	ErrProviderUnsupported = errors.New("provider capability unsupported")
	ErrBudgetExhausted     = errors.New("budget exhausted")
)
