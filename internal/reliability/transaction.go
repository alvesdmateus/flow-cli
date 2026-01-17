package reliability

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileOperation represents a type of file operation.
type FileOperation int

const (
	OpCreate FileOperation = iota
	OpModify
	OpDelete
	OpRename
)

// String returns the string representation of the operation.
func (op FileOperation) String() string {
	switch op {
	case OpCreate:
		return "create"
	case OpModify:
		return "modify"
	case OpDelete:
		return "delete"
	case OpRename:
		return "rename"
	default:
		return "unknown"
	}
}

// FileChange represents a single file change in a transaction.
type FileChange struct {
	Operation   FileOperation
	Path        string
	NewPath     string // For rename operations
	Content     []byte
	OldContent  []byte // For rollback
	Permissions os.FileMode
	Timestamp   time.Time
}

// Transaction provides atomic file operations.
type Transaction struct {
	id        string
	changes   []*FileChange
	committed bool
	rolledBack bool
	tempDir   string
	mu        sync.Mutex
}

// TransactionManager manages file transactions.
type TransactionManager struct {
	baseDir       string
	activeTransactions map[string]*Transaction
	mu            sync.RWMutex
}

// NewTransactionManager creates a new transaction manager.
func NewTransactionManager(baseDir string) *TransactionManager {
	return &TransactionManager{
		baseDir:            baseDir,
		activeTransactions: make(map[string]*Transaction),
	}
}

// Begin starts a new transaction.
func (tm *TransactionManager) Begin() (*Transaction, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	id := fmt.Sprintf("tx-%d", time.Now().UnixNano())
	tempDir := filepath.Join(tm.baseDir, ".vibe", "transactions", id)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create transaction directory: %w", err)
	}

	tx := &Transaction{
		id:      id,
		changes: make([]*FileChange, 0),
		tempDir: tempDir,
	}

	tm.activeTransactions[id] = tx
	return tx, nil
}

// Get retrieves an active transaction by ID.
func (tm *TransactionManager) Get(id string) (*Transaction, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tx, ok := tm.activeTransactions[id]
	return tx, ok
}

// Remove removes a transaction from the manager.
func (tm *TransactionManager) Remove(id string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.activeTransactions, id)
}

// ID returns the transaction ID.
func (tx *Transaction) ID() string {
	return tx.id
}

// AddCreate adds a file creation to the transaction.
func (tx *Transaction) AddCreate(path string, content []byte, perm os.FileMode) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finalized")
	}

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	tx.changes = append(tx.changes, &FileChange{
		Operation:   OpCreate,
		Path:        path,
		Content:     content,
		Permissions: perm,
		Timestamp:   time.Now(),
	})

	return nil
}

// AddModify adds a file modification to the transaction.
func (tx *Transaction) AddModify(path string, content []byte) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finalized")
	}

	// Read current content for rollback
	oldContent, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// Get current permissions
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	tx.changes = append(tx.changes, &FileChange{
		Operation:   OpModify,
		Path:        path,
		Content:     content,
		OldContent:  oldContent,
		Permissions: info.Mode(),
		Timestamp:   time.Now(),
	})

	return nil
}

// AddDelete adds a file deletion to the transaction.
func (tx *Transaction) AddDelete(path string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finalized")
	}

	// Read current content for rollback
	oldContent, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// Get current permissions
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	tx.changes = append(tx.changes, &FileChange{
		Operation:   OpDelete,
		Path:        path,
		OldContent:  oldContent,
		Permissions: info.Mode(),
		Timestamp:   time.Now(),
	})

	return nil
}

// AddRename adds a file rename to the transaction.
func (tx *Transaction) AddRename(oldPath, newPath string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finalized")
	}

	// Check if source exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", oldPath)
	}

	// Check if destination exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("destination file already exists: %s", newPath)
	}

	tx.changes = append(tx.changes, &FileChange{
		Operation: OpRename,
		Path:      oldPath,
		NewPath:   newPath,
		Timestamp: time.Now(),
	})

	return nil
}

