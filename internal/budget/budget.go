package budget

import (
	"fmt"
	"sync"
	"time"

	"github.com/toscodevjs/matriz/internal/core"
)

// Guard enforces a per-session spending and call ceiling. It is consulted BEFORE
// any paid generative call and fails closed: unknown costs count as the configured
// worst-case, never as zero.
type Guard struct {
	mu           sync.Mutex
	limitUSD     float64
	spentUSD     float64
	reservedUSD  float64
	calls        int
	inFlight     int
	maxCalls     int
	reservations map[string]float64
	seq          int64
}

// NewGuard creates a Guard with spending limit in USD and max allowed generative calls.
func NewGuard(limitUSD float64, maxCalls int) *Guard {
	return &Guard{
		limitUSD:     limitUSD,
		maxCalls:     maxCalls,
		reservations: make(map[string]float64),
	}
}

// Reserve returns an error if the estimated cost would exceed the ceiling or
// if the maximum number of generative calls has been reached.
func (g *Guard) Reserve(estimateUSD float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.calls+g.inFlight >= g.maxCalls {
		return fmt.Errorf("%w: max calls reached (%d of %d allowed across session)", core.ErrBudgetExhausted, g.calls+g.inFlight, g.maxCalls)
	}

	if g.spentUSD+g.reservedUSD+estimateUSD > g.limitUSD {
		return fmt.Errorf("%w: budget exhausted: spent $%.2f of $%.2f (estimate: $%.2f) across %d calls. "+
			"Raise MATRIZ_BUDGET_USD and restart the server, or use img_transform (free) if the change you want is deterministic",
			core.ErrBudgetExhausted, g.spentUSD, g.limitUSD, estimateUSD, g.calls)
	}

	return nil
}

// ReserveTicket locks funds for an in-flight asynchronous job and returns a ticket ID.
func (g *Guard) ReserveTicket(estimateUSD float64) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.calls+g.inFlight >= g.maxCalls {
		return "", fmt.Errorf("%w: max calls reached (%d of %d allowed across session)", core.ErrBudgetExhausted, g.calls+g.inFlight, g.maxCalls)
	}

	if g.spentUSD+g.reservedUSD+estimateUSD > g.limitUSD {
		return "", fmt.Errorf("%w: budget exhausted: spent $%.2f + reserved $%.2f of $%.2f (estimate: $%.2f) across %d calls",
			core.ErrBudgetExhausted, g.spentUSD, g.reservedUSD, g.limitUSD, estimateUSD, g.calls)
	}

	if g.reservations == nil {
		g.reservations = make(map[string]float64)
	}

	g.seq++
	ticket := fmt.Sprintf("ticket-%d-%d", time.Now().UnixNano(), g.seq)
	g.reservations[ticket] = estimateUSD
	g.reservedUSD += estimateUSD
	g.inFlight++
	return ticket, nil
}

// CommitTicket settles a previously reserved ticket with actual spend and increments completed calls.
func (g *Guard) CommitTicket(ticketID string, actualUSD float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	hold, ok := g.reservations[ticketID]
	if !ok {
		return fmt.Errorf("reservation ticket %q not found or already settled", ticketID)
	}

	delete(g.reservations, ticketID)
	g.reservedUSD -= hold
	g.inFlight--
	g.spentUSD += actualUSD
	g.calls++
	return nil
}

// ReleaseTicket cancels a reservation hold on failure or cancellation without recording spend.
func (g *Guard) ReleaseTicket(ticketID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	hold, ok := g.reservations[ticketID]
	if !ok {
		return fmt.Errorf("reservation ticket %q not found or already settled", ticketID)
	}

	delete(g.reservations, ticketID)
	g.reservedUSD -= hold
	g.inFlight--
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

// ReservedUSD returns the total amount currently locked by in-flight jobs.
func (g *Guard) ReservedUSD() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.reservedUSD
}

// LimitUSD returns the configured session budget limit.
func (g *Guard) LimitUSD() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.limitUSD
}

// BudgetLeft returns remaining spending capacity in USD taking into account holds.
func (g *Guard) BudgetLeft() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	left := g.limitUSD - (g.spentUSD + g.reservedUSD)
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

// InFlight returns the count of active in-flight jobs.
func (g *Guard) InFlight() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.inFlight
}

// MaxCalls returns the maximum permitted generative calls.
func (g *Guard) MaxCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.maxCalls
}
