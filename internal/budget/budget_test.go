package budget

import (
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		total int
	}{
		{"zero budget", 0},
		{"small budget", 1000},
		{"large budget", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.total)
			if b == nil {
				t.Fatal("New() returned nil")
			}
			if b.Total() != tt.total {
				t.Errorf("Total() = %d, want %d", b.Total(), tt.total)
			}
			if b.Used() != 0 {
				t.Errorf("Used() = %d, want 0", b.Used())
			}
			if b.Reserved() != 0 {
				t.Errorf("Reserved() = %d, want 0", b.Reserved())
			}
			if b.Available() != tt.total {
				t.Errorf("Available() = %d, want %d", b.Available(), tt.total)
			}
		})
	}
}

func TestAllocate(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		allocate     int
		wantAllocated int
		wantOK       bool
	}{
		{"allocate within budget", 10000, 5000, 5000, true},
		{"allocate exact budget", 10000, 10000, 10000, true},
		{"allocate more than budget", 10000, 15000, 10000, true}, // allocates available
		{"allocate from zero budget", 0, 1000, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.total)
			allocated, ok := b.Allocate(tt.allocate)
			if allocated != tt.wantAllocated {
				t.Errorf("Allocate() allocated = %d, want %d", allocated, tt.wantAllocated)
			}
			if ok != tt.wantOK {
				t.Errorf("Allocate() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestAllocate_UpdatesReserved(t *testing.T) {
	b := New(10000)

	allocated, ok := b.Allocate(3000)
	if !ok {
		t.Fatal("Allocate() should succeed")
	}
	if allocated != 3000 {
		t.Errorf("allocated = %d, want 3000", allocated)
	}
	if b.Reserved() != 3000 {
		t.Errorf("Reserved() = %d, want 3000", b.Reserved())
	}
	if b.Available() != 7000 {
		t.Errorf("Available() = %d, want 7000", b.Available())
	}

	// Second allocation
	allocated, ok = b.Allocate(5000)
	if !ok {
		t.Fatal("Allocate() should succeed")
	}
	if allocated != 5000 {
		t.Errorf("allocated = %d, want 5000", allocated)
	}
	if b.Reserved() != 8000 {
		t.Errorf("Reserved() = %d, want 8000", b.Reserved())
	}
	if b.Available() != 2000 {
		t.Errorf("Available() = %d, want 2000", b.Available())
	}
}

func TestRelease(t *testing.T) {
	b := New(10000)

	// Allocate some tokens
	b.Allocate(5000)
	if b.Reserved() != 5000 {
		t.Fatalf("Reserved() = %d, want 5000", b.Reserved())
	}

	// Release some
	b.Release(2000)
	if b.Reserved() != 3000 {
		t.Errorf("Reserved() = %d, want 3000", b.Reserved())
	}
	if b.Available() != 7000 {
		t.Errorf("Available() = %d, want 7000", b.Available())
	}

	// Release all remaining
	b.Release(3000)
	if b.Reserved() != 0 {
		t.Errorf("Reserved() = %d, want 0", b.Reserved())
	}
}

func TestRelease_CannotGoNegative(t *testing.T) {
	b := New(10000)
	b.Allocate(1000)

	// Try to release more than reserved
	b.Release(5000)

	if b.Reserved() < 0 {
		t.Error("Reserved() should not go negative")
	}
	if b.Reserved() != 0 {
		t.Errorf("Reserved() = %d, want 0", b.Reserved())
	}
}

func TestConsume(t *testing.T) {
	b := New(10000)

	// Allocate tokens
	b.Allocate(5000)

	// Consume some
	b.Consume(2000)

	if b.Used() != 2000 {
		t.Errorf("Used() = %d, want 2000", b.Used())
	}
	if b.Reserved() != 3000 {
		t.Errorf("Reserved() = %d, want 3000", b.Reserved())
	}
	if b.Available() != 5000 {
		t.Errorf("Available() = %d, want 5000", b.Available())
	}
}

func TestConsume_CannotGoNegativeReserved(t *testing.T) {
	b := New(10000)
	b.Allocate(1000)

	// Try to consume more than reserved
	b.Consume(5000)

	if b.Reserved() < 0 {
		t.Error("Reserved() should not go negative")
	}
	if b.Used() != 5000 {
		t.Errorf("Used() = %d, want 5000", b.Used())
	}
}

func TestStats(t *testing.T) {
	b := New(10000)
	b.Allocate(3000)
	b.Consume(1000)

	total, used, reserved, available := b.Stats()

	if total != 10000 {
		t.Errorf("Stats() total = %d, want 10000", total)
	}
	if used != 1000 {
		t.Errorf("Stats() used = %d, want 1000", used)
	}
	if reserved != 2000 { // 3000 allocated - 1000 consumed
		t.Errorf("Stats() reserved = %d, want 2000", reserved)
	}
	if available != 7000 { // 10000 - 1000 - 2000
		t.Errorf("Stats() available = %d, want 7000", available)
	}
}

func TestReset(t *testing.T) {
	b := New(10000)
	b.Allocate(5000)
	b.Consume(2000)

	b.Reset()

	if b.Used() != 0 {
		t.Errorf("Used() after Reset() = %d, want 0", b.Used())
	}
	if b.Reserved() != 0 {
		t.Errorf("Reserved() after Reset() = %d, want 0", b.Reserved())
	}
	if b.Available() != 10000 {
		t.Errorf("Available() after Reset() = %d, want 10000", b.Available())
	}
	if b.Total() != 10000 {
		t.Errorf("Total() after Reset() = %d, want 10000", b.Total())
	}
}

func TestSetTotal(t *testing.T) {
	b := New(10000)

	b.SetTotal(20000)

	if b.Total() != 20000 {
		t.Errorf("Total() after SetTotal() = %d, want 20000", b.Total())
	}
	if b.Available() != 20000 {
		t.Errorf("Available() after SetTotal() = %d, want 20000", b.Available())
	}
}

func TestSetTotal_WithExistingAllocations(t *testing.T) {
	b := New(10000)
	b.Allocate(5000)
	b.Consume(2000)

	b.SetTotal(8000)

	if b.Total() != 8000 {
		t.Errorf("Total() = %d, want 8000", b.Total())
	}
	// Available = total - used - reserved = 8000 - 2000 - 3000 = 3000
	if b.Available() != 3000 {
		t.Errorf("Available() = %d, want 3000", b.Available())
	}
}

func TestConcurrentAccess(t *testing.T) {
	b := New(100000)
	var wg sync.WaitGroup

	// Run multiple goroutines allocating and releasing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				allocated, _ := b.Allocate(100)
				b.Consume(allocated / 2)
				b.Release(allocated / 2)
			}
		}()
	}

	wg.Wait()

	// Check invariants
	total, used, reserved, available := b.Stats()
	if total != 100000 {
		t.Errorf("Total changed during concurrent access: %d", total)
	}
	if used < 0 || reserved < 0 || available < 0 {
		t.Error("Negative values after concurrent access")
	}
	if used+reserved+available != total {
		t.Errorf("Invariant broken: used(%d) + reserved(%d) + available(%d) != total(%d)",
			used, reserved, available, total)
	}
}

