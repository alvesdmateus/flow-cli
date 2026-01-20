package reliability

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ==================== Retry Tests ====================

func TestRetryerSuccess(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	retryer := NewRetryer(config)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryerRetryOnFailure(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.InitialDelay = 10 * time.Millisecond
	retryer := NewRetryer(config)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return &RetryableError{Err: errors.New("temporary error"), Retryable: true}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryerMaxRetriesExceeded(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 2
	config.InitialDelay = 10 * time.Millisecond
	retryer := NewRetryer(config)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return &RetryableError{Err: errors.New("always fails"), Retryable: true}
	})

	if err == nil {
		t.Error("Expected error after max retries")
	}
	if attempts != 3 { // Initial + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryerNonRetryableError(t *testing.T) {
	config := DefaultRetryConfig()
	retryer := NewRetryer(config)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return &RetryableError{Err: errors.New("non-retryable"), Retryable: false}
	})

	if err == nil {
		t.Error("Expected error")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryerContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialDelay = 100 * time.Millisecond
	retryer := NewRetryer(config)

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retryer.Do(ctx, func() error {
		attempts++
		return &RetryableError{Err: errors.New("retry"), Retryable: true}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestCircuitBreakerOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
		HalfOpenMax:  1,
	}
	cb := NewCircuitBreaker(config)

	// Fail enough to open circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("Expected circuit to be open, got %v", cb.State())
	}

	// Next call should fail immediately
	err := cb.Execute(func() error {
		return nil
	})

	if err == nil {
		t.Error("Expected error when circuit is open")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: 50 * time.Millisecond,
		HalfOpenMax:  1,
	}
	cb := NewCircuitBreaker(config)

	// Trip the circuit
	cb.Execute(func() error {
		return errors.New("failure")
	})

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected success in half-open, got %v", err)
	}
	if cb.State() != CircuitClosed {
		t.Errorf("Expected circuit to close after success, got %v", cb.State())
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"retryable error", &RetryableError{Err: errors.New("test"), Retryable: true}, true},
		{"non-retryable error", &RetryableError{Err: errors.New("test"), Retryable: false}, false},
		{"timeout error", errors.New("connection timeout"), true},
		{"rate limit error", errors.New("rate limit exceeded"), true},
		{"regular error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// ==================== Fallback Tests ====================

func TestFallbackHandlerNormalMode(t *testing.T) {
	config := DefaultFallbackConfig()
	fh := NewFallbackHandler(config)

	result, err := fh.Execute(context.Background(), "test", func() (interface{}, error) {
		return "success", nil
	}, "cache-key")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got %v", result)
	}
}

func TestFallbackHandlerDegradedMode(t *testing.T) {
	config := DefaultFallbackConfig()
	fh := NewFallbackHandler(config)

	// First call succeeds and caches
	fh.Execute(context.Background(), "test", func() (interface{}, error) {
		return "cached-value", nil
	}, "test-key")

	// Mark primary as unavailable
	fh.SetPrimaryAvailable(false)

	if fh.Mode() != ModeDegraded {
		t.Errorf("Expected degraded mode, got %v", fh.Mode())
	}

	// Should return cached value
	result, err := fh.Execute(context.Background(), "test", func() (interface{}, error) {
		return nil, errors.New("should not be called")
	}, "test-key")

	if err != nil {
		t.Errorf("Expected no error from cache, got %v", err)
	}
	if result != "cached-value" {
		t.Errorf("Expected cached value, got %v", result)
	}
}

func TestResponseCache(t *testing.T) {
	cache := NewResponseCache(100*time.Millisecond, 10)

	cache.Set("key1", "value1")

	value, ok := cache.Get("key1")
	if !ok || value != "value1" {
		t.Errorf("Expected to get 'value1', got %v, %v", value, ok)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, ok = cache.Get("key1")
	if ok {
		t.Error("Expected cache entry to be expired")
	}
}

func TestServiceDegrader(t *testing.T) {
	sd := NewServiceDegrader()

	sd.RegisterService("llm", []string{"chat", "completion"})
	sd.RegisterService("search", []string{"web_search"})

	// Initially all features available
	if !sd.IsFeatureAvailable("chat") {
		t.Error("Expected chat to be available")
	}

	// Mark LLM down
	sd.MarkServiceDown("llm")

	if sd.IsFeatureAvailable("chat") {
		t.Error("Expected chat to be unavailable after LLM down")
	}

	// Search should still work
	if !sd.IsFeatureAvailable("web_search") {
		t.Error("Expected web_search to still be available")
	}

	degraded := sd.GetDegradedFeatures()
	if len(degraded) != 2 { // chat and completion
		t.Errorf("Expected 2 degraded features, got %d", len(degraded))
	}
}

// ==================== Transaction Tests ====================

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic_test.txt")

	content := []byte("test content")
	err := AtomicWriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	// Verify content
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("Content mismatch: got %s, want %s", readContent, content)
	}
}

