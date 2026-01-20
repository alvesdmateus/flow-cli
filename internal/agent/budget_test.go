package agent

import (
	"sync"
	"testing"
)

func TestTokenBudget_NewTokenBudget(t *testing.T) {
	budget := NewTokenBudget(100000)

	if budget.Total() != 100000 {
		t.Errorf("expected total 100000, got %d", budget.Total())
	}

	if budget.Available() != 100000 {
		t.Errorf("expected available 100000, got %d", budget.Available())
	}

	if budget.Used() != 0 {
		t.Errorf("expected used 0, got %d", budget.Used())
	}

	if budget.Reserved() != 0 {
		t.Errorf("expected reserved 0, got %d", budget.Reserved())
	}
}

func TestTokenBudget_Allocate(t *testing.T) {
	budget := NewTokenBudget(100000)

	// Allocate some tokens
	allocated, ok := budget.Allocate(20000)
	if !ok {
		t.Error("expected allocation to succeed")
	}
	if allocated != 20000 {
		t.Errorf("expected allocated 20000, got %d", allocated)
	}

	// Check available decreased
	if budget.Available() != 80000 {
		t.Errorf("expected available 80000, got %d", budget.Available())
	}

	// Check reserved increased
	if budget.Reserved() != 20000 {
		t.Errorf("expected reserved 20000, got %d", budget.Reserved())
	}
}

func TestTokenBudget_AllocatePartial(t *testing.T) {
	budget := NewTokenBudget(10000)

	// Try to allocate more than available
	allocated, ok := budget.Allocate(15000)
	if !ok {
		t.Error("expected partial allocation to succeed")
	}
	if allocated != 10000 {
		t.Errorf("expected allocated 10000 (partial), got %d", allocated)
	}
}

func TestTokenBudget_AllocateNoAvailable(t *testing.T) {
	budget := NewTokenBudget(10000)

	// Allocate all
	budget.Allocate(10000)

	// Try to allocate more
	allocated, ok := budget.Allocate(5000)
	if ok {
		t.Error("expected allocation to fail when no tokens available")
	}
	if allocated != 0 {
		t.Errorf("expected allocated 0, got %d", allocated)
	}
}

func TestTokenBudget_Release(t *testing.T) {
	budget := NewTokenBudget(100000)

	// Allocate and then release
	budget.Allocate(20000)
	budget.Release(10000)

	if budget.Reserved() != 10000 {
		t.Errorf("expected reserved 10000, got %d", budget.Reserved())
	}

	if budget.Available() != 90000 {
		t.Errorf("expected available 90000, got %d", budget.Available())
	}
}

func TestTokenBudget_Consume(t *testing.T) {
	budget := NewTokenBudget(100000)

	// Allocate and consume
	budget.Allocate(20000)
	budget.Consume(15000)

	if budget.Used() != 15000 {
		t.Errorf("expected used 15000, got %d", budget.Used())
	}

	// Reserved should decrease
	if budget.Reserved() != 5000 {
		t.Errorf("expected reserved 5000, got %d", budget.Reserved())
	}

	// Available should remain same (consumed from reserved)
	if budget.Available() != 80000 {
		t.Errorf("expected available 80000, got %d", budget.Available())
	}
}

func TestTokenBudget_Stats(t *testing.T) {
	budget := NewTokenBudget(100000)

	budget.Allocate(30000)
	budget.Consume(20000)

	total, used, reserved, available := budget.Stats()

	if total != 100000 {
		t.Errorf("expected total 100000, got %d", total)
	}
	if used != 20000 {
		t.Errorf("expected used 20000, got %d", used)
	}
	if reserved != 10000 {
		t.Errorf("expected reserved 10000, got %d", reserved)
	}
	if available != 70000 {
		t.Errorf("expected available 70000, got %d", available)
	}
}

func TestTokenBudget_Reset(t *testing.T) {
	budget := NewTokenBudget(100000)

	budget.Allocate(30000)
	budget.Consume(20000)
	budget.Reset()

	if budget.Used() != 0 {
		t.Errorf("expected used 0 after reset, got %d", budget.Used())
	}
	if budget.Reserved() != 0 {
		t.Errorf("expected reserved 0 after reset, got %d", budget.Reserved())
	}
	if budget.Available() != 100000 {
		t.Errorf("expected available 100000 after reset, got %d", budget.Available())
	}
}

func TestTokenBudget_SetTotal(t *testing.T) {
	budget := NewTokenBudget(100000)

	budget.SetTotal(200000)

	if budget.Total() != 200000 {
		t.Errorf("expected total 200000, got %d", budget.Total())
	}
}

func TestTokenBudget_ConcurrentAccess(t *testing.T) {
	budget := NewTokenBudget(100000)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently allocate
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			budget.Allocate(100)
		}()
	}

	wg.Wait()

	// Should have allocated up to 100000 tokens
	reserved := budget.Reserved()
	if reserved > 100000 {
		t.Errorf("reserved should not exceed total budget, got %d", reserved)
	}
}
