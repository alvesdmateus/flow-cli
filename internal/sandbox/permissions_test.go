package sandbox

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// mockApprovalFunc creates a mock approval function that returns the specified result
func mockApprovalFunc(approved bool, err error) ApprovalFunc {
	return func(op *Operation) (bool, error) {
		return approved, err
	}
}

// trackingApprovalFunc creates a mock that tracks calls
func trackingApprovalFunc(approved bool) (ApprovalFunc, *int) {
	callCount := 0
	return func(op *Operation) (bool, error) {
		callCount++
		return approved, nil
	}, &callCount
}

func TestNewManager(t *testing.T) {
	policy := DefaultPolicy()

	manager, err := NewManager(policy, mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.policy != policy {
		t.Error("manager.policy not set correctly")
	}

	if manager.validator == nil {
		t.Error("manager.validator is nil")
	}
}

func TestManager_CheckAndApprove_PermissionNone(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	approvalFn, callCount := trackingApprovalFunc(true)
	manager, err := NewManager(policy, approvalFn)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Read file in project dir should be PermissionNone (no approval needed)
	op := &Operation{
		Type:   OpReadFile,
		Target: filepath.Join(tmpDir, "file.txt"),
	}

	err = manager.CheckAndApprove(context.Background(), op)
	if err != nil {
		t.Errorf("CheckAndApprove() error = %v", err)
	}

	// Approval function should not have been called
	if *callCount != 0 {
		t.Errorf("expected 0 approval calls, got %d", *callCount)
	}
}

func TestManager_CheckAndApprove_PermissionDenied(t *testing.T) {
	tmpDir := t.TempDir()
	deniedDir := filepath.Join(tmpDir, "secret")
	policy := &Policy{
		ProjectDir:  tmpDir,
		DeniedPaths: []string{deniedDir},
	}

	manager, err := NewManager(policy, mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Read file in denied path should be rejected
	op := &Operation{
		Type:        OpReadFile,
		Target:      filepath.Join(deniedDir, "secret.txt"),
		Description: "Reading secret file",
	}

	err = manager.CheckAndApprove(context.Background(), op)
	if err == nil {
		t.Error("expected error for denied operation")
	}
}

func TestManager_CheckAndApprove_AutoApprove(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir:  tmpDir,
		AutoApprove: true,
	}

	approvalFn, callCount := trackingApprovalFunc(true)
	manager, err := NewManager(policy, approvalFn)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// With auto-approve, medium risk operations should not prompt
	op := &Operation{
		Type:        OpWriteFile,
		Target:      filepath.Join(tmpDir, "file.txt"),
		Description: "Writing file",
	}

	err = manager.CheckAndApprove(context.Background(), op)
	if err != nil {
		t.Errorf("CheckAndApprove() error = %v", err)
	}

	// Approval function should not have been called
	if *callCount != 0 {
		t.Errorf("expected 0 approval calls with auto-approve, got %d", *callCount)
	}
}

func TestManager_CheckAndApprove_HighRiskAlwaysPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir:  tmpDir,
		AutoApprove: true, // Even with auto-approve
	}

	approvalFn, callCount := trackingApprovalFunc(true)
	manager, err := NewManager(policy, approvalFn)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// High risk operations should always prompt
	op := &Operation{
		Type:        OpProcessKill,
		Target:      "1234",
		Description: "Killing process",
	}

	err = manager.CheckAndApprove(context.Background(), op)
	if err != nil {
		t.Errorf("CheckAndApprove() error = %v", err)
	}

	// Approval function should have been called
	if *callCount != 1 {
		t.Errorf("expected 1 approval call for high-risk op, got %d", *callCount)
	}
}