func TestErrInsufficientBudget(t *testing.T) {
	if ErrInsufficientBudget == nil {
		t.Error("ErrInsufficientBudget should not be nil")
	}
	if ErrInsufficientBudget.Error() == "" {
		t.Error("ErrInsufficientBudget should have a message")
	}
}

func TestWorkflow_TypicalSubagent(t *testing.T) {
	// Simulate a typical subagent workflow
	parentBudget := New(100000)

	// Subagent 1: Explorer
	alloc1, ok := parentBudget.Allocate(15000)
	if !ok {
		t.Fatal("Failed to allocate for explorer")
	}
	if alloc1 != 15000 {
		t.Errorf("Explorer allocation = %d, want 15000", alloc1)
	}

	// Explorer uses 12000 tokens
	parentBudget.Consume(12000)
	parentBudget.Release(3000) // Return unused

	// Subagent 2: Coder
	alloc2, ok := parentBudget.Allocate(30000)
	if !ok {
		t.Fatal("Failed to allocate for coder")
	}
	if alloc2 != 30000 {
		t.Errorf("Coder allocation = %d, want 30000", alloc2)
	}

	// Coder uses 25000 tokens
	parentBudget.Consume(25000)
	parentBudget.Release(5000) // Return unused

	// Check final state
	total, used, reserved, available := parentBudget.Stats()
	if total != 100000 {
		t.Errorf("Total = %d, want 100000", total)
	}
	if used != 37000 { // 12000 + 25000
		t.Errorf("Used = %d, want 37000", used)
	}
	if reserved != 0 {
		t.Errorf("Reserved = %d, want 0", reserved)
	}
	if available != 63000 { // 100000 - 37000
		t.Errorf("Available = %d, want 63000", available)
	}
}