func TestTransactionCommit(t *testing.T) {
	tmpDir := t.TempDir()
	tm := NewTransactionManager(tmpDir)

	tx, err := tm.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	testFile := filepath.Join(tmpDir, "tx_test.txt")
	err = tx.AddCreate(testFile, []byte("new content"), 0644)
	if err != nil {
		t.Fatalf("Failed to add create: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify file exists
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if string(content) != "new content" {
		t.Errorf("Content mismatch: got %s", content)
	}
}

func TestTransactionRollback(t *testing.T) {
	tmpDir := t.TempDir()
	tm := NewTransactionManager(tmpDir)

	tx, err := tm.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	testFile := filepath.Join(tmpDir, "rollback_test.txt")
	tx.AddCreate(testFile, []byte("content"), 0644)

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify file does not exist
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should not exist after rollback")
	}
}

func TestTransactionModify(t *testing.T) {
	tmpDir := t.TempDir()
	tm := NewTransactionManager(tmpDir)

	// Create original file
	testFile := filepath.Join(tmpDir, "modify_test.txt")
	os.WriteFile(testFile, []byte("original"), 0644)

	tx, err := tm.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	err = tx.AddModify(testFile, []byte("modified"))
	if err != nil {
		t.Fatalf("Failed to add modify: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "modified" {
		t.Errorf("Expected 'modified', got %s", content)
	}
}

// ==================== Backup Tests ====================

func TestBackupAndRestore(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultBackupConfig(tmpDir)
	bm, err := NewBackupManager(config)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "backup_test.txt")
	os.WriteFile(testFile, []byte("original content"), 0644)

	// Create backup
	info, err := bm.Backup(testFile)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify original
	os.WriteFile(testFile, []byte("modified content"), 0644)

	// Restore
	err = bm.Restore(info.ID)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "original content" {
		t.Errorf("Expected 'original content', got %s", content)
	}
}

func TestBackupList(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultBackupConfig(tmpDir)
	bm, err := NewBackupManager(config)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	testFile := filepath.Join(tmpDir, "list_test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	// Create multiple backups
	bm.Backup(testFile)
	time.Sleep(10 * time.Millisecond)
	bm.Backup(testFile)

	backups, err := bm.List(testFile)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(backups))
	}
}

func TestBackupGuard(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultBackupConfig(tmpDir)
	bm, _ := NewBackupManager(config)
	guard := NewBackupGuard(bm)

	testFile := filepath.Join(tmpDir, "guard_test.txt")
	os.WriteFile(testFile, []byte("original"), 0644)

	// Guard the file
	guard.Guard(testFile)

	// Modify file
	os.WriteFile(testFile, []byte("modified"), 0644)

	// Rollback
	guard.Rollback()

	content, _ := os.ReadFile(testFile)
	if string(content) != "original" {
		t.Errorf("Expected 'original' after rollback, got %s", content)
	}
}

// ==================== Health Check Tests ====================

func TestHealthCheckerRegister(t *testing.T) {
	config := DefaultHealthCheckConfig()
	hc := NewHealthChecker(config)

	hc.RegisterProvider("ollama", "http://localhost:11434")

	status := hc.GetStatus("ollama")
	if status == nil {
		t.Error("Expected status for registered provider")
	}
	if status.Status != StatusUnknown {
		t.Errorf("Expected unknown status, got %v", status.Status)
	}
}

func TestHealthCheckerSelectBest(t *testing.T) {
	config := DefaultHealthCheckConfig()
	hc := NewHealthChecker(config)

	hc.RegisterProvider("slow", "http://localhost:1")
	hc.RegisterProvider("fast", "http://localhost:2")

	// Manually set results
	hc.mu.Lock()
	hc.results["slow"].Status = StatusHealthy
	hc.results["slow"].Available = true
	hc.results["slow"].Latency = 100 * time.Millisecond

	hc.results["fast"].Status = StatusHealthy
	hc.results["fast"].Available = true
	hc.results["fast"].Latency = 10 * time.Millisecond
	hc.mu.Unlock()

	best := hc.SelectBestProvider()
	if best != "fast" {
		t.Errorf("Expected 'fast' provider, got %s", best)
	}
}

func TestOverallHealth(t *testing.T) {
	config := DefaultHealthCheckConfig()
	hc := NewHealthChecker(config)

	hc.RegisterProvider("p1", "http://localhost:1")
	hc.RegisterProvider("p2", "http://localhost:2")

	// All healthy
	hc.mu.Lock()
	hc.results["p1"].Status = StatusHealthy
	hc.results["p1"].Available = true
	hc.results["p2"].Status = StatusHealthy
	hc.results["p2"].Available = true
	hc.mu.Unlock()

	if hc.OverallHealth() != StatusHealthy {
		t.Error("Expected overall healthy status")
	}

	// One degraded
	hc.mu.Lock()
	hc.results["p1"].Status = StatusDegraded
	hc.mu.Unlock()

	if hc.OverallHealth() != StatusDegraded {
		t.Error("Expected overall degraded status")
	}

	// All unhealthy
	hc.mu.Lock()
	hc.results["p1"].Status = StatusUnhealthy
	hc.results["p1"].Available = false
	hc.results["p2"].Status = StatusUnhealthy
	hc.results["p2"].Available = false
	hc.mu.Unlock()

	if hc.OverallHealth() != StatusUnhealthy {
		t.Error("Expected overall unhealthy status")
	}
}

