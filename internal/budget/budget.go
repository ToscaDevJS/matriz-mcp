package budget

import (
	"fmt"
	"sync"

	"github.com/toscodevjs/matriz/internal/core"
)

// Guard enforces a per-session spending and call ceiling. It is consulted BEFORE
// any paid generative call and fails closed: unknown costs count as the configured
// worst-case, never as zero.
type Guard struct {
	mu       sync.Mutex
	limitUSD float64
	spentUSD float64
	calls    int
	maxCalls int
}

// NewGuard creates a Guard with spending limit in USD and max allowed generative calls.
func NewGuard(limitUSD float64, maxCalls int) *Guard {
	if limitUSD <= 0 {
		limitUSD = 2.00
	}
	if maxCalls <= 0 {
		maxCalls = 20
	}
	return &Guard{
		limitUSD: limitUSD,
		maxCalls: maxCalls,
	}
}

// Reserve returns an error if the estimated cost would exceed the ceiling or
// if the maximum number of generative calls has been reached.
func (g *Guard) Reserve(estimateUSD float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.calls >= g.maxCalls {
		return fmt.Errorf("%w: max calls reached (%d of %d allowed across session)", core.ErrBudgetExhausted, g.calls, g.maxCalls)
	}

	if g.spentUSD+estimateUSD > g.limitUSD {
		return fmt.Errorf("%w: budget exhausted: spent $%.2f of $%.2f (estimate: $%.2f) across %d calls. "+
			"Raise MATRIZ_BUDGET_USD and restart the server, or use img_transform (free) if the change you want is deterministic",
			core.ErrBudgetExhausted, g.spentUSD, g.limitUSD, estimateUSD, g.calls)
	}

	return nil
}

// Commit records actual spend and increments the call count after a successful generative call.
func (g *Guard) Commit(actualUSD float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.spentUSD += actualUSD
	g.calls++
}

// SpentUSD returns the total amount spent so far.
func (g *Guard) SpentUSD() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.spentUSD
}

// LimitUSD returns the configured session budget limit.
func (g *Guard) LimitUSD() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.limitUSD
}

// BudgetLeft returns remaining spending capacity in USD.
func (g *Guard) BudgetLeft() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	left := g.limitUSD - g.spentUSD
	if left < 0 {
		return 0
	}
	return left
}

// Calls returns the count of completed generative calls.
func (g *Guard) Calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

// MaxCalls returns the maximum permitted generative calls.
func (g *Guard) MaxCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.maxCalls
}
