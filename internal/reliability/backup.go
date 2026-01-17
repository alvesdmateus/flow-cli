package reliability

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// BackupManager manages file backups before destructive operations.
type BackupManager struct {
	backupDir     string
	maxBackups    int
	compressAfter time.Duration
	retention     time.Duration
	mu            sync.Mutex
}

// BackupConfig contains configuration for the backup manager.
type BackupConfig struct {
	BackupDir     string
	MaxBackups    int           // Maximum backups per file
	CompressAfter time.Duration // Compress backups older than this
	Retention     time.Duration // Delete backups older than this
}

// DefaultBackupConfig returns sensible defaults.
func DefaultBackupConfig(baseDir string) BackupConfig {
	return BackupConfig{
		BackupDir:     filepath.Join(baseDir, ".vibe", "backups"),
		MaxBackups:    10,
		CompressAfter: 24 * time.Hour,
		Retention:     7 * 24 * time.Hour, // 7 days
	}
}

// NewBackupManager creates a new backup manager.
func NewBackupManager(config BackupConfig) (*BackupManager, error) {
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &BackupManager{
		backupDir:     config.BackupDir,
		maxBackups:    config.MaxBackups,
		compressAfter: config.CompressAfter,
		retention:     config.Retention,
	}, nil
}

// BackupInfo contains metadata about a backup.
type BackupInfo struct {
	ID           string    `json:"id"`
	OriginalPath string    `json:"original_path"`
	BackupPath   string    `json:"backup_path"`
	Timestamp    time.Time `json:"timestamp"`
	Size         int64     `json:"size"`
	Compressed   bool      `json:"compressed"`
	Checksum     string    `json:"checksum,omitempty"`
}

// Backup creates a backup of a file before modification.
func (bm *BackupManager) Backup(path string) (*BackupInfo, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Read original file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Generate backup ID and path
	timestamp := time.Now()
	id := fmt.Sprintf("%d", timestamp.UnixNano())

	// Create directory structure mirroring original path
	relPath := sanitizePath(path)
	backupSubDir := filepath.Join(bm.backupDir, relPath)
	if err := os.MkdirAll(backupSubDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup subdirectory: %w", err)
	}

	backupFilename := fmt.Sprintf("%s_%s", id, filepath.Base(path))
	backupPath := filepath.Join(backupSubDir, backupFilename)

	// Write backup file
	if err := os.WriteFile(backupPath, content, info.Mode()); err != nil {
		return nil, fmt.Errorf("failed to write backup: %w", err)
	}

	// Create backup info
	backupInfo := &BackupInfo{
		ID:           id,
		OriginalPath: path,
		BackupPath:   backupPath,
		Timestamp:    timestamp,
		Size:         info.Size(),
		Compressed:   false,
	}

	// Save metadata
	if err := bm.saveMetadata(backupInfo); err != nil {
		os.Remove(backupPath)
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// Cleanup old backups
	bm.cleanupOldBackups(path)

	return backupInfo, nil
}

// Restore restores a file from backup.
func (bm *BackupManager) Restore(backupID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	info, err := bm.loadMetadata(backupID)
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	var content []byte
	if info.Compressed {
		content, err = bm.readCompressed(info.BackupPath)
	} else {
		content, err = os.ReadFile(info.BackupPath)
	}
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(info.OriginalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Restore file
	if err := AtomicWriteFile(info.OriginalPath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// List returns all backups for a file.
func (bm *BackupManager) List(path string) ([]*BackupInfo, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.listLocked(path)
}

// listLocked is the internal List that doesn't acquire a lock.
func (bm *BackupManager) listLocked(path string) ([]*BackupInfo, error) {
	relPath := sanitizePath(path)
	backupSubDir := filepath.Join(bm.backupDir, relPath)

	if _, err := os.Stat(backupSubDir); os.IsNotExist(err) {
		return nil, nil
	}

	var backups []*BackupInfo
	metadataFiles, _ := filepath.Glob(filepath.Join(backupSubDir, "*.meta.json"))

	for _, metaFile := range metadataFiles {
		info, err := bm.loadMetadataFile(metaFile)
		if err != nil {
			continue
		}
		if info.OriginalPath == path {
			backups = append(backups, info)
		}
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// GetLatest returns the most recent backup for a file.
func (bm *BackupManager) GetLatest(path string) (*BackupInfo, error) {
	backups, err := bm.List(path)
	if err != nil {
		return nil, err
	}
	if len(backups) == 0 {
		return nil, fmt.Errorf("no backups found for: %s", path)
	}
	return backups[0], nil
}

// Delete removes a backup.
func (bm *BackupManager) Delete(backupID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.deleteLocked(backupID)
}

// deleteLocked is the internal Delete that doesn't acquire a lock.
func (bm *BackupManager) deleteLocked(backupID string) error {
	info, err := bm.loadMetadata(backupID)
	if err != nil {
		return err
	}

	// Remove backup file
	os.Remove(info.BackupPath)
	if info.Compressed {
		os.Remove(info.BackupPath + ".gz")
	}

	// Remove metadata
	relPath := sanitizePath(info.OriginalPath)
	metaPath := filepath.Join(bm.backupDir, relPath, info.ID+".meta.json")
	os.Remove(metaPath)

	return nil
}

// Compress compresses old backups.
func (bm *BackupManager) Compress() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	cutoff := time.Now().Add(-bm.compressAfter)

	return filepath.Walk(bm.backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		if strings.HasSuffix(path, ".meta.json") || strings.HasSuffix(path, ".gz") {
			return nil
		}

		if info.ModTime().Before(cutoff) {
			return bm.compressFile(path)
		}

		return nil
	})
}

// Cleanup removes old backups based on retention policy.
func (bm *BackupManager) Cleanup() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	cutoff := time.Now().Add(-bm.retention)

	return filepath.Walk(bm.backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(path)
		}

		return nil
	})
}

func (bm *BackupManager) saveMetadata(info *BackupInfo) error {
	relPath := sanitizePath(info.OriginalPath)
	metaPath := filepath.Join(bm.backupDir, relPath, info.ID+".meta.json")

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0644)
}