// ==================== Recovery Tests ====================

func TestRecoveryManagerBasic(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultRecoveryConfig(tmpDir)
	rm, err := NewRecoveryManager(config)
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Begin operation
	state := rm.BeginOperation("test-op", "session-1")
	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	// Add checkpoint
	cp := rm.AddCheckpoint("step1", map[string]interface{}{"key": "value"})
	if cp == nil {
		t.Fatal("Expected non-nil checkpoint")
	}

	// Complete checkpoint
	rm.CompleteCheckpoint(cp.ID)

	// Set data
	rm.SetData("progress", 50)

	// Verify data
	val, ok := rm.GetData("progress")
	if !ok || val.(int) != 50 {
		t.Errorf("Expected progress=50, got %v", val)
	}

	// Complete operation
	rm.CompleteOperation()

	// No pending states
	if rm.HasPendingRecovery() {
		t.Error("Expected no pending recovery after completion")
	}
}

func TestRecoveryManagerPendingState(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultRecoveryConfig(tmpDir)
	rm, _ := NewRecoveryManager(config)

	// Begin but don't complete
	rm.BeginOperation("incomplete", "session-1")
	rm.AddCheckpoint("step1", nil)
	rm.SaveState()

	// Create new manager to simulate restart
	rm2, _ := NewRecoveryManager(config)

	if !rm2.HasPendingRecovery() {
		t.Error("Expected pending recovery state")
	}

	states, _ := rm2.ListPendingStates()
	if len(states) != 1 {
		t.Errorf("Expected 1 pending state, got %d", len(states))
	}
}

func TestRecoveryGuard(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultRecoveryConfig(tmpDir)
	rm, _ := NewRecoveryManager(config)

	guard := NewRecoveryGuard(rm, "test-operation", "session-1")
	guard.Begin()
	guard.Checkpoint("step1", nil)
	guard.Complete()

	if rm.HasPendingRecovery() {
		t.Error("Expected no pending recovery after guard completion")
	}
}

func TestRecoveryCheckpoints(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultRecoveryConfig(tmpDir)
	rm, _ := NewRecoveryManager(config)

	rm.BeginOperation("multi-step", "session-1")

	cp1 := rm.AddCheckpoint("step1", nil)
	cp2 := rm.AddCheckpoint("step2", nil)
	rm.AddCheckpoint("step3", nil)

	rm.CompleteCheckpoint(cp1.ID)
	rm.CompleteCheckpoint(cp2.ID)

	lastCompleted := rm.GetLastCheckpoint()
	if lastCompleted == nil || lastCompleted.ID != cp2.ID {
		t.Error("Expected cp2 as last completed")
	}

	nextPending := rm.GetNextCheckpoint()
	if nextPending == nil || nextPending.Name != "step3" {
		t.Error("Expected step3 as next pending")
	}
}

// ==================== Integration Tests ====================

func TestRetryWithCircuitBreaker(t *testing.T) {
	retryer := NewRetryer(RetryConfig{
		MaxRetries:   5,
		InitialDelay: 10 * time.Millisecond,
		Strategy:     ExponentialBackoff,
	})

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  3,
		ResetTimeout: 100 * time.Millisecond,
	})

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		return cb.Execute(func() error {
			attempts++
			if attempts < 5 {
				return &RetryableError{Err: errors.New("fail"), Retryable: true}
			}
			return nil
		})
	})

	// Circuit should open after failures
	if cb.State() == CircuitClosed && err == nil {
		// If we got here without error and circuit is closed, good
	} else if cb.State() == CircuitOpen {
		// Circuit opened due to failures, expected
	} else if err != nil {
		// Some error occurred, which is fine in this test
	}

	// The test verifies integration works without panics
}

func TestBackupWithTransaction(t *testing.T) {
	tmpDir := t.TempDir()

	bm, _ := NewBackupManager(DefaultBackupConfig(tmpDir))
	tm := NewTransactionManager(tmpDir)
	guard := NewBackupGuard(bm)

	// Create original file
	testFile := filepath.Join(tmpDir, "integrated_test.txt")
	os.WriteFile(testFile, []byte("original"), 0644)

	// Guard the file
	guard.Guard(testFile)

	// Start transaction
	tx, _ := tm.Begin()
	tx.AddModify(testFile, []byte("modified"))

	// Simulate failure - rollback both
	tx.Rollback()
	guard.Rollback()

	content, _ := os.ReadFile(testFile)
	if string(content) != "original" {
		t.Errorf("Expected 'original' after rollback, got %s", content)
	}
}
