package reliability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RecoveryState represents the saved state for crash recovery.
type RecoveryState struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Operation   string                 `json:"operation"`
	Status      RecoveryStatus         `json:"status"`
	Data        map[string]interface{} `json:"data"`
	Checkpoints []*Checkpoint          `json:"checkpoints"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// RecoveryStatus represents the status of a recovery state.
type RecoveryStatus string

const (
	StatusPending    RecoveryStatus = "pending"
	StatusInProgress RecoveryStatus = "in_progress"
	StatusCompleted  RecoveryStatus = "completed"
	StatusFailed     RecoveryStatus = "failed"
	StatusRecovering RecoveryStatus = "recovering"
)

// Checkpoint represents a point in execution that can be resumed from.
type Checkpoint struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Completed bool                   `json:"completed"`
}

// RecoveryManager manages crash recovery state.
type RecoveryManager struct {
	stateDir    string
	currentState *RecoveryState
	autoSave    bool
	saveInterval time.Duration
	onRecover   func(*RecoveryState) error
	mu          sync.Mutex
	stopChan    chan struct{}
	running     bool
}

// RecoveryConfig contains configuration for recovery management.
type RecoveryConfig struct {
	StateDir     string
	AutoSave     bool
	SaveInterval time.Duration
}

// DefaultRecoveryConfig returns sensible defaults.
func DefaultRecoveryConfig(baseDir string) RecoveryConfig {
	return RecoveryConfig{
		StateDir:     filepath.Join(baseDir, ".vibe", "recovery"),
		AutoSave:     true,
		SaveInterval: 5 * time.Second,
	}
}

// NewRecoveryManager creates a new recovery manager.
func NewRecoveryManager(config RecoveryConfig) (*RecoveryManager, error) {
	if err := os.MkdirAll(config.StateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recovery directory: %w", err)
	}

	return &RecoveryManager{
		stateDir:     config.StateDir,
		autoSave:     config.AutoSave,
		saveInterval: config.SaveInterval,
		stopChan:     make(chan struct{}),
	}, nil
}

// OnRecover sets a callback for recovery operations.
func (rm *RecoveryManager) OnRecover(fn func(*RecoveryState) error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onRecover = fn
}

// Start begins automatic state saving.
func (rm *RecoveryManager) Start() {
	rm.mu.Lock()
	if rm.running || !rm.autoSave {
		rm.mu.Unlock()
		return
	}
	rm.running = true
	rm.stopChan = make(chan struct{})
	rm.mu.Unlock()

	go func() {
		ticker := time.NewTicker(rm.saveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rm.SaveState()
			case <-rm.stopChan:
				return
			}
		}
	}()
}

// Stop stops automatic state saving.
func (rm *RecoveryManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.running {
		close(rm.stopChan)
		rm.running = false
	}
}

// BeginOperation starts tracking a new operation.
func (rm *RecoveryManager) BeginOperation(operation, sessionID string) *RecoveryState {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	state := &RecoveryState{
		ID:          fmt.Sprintf("recovery-%d", time.Now().UnixNano()),
		SessionID:   sessionID,
		Timestamp:   time.Now(),
		Operation:   operation,
		Status:      StatusInProgress,
		Data:        make(map[string]interface{}),
		Checkpoints: make([]*Checkpoint, 0),
		Metadata:    make(map[string]string),
	}

	rm.currentState = state
	rm.saveStateLocked()

	return state
}

// AddCheckpoint adds a checkpoint to the current operation.
func (rm *RecoveryManager) AddCheckpoint(name string, data map[string]interface{}) *Checkpoint {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState == nil {
		return nil
	}

	checkpoint := &Checkpoint{
		ID:        fmt.Sprintf("cp-%d", len(rm.currentState.Checkpoints)),
		Name:      name,
		Timestamp: time.Now(),
		Data:      data,
		Completed: false,
	}

	rm.currentState.Checkpoints = append(rm.currentState.Checkpoints, checkpoint)
	rm.saveStateLocked()

	return checkpoint
}

// CompleteCheckpoint marks a checkpoint as completed.
func (rm *RecoveryManager) CompleteCheckpoint(checkpointID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState == nil {
		return
	}

	for _, cp := range rm.currentState.Checkpoints {
		if cp.ID == checkpointID {
			cp.Completed = true
			break
		}
	}

	rm.saveStateLocked()
}

// SetData sets data in the current operation state.
func (rm *RecoveryManager) SetData(key string, value interface{}) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState != nil {
		rm.currentState.Data[key] = value
		rm.saveStateLocked()
	}
}

// GetData retrieves data from the current operation state.
func (rm *RecoveryManager) GetData(key string) (interface{}, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState == nil {
		return nil, false
	}

	value, ok := rm.currentState.Data[key]
	return value, ok
}

// CompleteOperation marks the current operation as completed.
func (rm *RecoveryManager) CompleteOperation() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState != nil {
		rm.currentState.Status = StatusCompleted
		rm.saveStateLocked()
		rm.clearStateLocked()
	}
}

// FailOperation marks the current operation as failed.
func (rm *RecoveryManager) FailOperation(err error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState != nil {
		rm.currentState.Status = StatusFailed
		rm.currentState.Metadata["error"] = err.Error()
		rm.saveStateLocked()
	}
}

// SaveState saves the current state to disk.
func (rm *RecoveryManager) SaveState() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.saveStateLocked()
}

func (rm *RecoveryManager) saveStateLocked() error {
	if rm.currentState == nil {
		return nil
	}

	statePath := filepath.Join(rm.stateDir, rm.currentState.ID+".json")
	data, err := json.MarshalIndent(rm.currentState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return AtomicWriteFile(statePath, data, 0644)
}

func (rm *RecoveryManager) clearStateLocked() {
	if rm.currentState != nil {
		statePath := filepath.Join(rm.stateDir, rm.currentState.ID+".json")
		os.Remove(statePath)
		rm.currentState = nil
	}
}

// HasPendingRecovery checks if there's a pending recovery state.
func (rm *RecoveryManager) HasPendingRecovery() bool {
	states, _ := rm.ListPendingStates()
	return len(states) > 0
}

// ListPendingStates returns all pending recovery states.
func (rm *RecoveryManager) ListPendingStates() ([]*RecoveryState, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(rm.stateDir, "recovery-*.json"))
	if err != nil {
		return nil, err
	}

	var states []*RecoveryState
	for _, file := range files {
		state, err := rm.loadStateFile(file)
		if err != nil {
			continue
		}
		if state.Status == StatusInProgress || state.Status == StatusFailed {
			states = append(states, state)
		}
	}

	return states, nil
}

// LoadState loads a specific recovery state.
func (rm *RecoveryManager) LoadState(stateID string) (*RecoveryState, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	statePath := filepath.Join(rm.stateDir, stateID+".json")
	return rm.loadStateFile(statePath)
}

func (rm *RecoveryManager) loadStateFile(path string) (*RecoveryState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state RecoveryState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Recover attempts to recover from a pending state.
func (rm *RecoveryManager) Recover(stateID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	state, err := rm.LoadState(stateID)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	state.Status = StatusRecovering
	rm.currentState = state
	rm.saveStateLocked()

	if rm.onRecover != nil {
		if err := rm.onRecover(state); err != nil {
			state.Status = StatusFailed
			state.Metadata["recovery_error"] = err.Error()
			rm.saveStateLocked()
			return fmt.Errorf("recovery failed: %w", err)
		}
	}

	state.Status = StatusCompleted
	rm.saveStateLocked()
	rm.clearStateLocked()

	return nil
}

// RecoverLatest attempts to recover the most recent pending state.
func (rm *RecoveryManager) RecoverLatest() error {
	states, err := rm.ListPendingStates()
	if err != nil {
		return err
	}

	if len(states) == 0 {
		return fmt.Errorf("no pending states to recover")
	}

	// Find most recent
	var latest *RecoveryState
	for _, state := range states {
		if latest == nil || state.Timestamp.After(latest.Timestamp) {
			latest = state
		}
	}

	return rm.Recover(latest.ID)
}

// GetLastCheckpoint returns the last completed checkpoint.
func (rm *RecoveryManager) GetLastCheckpoint() *Checkpoint {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState == nil || len(rm.currentState.Checkpoints) == 0 {
		return nil
	}

	for i := len(rm.currentState.Checkpoints) - 1; i >= 0; i-- {
		if rm.currentState.Checkpoints[i].Completed {
			return rm.currentState.Checkpoints[i]
		}
	}

	return nil
}

// GetNextCheckpoint returns the next uncompleted checkpoint.
func (rm *RecoveryManager) GetNextCheckpoint() *Checkpoint {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState == nil {
		return nil
	}

	for _, cp := range rm.currentState.Checkpoints {
		if !cp.Completed {
			return cp
		}
	}

	return nil
}

// ClearState removes a specific state file.
func (rm *RecoveryManager) ClearState(stateID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	statePath := filepath.Join(rm.stateDir, stateID+".json")
	return os.Remove(statePath)
}

// ClearAllStates removes all recovery state files.
func (rm *RecoveryManager) ClearAllStates() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(rm.stateDir, "recovery-*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		os.Remove(file)
	}

	rm.currentState = nil
	return nil
}

// CurrentState returns the current operation state.
func (rm *RecoveryManager) CurrentState() *RecoveryState {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.currentState == nil {
		return nil
	}

	// Return a copy
	copy := *rm.currentState
	return &copy
}

// RecoveryGuard provides automatic recovery state management.
type RecoveryGuard struct {
	manager   *RecoveryManager
	operation string
	sessionID string
	started   bool
}

// NewRecoveryGuard creates a new recovery guard.
func NewRecoveryGuard(manager *RecoveryManager, operation, sessionID string) *RecoveryGuard {
	return &RecoveryGuard{
		manager:   manager,
		operation: operation,
		sessionID: sessionID,
	}
}

// Begin starts the guarded operation.
func (rg *RecoveryGuard) Begin() *RecoveryState {
	if rg.started {
		return rg.manager.CurrentState()
	}
	rg.started = true
	return rg.manager.BeginOperation(rg.operation, rg.sessionID)
}

// Checkpoint adds a checkpoint.
func (rg *RecoveryGuard) Checkpoint(name string, data map[string]interface{}) *Checkpoint {
	return rg.manager.AddCheckpoint(name, data)
}

// Complete marks the operation as completed.
func (rg *RecoveryGuard) Complete() {
	if rg.started {
		rg.manager.CompleteOperation()
		rg.started = false
	}
}

// Fail marks the operation as failed.
func (rg *RecoveryGuard) Fail(err error) {
	if rg.started {
		rg.manager.FailOperation(err)
	}
}

// Done should be called in defer to handle panics.
func (rg *RecoveryGuard) Done() {
	if r := recover(); r != nil {
		if rg.started {
			rg.manager.FailOperation(fmt.Errorf("panic: %v", r))
		}
		panic(r) // Re-panic after saving state
	}
}