func (bm *BackupManager) loadMetadata(backupID string) (*BackupInfo, error) {
	var info *BackupInfo

	err := filepath.Walk(bm.backupDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}

		if strings.HasSuffix(path, backupID+".meta.json") {
			info, err = bm.loadMetadataFile(path)
			if err == nil {
				return filepath.SkipAll
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("backup not found: %s", backupID)
	}
	return info, nil
}

func (bm *BackupManager) loadMetadataFile(path string) (*BackupInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var info BackupInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (bm *BackupManager) cleanupOldBackups(path string) {
	backups, err := bm.listLocked(path)
	if err != nil || len(backups) <= bm.maxBackups {
		return
	}

	// Remove oldest backups
	for i := bm.maxBackups; i < len(backups); i++ {
		bm.deleteLocked(backups[i].ID)
	}
}

func (bm *BackupManager) compressFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	gzPath := path + ".gz"
	gzFile, err := os.Create(gzPath)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	gzWriter := gzip.NewWriter(gzFile)
	if _, err := gzWriter.Write(content); err != nil {
		os.Remove(gzPath)
		return err
	}
	gzWriter.Close()

	// Update metadata
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".meta.json"
	// Find the correct metadata file
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	id := strings.Split(base, "_")[0]
	metaPath = filepath.Join(dir, id+".meta.json")

	if info, err := bm.loadMetadataFile(metaPath); err == nil {
		info.Compressed = true
		info.BackupPath = gzPath
		data, _ := json.MarshalIndent(info, "", "  ")
		os.WriteFile(metaPath, data, 0644)
	}

	// Remove original
	os.Remove(path)

	return nil
}

func (bm *BackupManager) readCompressed(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	return io.ReadAll(gzReader)
}

func sanitizePath(path string) string {
	// Convert to relative path and replace separators
	abs, _ := filepath.Abs(path)
	// Replace colons (Windows drive letters) and path separators
	sanitized := strings.ReplaceAll(abs, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, string(filepath.Separator), "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	return sanitized
}

// BackupGuard automatically creates backups before operations.
type BackupGuard struct {
	manager *BackupManager
	backups map[string]*BackupInfo
	mu      sync.Mutex
}

// NewBackupGuard creates a new backup guard.
func NewBackupGuard(manager *BackupManager) *BackupGuard {
	return &BackupGuard{
		manager: manager,
		backups: make(map[string]*BackupInfo),
	}
}

// Guard creates a backup before an operation.
func (bg *BackupGuard) Guard(path string) error {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // No backup needed for new files
	}

	info, err := bg.manager.Backup(path)
	if err != nil {
		return err
	}

	bg.backups[path] = info
	return nil
}

// Rollback restores all guarded files.
func (bg *BackupGuard) Rollback() error {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	var lastErr error
	for _, info := range bg.backups {
		if err := bg.manager.Restore(info.ID); err != nil {
			lastErr = err
		}
	}

	bg.backups = make(map[string]*BackupInfo)
	return lastErr
}

// Commit clears the guard without restoring.
func (bg *BackupGuard) Commit() {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.backups = make(map[string]*BackupInfo)
}

// GetBackups returns all backups created by this guard.
func (bg *BackupGuard) GetBackups() []*BackupInfo {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	backups := make([]*BackupInfo, 0, len(bg.backups))
	for _, info := range bg.backups {
		backups = append(backups, info)
	}
	return backups
}
