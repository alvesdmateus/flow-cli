package budget

import (
	"errors"
	"sync"
)

// ErrInsufficientBudget is returned when there aren't enough tokens available
var ErrInsufficientBudget = errors.New("insufficient token budget")

// TokenBudget manages token allocation for agents and subagents
type TokenBudget struct {
	total    int // Total available tokens
	used     int // Tokens consumed
	reserved int // Tokens reserved for subagents
	mu       sync.Mutex
}

// New creates a new token budget with the specified total
func New(total int) *TokenBudget {
	return &TokenBudget{
		total: total,
	}
}

// Allocate reserves tokens for a subagent
// Returns the amount actually allocated and whether it was successful
func (b *TokenBudget) Allocate(amount int) (int, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	available := b.total - b.used - b.reserved
	if available <= 0 {
		return 0, false
	}

	// Allocate the requested amount or whatever is available
	allocated := amount
	if allocated > available {
		allocated = available
	}

	b.reserved += allocated
	return allocated, true
}

// Release returns unused tokens back to the budget
func (b *TokenBudget) Release(amount int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.reserved -= amount
	if b.reserved < 0 {
		b.reserved = 0
	}
}

// Consume marks tokens as used (from reserved pool)
func (b *TokenBudget) Consume(amount int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Move from reserved to used
	b.reserved -= amount
	if b.reserved < 0 {
		b.reserved = 0
	}
	b.used += amount
}

// Available returns the number of tokens available for allocation
func (b *TokenBudget) Available() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.total - b.used - b.reserved
}

// Total returns the total token budget
func (b *TokenBudget) Total() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.total
}

// Used returns the number of tokens consumed
func (b *TokenBudget) Used() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.used
}

// Reserved returns the number of tokens reserved for subagents
func (b *TokenBudget) Reserved() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.reserved
}

// Stats returns all budget statistics at once
func (b *TokenBudget) Stats() (total, used, reserved, available int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.total, b.used, b.reserved, b.total - b.used - b.reserved
}

// Reset resets the budget to its initial state
func (b *TokenBudget) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.used = 0
	b.reserved = 0
}

// SetTotal updates the total budget (useful for dynamic adjustment)
func (b *TokenBudget) SetTotal(total int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.total = total
}
