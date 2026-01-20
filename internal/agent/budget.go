package agent

import (
	"github.com/mateus/flow-cli/internal/budget"
)

// TokenBudget is an alias for budget.TokenBudget
type TokenBudget = budget.TokenBudget

// ErrInsufficientBudget is an alias for budget.ErrInsufficientBudget
var ErrInsufficientBudget = budget.ErrInsufficientBudget

// NewTokenBudget creates a new token budget with the specified total
func NewTokenBudget(total int) *TokenBudget {
	return budget.New(total)
}
