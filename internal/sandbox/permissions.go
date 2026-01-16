package sandbox

import (
	"context"
	"fmt"
	"sync"
)

// ApprovalFunc is a function that prompts the user for approval
type ApprovalFunc func(op *Operation) (bool, error)

// Manager handles permission checks and approval workflows
type Manager struct {
	policy     *Policy
	validator  *Validator
	approvalFn ApprovalFunc

	// Track approved operations to avoid re-prompting
	mu              sync.RWMutex
	approvedOnce    map[string]bool // Operations approved for single use
	approvedSession map[string]bool // Operations approved for the session
}

// NewManager creates a new permission manager
func NewManager(policy *Policy, approvalFn ApprovalFunc) (*Manager, error) {
	validator, err := NewValidator(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	return &Manager{
		policy:          policy,
		validator:       validator,
		approvalFn:      approvalFn,
		approvedOnce:    make(map[string]bool),
		approvedSession: make(map[string]bool),
	}, nil
}

// CheckAndApprove validates an operation and requests approval if needed
func (m *Manager) CheckAndApprove(ctx context.Context, op *Operation) error {
	// Validate the operation (this sets the permission level)
	if err := m.validator.ValidateOperation(op); err != nil {
		if op.Level == PermissionDenied {
			return fmt.Errorf("operation denied: %w", err)
		}
		// Non-fatal validation issues
	}

	// Check permission level
	switch op.Level {
	case PermissionDenied:
		return fmt.Errorf("operation is not permitted: %s", op.Description)

	case PermissionNone:
		// No approval needed
		return nil

	case PermissionLow:
		// Low risk - auto-approve if auto-approve is enabled
		if m.policy.AutoApprove {
			return nil
		}
		// Otherwise, check if already approved for session
		if m.isApprovedForSession(op) {
			return nil
		}
		// Request approval
		return m.requestApproval(op)

	case PermissionMedium:
		// Medium risk - needs approval unless auto-approve
		if m.policy.AutoApprove {
			return nil
		}
		return m.requestApproval(op)

	case PermissionHigh:
		// High risk - always needs explicit approval
		return m.requestApproval(op)
	}

	return nil
}

// requestApproval prompts the user for approval
func (m *Manager) requestApproval(op *Operation) error {
	if m.approvalFn == nil {
		return fmt.Errorf("no approval function configured")
	}

	approved, err := m.approvalFn(op)
	if err != nil {
		return fmt.Errorf("approval failed: %w", err)
	}

	if !approved {
		return fmt.Errorf("operation rejected by user: %s", op.Description)
	}

	return nil
}

// ApproveForSession marks an operation type as approved for the session
func (m *Manager) ApproveForSession(opType OperationType, target string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", opType, target)
	m.approvedSession[key] = true
}

// isApprovedForSession checks if an operation was approved for the session
func (m *Manager) isApprovedForSession(op *Operation) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", op.Type, op.Target)
	return m.approvedSession[key]
}

// CanRead checks if a path can be read (convenience method)
func (m *Manager) CanRead(path string) (bool, error) {
	level, err := m.validator.ValidateReadPath(path)
	if err != nil && level == PermissionDenied {
		return false, err
	}
	return level != PermissionDenied, nil
}

// CanWrite checks if a path can be written (convenience method)
func (m *Manager) CanWrite(path string) (bool, error) {
	level, err := m.validator.ValidateWritePath(path)
	if err != nil && level == PermissionDenied {
		return false, err
	}
	return level != PermissionDenied, nil
}

// CanExecute checks if a command can be executed (convenience method)
func (m *Manager) CanExecute(command string) (bool, error) {
	level, err := m.validator.ValidateCommand(command)
	if err != nil && level == PermissionDenied {
		return false, err
	}
	return level != PermissionDenied, nil
}

// GetPolicy returns the current policy
func (m *Manager) GetPolicy() *Policy {
	return m.policy
}

// GetValidator returns the validator
func (m *Manager) GetValidator() *Validator {
	return m.validator
}

// SetAutoApprove enables or disables auto-approve mode
func (m *Manager) SetAutoApprove(enabled bool) {
	m.policy.AutoApprove = enabled
}

// IsAutoApprove returns whether auto-approve is enabled
func (m *Manager) IsAutoApprove() bool {
	return m.policy.AutoApprove
}

// ClearSessionApprovals clears all session-level approvals
func (m *Manager) ClearSessionApprovals() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.approvedSession = make(map[string]bool)
}