func TestManager_CheckAndApprove_UserRejects(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	// User rejects the operation
	manager, err := NewManager(policy, mockApprovalFunc(false, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	op := &Operation{
		Type:        OpWriteFile,
		Target:      filepath.Join(tmpDir, "file.txt"),
		Description: "Writing file",
	}

	err = manager.CheckAndApprove(context.Background(), op)
	if err == nil {
		t.Error("expected error when user rejects operation")
	}
}

func TestManager_CheckAndApprove_ApprovalError(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	// Approval function returns error
	manager, err := NewManager(policy, mockApprovalFunc(false, errors.New("approval failed")))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	op := &Operation{
		Type:        OpWriteFile,
		Target:      filepath.Join(tmpDir, "file.txt"),
		Description: "Writing file",
	}

	err = manager.CheckAndApprove(context.Background(), op)
	if err == nil {
		t.Error("expected error when approval function fails")
	}
}

func TestManager_CheckAndApprove_NoApprovalFunc(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	// Create manager without approval function
	manager, err := NewManager(policy, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	op := &Operation{
		Type:        OpWriteFile,
		Target:      filepath.Join(tmpDir, "file.txt"),
		Description: "Writing file",
	}

	err = manager.CheckAndApprove(context.Background(), op)
	if err == nil {
		t.Error("expected error when no approval function configured")
	}
}

func TestManager_ApproveForSession(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	approvalFn, callCount := trackingApprovalFunc(true)
	manager, err := NewManager(policy, approvalFn)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	target := filepath.Join(tmpDir, "file.txt")

	// Pre-approve the operation for the session
	manager.ApproveForSession(OpReadFile, target)

	// Now check the operation - should not prompt
	op := &Operation{
		Type:   OpReadFile,
		Target: target,
	}
	// Force it to be PermissionLow to test session approval
	op.Level = PermissionLow

	// The operation should be approved without prompting
	isApproved := manager.isApprovedForSession(op)
	if !isApproved {
		t.Error("isApprovedForSession() returned false for session-approved operation")
	}

	// Approval function should not be called for session-approved operations
	// (This is tested indirectly through the isApprovedForSession check)
	_ = callCount
}

func TestManager_ClearSessionApprovals(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	manager, err := NewManager(policy, mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	target := filepath.Join(tmpDir, "file.txt")

	// Approve for session
	manager.ApproveForSession(OpReadFile, target)

	op := &Operation{
		Type:   OpReadFile,
		Target: target,
	}

	// Should be approved
	if !manager.isApprovedForSession(op) {
		t.Error("operation should be approved for session")
	}

	// Clear session approvals
	manager.ClearSessionApprovals()

	// Should no longer be approved
	if manager.isApprovedForSession(op) {
		t.Error("operation should not be approved after clearing session")
	}
}

func TestManager_CanRead(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir:  tmpDir,
		DeniedPaths: []string{filepath.Join(tmpDir, "secret")},
	}

	manager, err := NewManager(policy, mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Can read file in project
	canRead, err := manager.CanRead(filepath.Join(tmpDir, "file.txt"))
	if !canRead {
		t.Error("CanRead() should return true for file in project")
	}
	if err != nil {
		t.Errorf("CanRead() error = %v", err)
	}

	// Cannot read file in denied path
	canRead, err = manager.CanRead(filepath.Join(tmpDir, "secret", "file.txt"))
	if canRead {
		t.Error("CanRead() should return false for file in denied path")
	}
	if err == nil {
		t.Error("CanRead() should return error for denied path")
	}
}

func TestManager_CanWrite(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir:  tmpDir,
		DeniedPaths: []string{filepath.Join(tmpDir, "readonly")},
	}

	manager, err := NewManager(policy, mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Can write file in project
	canWrite, err := manager.CanWrite(filepath.Join(tmpDir, "file.txt"))
	if !canWrite {
		t.Error("CanWrite() should return true for file in project")
	}
	if err != nil {
		t.Errorf("CanWrite() error = %v", err)
	}

	// Cannot write file in denied path
	canWrite, err = manager.CanWrite(filepath.Join(tmpDir, "readonly", "file.txt"))
	if canWrite {
		t.Error("CanWrite() should return false for file in denied path")
	}
	if err == nil {
		t.Error("CanWrite() should return error for denied path")
	}
}

func TestManager_CanExecute(t *testing.T) {
	policy := &Policy{
		ProjectDir: t.TempDir(),
		AllowedCommands: []string{
			"go test",
		},
		BlockedCommands: []string{
			"rm -rf /",
		},
	}

	manager, err := NewManager(policy, mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Can execute allowed command
	canExec, err := manager.CanExecute("go test ./...")
	if !canExec {
		t.Error("CanExecute() should return true for allowed command")
	}
	if err != nil {
		t.Errorf("CanExecute() error = %v", err)
	}

	// Cannot execute blocked command
	canExec, err = manager.CanExecute("rm -rf /")
	if canExec {
		t.Error("CanExecute() should return false for blocked command")
	}
	if err == nil {
		t.Error("CanExecute() should return error for blocked command")
	}
}

func TestManager_GetPolicy(t *testing.T) {
	policy := DefaultPolicy()
	manager, err := NewManager(policy, mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager.GetPolicy() != policy {
		t.Error("GetPolicy() should return the policy")
	}
}

func TestManager_GetValidator(t *testing.T) {
	manager, err := NewManager(DefaultPolicy(), mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager.GetValidator() == nil {
		t.Error("GetValidator() should not return nil")
	}
}

func TestManager_SetAutoApprove(t *testing.T) {
	manager, err := NewManager(DefaultPolicy(), mockApprovalFunc(true, nil))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Default should be false
	if manager.IsAutoApprove() {
		t.Error("IsAutoApprove() should be false by default")
	}

	// Enable auto-approve
	manager.SetAutoApprove(true)
	if !manager.IsAutoApprove() {
		t.Error("IsAutoApprove() should be true after SetAutoApprove(true)")
	}

	// Disable auto-approve
	manager.SetAutoApprove(false)
	if manager.IsAutoApprove() {
		t.Error("IsAutoApprove() should be false after SetAutoApprove(false)")
	}
}
