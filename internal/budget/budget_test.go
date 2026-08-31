package budget_test

import (
	"strings"
	"testing"

	"github.com/toscodevjs/matriz/internal/budget"
)

// TestT09_Guard_Reserve_RejectsWhenExceeded verifies T-09:
// Guard.Reserve rejects when spent + estimate > limit, and the error contains limit, spent, and estimate.
func TestT09_Guard_Reserve_RejectsWhenExceeded(t *testing.T) {
	g := budget.NewGuard(2.00, 20)

	// Spend 1.90
	if err := g.Reserve(1.90); err != nil {
		t.Fatalf("unexpected reserve error: %v", err)
	}
	g.Commit(1.90)

	// Attempt to reserve 0.15 (total would be 2.05 > 2.00)
	err := g.Reserve(0.15)
	if err == nil {
		t.Fatalf("expected error when exceeding limit")
	}

	errMsg := err.Error()
	// Must contain spent (1.90), limit (2.00), and estimate (0.15) or numbers
	if !strings.Contains(errMsg, "1.90") && !strings.Contains(errMsg, "1.9") {
		t.Errorf("error message missing spent amount: %s", errMsg)
	}
	if !strings.Contains(errMsg, "2.00") && !strings.Contains(errMsg, "2") {
		t.Errorf("error message missing limit: %s", errMsg)
	}
	if !strings.Contains(errMsg, "0.15") {
		t.Errorf("error message missing estimate: %s", errMsg)
	}
}

// TestT10_Guard_RejectsMaxCalls verifies T-10:
// Guard rejects the (maxCalls+1)th call even when budget remains.
func TestT10_Guard_RejectsMaxCalls(t *testing.T) {
	g := budget.NewGuard(100.00, 3)

	for i := 1; i <= 3; i++ {
		if err := g.Reserve(0.10); err != nil {
			t.Fatalf("call %d unexpectedly failed: %v", i, err)
		}
		g.Commit(0.10)
	}

	// 4th call should be rejected
	err := g.Reserve(0.10)
	if err == nil {
		t.Fatalf("expected 4th call to be rejected by maxCalls limit")
	}
	if !strings.Contains(err.Error(), "max calls") && !strings.Contains(err.Error(), "3") {
		t.Errorf("error message should mention call limit: %s", err.Error())
	}
}
