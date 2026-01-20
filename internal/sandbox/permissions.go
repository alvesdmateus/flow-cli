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
	classifier *ChangeClassifier
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

	classifier := NewChangeClassifier(policy.Classification)

	return &Manager{
		policy:          policy,
		validator:       validator,
		classifier:      classifier,
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

	// Denied operations are never allowed
	if op.Level == PermissionDenied {
		return fmt.Errorf("operation is not permitted: %s", op.Description)
	}

	// No permission needed for PermissionNone
	if op.Level == PermissionNone {
		return nil
	}

	// Check if already approved for session
	if m.isApprovedForSession(op) {
		return nil
	}

	// Determine if approval is required based on permissivity mode
	if m.shouldRequireApproval(op) {
		return m.requestApproval(op)
	}

	return nil
}

// shouldRequireApproval determines if an operation needs user approval
// based on the configured permissivity mode
func (m *Manager) shouldRequireApproval(op *Operation) bool {
	// Handle backward compatibility with AutoApprove
	if m.policy.AutoApprove && m.policy.Permissivity == "" {
		// Old behavior: auto-approve skips medium and below
		return op.Level >= PermissionHigh
	}

	// Get change size from operation details if available
	changeSize := m.getChangeSize(op)

	switch m.policy.Permissivity {
	case PermissivityAutoAccept:
		// Only require approval for explicitly denied operations
		// (which are already handled above)
		return false

	case PermissivityRevisionAlways:
		// Always require approval for any risky operation
		return op.Level >= PermissionLow

	case PermissivityRevisionBig:
		// Require approval for:
		// - Medium or higher risk operations
		// - Medium or larger changes
		return op.Level >= PermissionMedium || changeSize >= ChangeMedium

	case PermissivityRevisionArchitecture:
		// Only require approval for:
		// - High risk operations
		// - Architectural changes
		return op.Level >= PermissionHigh || changeSize >= ChangeArchitectural

	default:
		// Default to revision-big behavior
		return op.Level >= PermissionMedium || changeSize >= ChangeMedium
	}
}

// getChangeSize extracts or calculates the change size for an operation
func (m *Manager) getChangeSize(op *Operation) ChangeSize {
	// Check if change size was explicitly set in details
	if sizeStr, ok := op.Details["change_size"]; ok {
		switch sizeStr {
		case "trivial":
			return ChangeTrivial
		case "small":
			return ChangeSmall
		case "medium":
			return ChangeMedium
		case "large":
			return ChangeLarge
		case "architectural":
			return ChangeArchitectural
		}
	}

	// For file operations, use the classifier
	if op.Type == OpWriteFile || op.Type == OpDeleteFile {
		linesAdded := 0
		linesRemoved := 0
		isNewFile := false

		if la, ok := op.Details["lines_added"]; ok {
			fmt.Sscanf(la, "%d", &linesAdded)
		}
		if lr, ok := op.Details["lines_removed"]; ok {
			fmt.Sscanf(lr, "%d", &linesRemoved)
		}
		if nf, ok := op.Details["is_new_file"]; ok {
			isNewFile = nf == "true"
		}

		return m.classifier.ClassifyChange(ChangeInfo{
			Path:         op.Target,
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
			IsNewFile:    isNewFile,
		})
	}

	// Default to trivial for non-file operations
	return ChangeTrivial
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
// Deprecated: Use SetPermissivity instead
func (m *Manager) SetAutoApprove(enabled bool) {
	m.policy.AutoApprove = enabled
	if enabled {
		m.policy.Permissivity = PermissivityAutoAccept
	} else {
		// Reset to default if previously set to auto-accept
		if m.policy.Permissivity == PermissivityAutoAccept {
			m.policy.Permissivity = PermissivityRevisionBig
		}
	}
}

// IsAutoApprove returns whether auto-approve is enabled
// Deprecated: Use GetPermissivity instead
func (m *Manager) IsAutoApprove() bool {
	return m.policy.AutoApprove || m.policy.Permissivity == PermissivityAutoAccept
}

// SetPermissivity sets the permissivity mode
func (m *Manager) SetPermissivity(mode PermissivityMode) {
	m.policy.Permissivity = mode
	// Also set AutoApprove for backward compatibility
	m.policy.AutoApprove = (mode == PermissivityAutoAccept)
}

// GetPermissivity returns the current permissivity mode
func (m *Manager) GetPermissivity() PermissivityMode {
	if m.policy.Permissivity == "" {
		if m.policy.AutoApprove {
			return PermissivityAutoAccept
		}
		return PermissivityRevisionBig
	}
	return m.policy.Permissivity
}

// GetClassifier returns the change classifier
func (m *Manager) GetClassifier() *ChangeClassifier {
	return m.classifier
}

// ClearSessionApprovals clears all session-level approvals
func (m *Manager) ClearSessionApprovals() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.approvedSession = make(map[string]bool)
}