// Commit applies all changes atomically.
func (tx *Transaction) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction was rolled back")
	}

	// First, write all changes to temp files
	tempFiles := make(map[int]string)
	for i, change := range tx.changes {
		if change.Operation == OpCreate || change.Operation == OpModify {
			tempPath := filepath.Join(tx.tempDir, fmt.Sprintf("change_%d", i))
			if err := os.WriteFile(tempPath, change.Content, change.Permissions); err != nil {
				// Clean up temp files
				tx.cleanupTempFiles(tempFiles)
				return fmt.Errorf("failed to write temp file: %w", err)
			}
			tempFiles[i] = tempPath
		}
	}

	// Apply all changes
	appliedChanges := 0
	for i, change := range tx.changes {
		var err error
		switch change.Operation {
		case OpCreate:
			// Ensure parent directory exists
			dir := filepath.Dir(change.Path)
			if err = os.MkdirAll(dir, 0755); err != nil {
				break
			}
			err = os.Rename(tempFiles[i], change.Path)
		case OpModify:
			err = os.Rename(tempFiles[i], change.Path)
		case OpDelete:
			err = os.Remove(change.Path)
		case OpRename:
			err = os.Rename(change.Path, change.NewPath)
		}

		if err != nil {
			// Rollback applied changes
			tx.rollbackChanges(appliedChanges)
			tx.cleanupTempFiles(tempFiles)
			return fmt.Errorf("failed to apply change %d (%s): %w", i, change.Operation, err)
		}
		appliedChanges++
	}

	tx.committed = true
	tx.cleanup()
	return nil
}

// Rollback undoes all changes.
func (tx *Transaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.rolledBack {
		return nil // Already rolled back
	}
	if tx.committed {
		return fmt.Errorf("cannot rollback committed transaction")
	}

	tx.rolledBack = true
	tx.cleanup()
	return nil
}

func (tx *Transaction) rollbackChanges(count int) {
	// Rollback in reverse order
	for i := count - 1; i >= 0; i-- {
		change := tx.changes[i]
		switch change.Operation {
		case OpCreate:
			os.Remove(change.Path)
		case OpModify:
			os.WriteFile(change.Path, change.OldContent, change.Permissions)
		case OpDelete:
			os.WriteFile(change.Path, change.OldContent, change.Permissions)
		case OpRename:
			os.Rename(change.NewPath, change.Path)
		}
	}
}

func (tx *Transaction) cleanupTempFiles(tempFiles map[int]string) {
	for _, path := range tempFiles {
		os.Remove(path)
	}
}

func (tx *Transaction) cleanup() {
	os.RemoveAll(tx.tempDir)
}

// Changes returns the list of changes in the transaction.
func (tx *Transaction) Changes() []*FileChange {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return tx.changes
}

// AtomicWriter provides atomic file writing.
type AtomicWriter struct {
	path     string
	tempPath string
	file     *os.File
	perm     os.FileMode
}

// NewAtomicWriter creates a new atomic writer.
func NewAtomicWriter(path string, perm os.FileMode) (*AtomicWriter, error) {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	return &AtomicWriter{
		path:     path,
		tempPath: tempFile.Name(),
		file:     tempFile,
		perm:     perm,
	}, nil
}

// Write writes data to the atomic writer.
func (aw *AtomicWriter) Write(p []byte) (n int, err error) {
	return aw.file.Write(p)
}

// WriteString writes a string to the atomic writer.
func (aw *AtomicWriter) WriteString(s string) (n int, err error) {
	return aw.file.WriteString(s)
}

// Close closes the writer and atomically moves the temp file to the target.
func (aw *AtomicWriter) Close() error {
	if err := aw.file.Sync(); err != nil {
		aw.Abort()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := aw.file.Close(); err != nil {
		aw.Abort()
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(aw.tempPath, aw.perm); err != nil {
		aw.Abort()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(aw.tempPath, aw.path); err != nil {
		aw.Abort()
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Abort cancels the atomic write and removes the temp file.
func (aw *AtomicWriter) Abort() {
	aw.file.Close()
	os.Remove(aw.tempPath)
}

// AtomicWriteFile writes content to a file atomically.
func AtomicWriteFile(path string, content []byte, perm os.FileMode) error {
	writer, err := NewAtomicWriter(path, perm)
	if err != nil {
		return err
	}

	if _, err := writer.Write(content); err != nil {
		writer.Abort()
		return err
	}

	return writer.Close()
}

// AtomicCopyFile copies a file atomically.
func AtomicCopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	writer, err := NewAtomicWriter(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	if _, err := io.Copy(writer, srcFile); err != nil {
		writer.Abort()
		return fmt.Errorf("failed to copy content: %w", err)
	}

	return writer.Close()
}

// SafeDelete deletes a file with backup.
func SafeDelete(path, backupDir string) error {
	// Create backup
	backupPath := filepath.Join(backupDir, filepath.Base(path)+".bak")
	if err := AtomicCopyFile(path, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Delete original
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
