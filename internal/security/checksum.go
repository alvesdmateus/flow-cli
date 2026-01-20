package security

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"sync"
)

// HashAlgorithm represents a supported hash algorithm.
type HashAlgorithm string

const (
	HashMD5    HashAlgorithm = "md5"
	HashSHA1   HashAlgorithm = "sha1"
	HashSHA256 HashAlgorithm = "sha256"
	HashSHA512 HashAlgorithm = "sha512"
)

// ChecksumVerifier provides checksum calculation and verification.
type ChecksumVerifier struct {
	algorithm HashAlgorithm
	cache     map[string]string // path -> checksum
	mu        sync.RWMutex
}

// NewChecksumVerifier creates a new checksum verifier.
func NewChecksumVerifier(algorithm HashAlgorithm) *ChecksumVerifier {
	if algorithm == "" {
		algorithm = HashSHA256
	}
	return &ChecksumVerifier{
		algorithm: algorithm,
		cache:     make(map[string]string),
	}
}

// getHasher returns a new hash.Hash for the configured algorithm.
func (cv *ChecksumVerifier) getHasher() (hash.Hash, error) {
	switch cv.algorithm {
	case HashMD5:
		return md5.New(), nil
	case HashSHA1:
		return sha1.New(), nil
	case HashSHA256:
		return sha256.New(), nil
	case HashSHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", cv.algorithm)
	}
}

// CalculateFile calculates the checksum of a file.
func (cv *ChecksumVerifier) CalculateFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher, err := cv.getHasher()
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Update cache
	cv.mu.Lock()
	cv.cache[path] = checksum
	cv.mu.Unlock()

	return checksum, nil
}

// CalculateContent calculates the checksum of content.
func (cv *ChecksumVerifier) CalculateContent(content []byte) (string, error) {
	hasher, err := cv.getHasher()
	if err != nil {
		return "", err
	}

	hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// CalculateString calculates the checksum of a string.
func (cv *ChecksumVerifier) CalculateString(content string) (string, error) {
	return cv.CalculateContent([]byte(content))
}

// VerifyFile verifies a file against an expected checksum.
func (cv *ChecksumVerifier) VerifyFile(path string, expected string) (bool, error) {
	actual, err := cv.CalculateFile(path)
	if err != nil {
		return false, err
	}
	return actual == expected, nil
}

// VerifyContent verifies content against an expected checksum.
func (cv *ChecksumVerifier) VerifyContent(content []byte, expected string) (bool, error) {
	actual, err := cv.CalculateContent(content)
	if err != nil {
		return false, err
	}
	return actual == expected, nil
}

// GetCached returns a cached checksum if available.
func (cv *ChecksumVerifier) GetCached(path string) (string, bool) {
	cv.mu.RLock()
	defer cv.mu.RUnlock()
	checksum, ok := cv.cache[path]
	return checksum, ok
}

// InvalidateCache removes a path from the cache.
func (cv *ChecksumVerifier) InvalidateCache(path string) {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	delete(cv.cache, path)
}

// ClearCache clears the entire cache.
func (cv *ChecksumVerifier) ClearCache() {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	cv.cache = make(map[string]string)
}

// FileIntegrityChecker monitors file integrity using checksums.
type FileIntegrityChecker struct {
	verifier    *ChecksumVerifier
	baseline    map[string]string // path -> baseline checksum
	mu          sync.RWMutex
}

// NewFileIntegrityChecker creates a new file integrity checker.
func NewFileIntegrityChecker(algorithm HashAlgorithm) *FileIntegrityChecker {
	return &FileIntegrityChecker{
		verifier: NewChecksumVerifier(algorithm),
		baseline: make(map[string]string),
	}
}

// RecordBaseline records the baseline checksum for a file.
func (fic *FileIntegrityChecker) RecordBaseline(path string) error {
	checksum, err := fic.verifier.CalculateFile(path)
	if err != nil {
		return err
	}

	fic.mu.Lock()
	fic.baseline[path] = checksum
	fic.mu.Unlock()

	return nil
}

// CheckIntegrity checks if a file has been modified from baseline.
func (fic *FileIntegrityChecker) CheckIntegrity(path string) (bool, error) {
	fic.mu.RLock()
	baselineChecksum, exists := fic.baseline[path]
	fic.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("no baseline recorded for: %s", path)
	}

	return fic.verifier.VerifyFile(path, baselineChecksum)
}

// IntegrityReport represents the result of an integrity check.
type IntegrityReport struct {
	Path     string
	Expected string
	Actual   string
	Status   IntegrityStatus
	Error    error
}

// IntegrityStatus represents the status of a file's integrity.
type IntegrityStatus string

const (
	IntegrityOK       IntegrityStatus = "ok"
	IntegrityModified IntegrityStatus = "modified"
	IntegrityMissing  IntegrityStatus = "missing"
	IntegrityError    IntegrityStatus = "error"
)

// CheckAllIntegrity checks all recorded files and returns a report.
func (fic *FileIntegrityChecker) CheckAllIntegrity() []IntegrityReport {
	fic.mu.RLock()
	paths := make([]string, 0, len(fic.baseline))
	for path := range fic.baseline {
		paths = append(paths, path)
	}
	fic.mu.RUnlock()

	var reports []IntegrityReport

	for _, path := range paths {
		report := IntegrityReport{
			Path: path,
		}

		fic.mu.RLock()
		report.Expected = fic.baseline[path]
		fic.mu.RUnlock()

		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			report.Status = IntegrityMissing
			reports = append(reports, report)
			continue
		}

		// Calculate current checksum
		actual, err := fic.verifier.CalculateFile(path)
		if err != nil {
			report.Status = IntegrityError
			report.Error = err
			reports = append(reports, report)
			continue
		}

		report.Actual = actual
		if actual == report.Expected {
			report.Status = IntegrityOK
		} else {
			report.Status = IntegrityModified
		}

		reports = append(reports, report)
	}

	return reports
}

// RemoveBaseline removes a file from the baseline.
func (fic *FileIntegrityChecker) RemoveBaseline(path string) {
	fic.mu.Lock()
	defer fic.mu.Unlock()
	delete(fic.baseline, path)
}

// ClearBaseline clears all baselines.
func (fic *FileIntegrityChecker) ClearBaseline() {
	fic.mu.Lock()
	defer fic.mu.Unlock()
	fic.baseline = make(map[string]string)
}

// SaveBaseline saves the baseline to a file.
func (fic *FileIntegrityChecker) SaveBaseline(path string) error {
	fic.mu.RLock()
	defer fic.mu.RUnlock()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for filePath, checksum := range fic.baseline {
		fmt.Fprintf(file, "%s  %s\n", checksum, filePath)
	}

	return nil
}

// LoadBaseline loads a baseline from a file.
func (fic *FileIntegrityChecker) LoadBaseline(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fic.mu.Lock()
	defer fic.mu.Unlock()

	var checksum, filePath string
	for {
		_, err := fmt.Fscanf(file, "%s  %s\n", &checksum, &filePath)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid baseline format: %w", err)
		}
		fic.baseline[filePath] = checksum
	}

	return nil
}

// WriteVerifier wraps file writes with checksum verification.
type WriteVerifier struct {
	verifier *ChecksumVerifier
}

// NewWriteVerifier creates a new write verifier.
func NewWriteVerifier() *WriteVerifier {
	return &WriteVerifier{
		verifier: NewChecksumVerifier(HashSHA256),
	}
}

// WriteAndVerify writes content to a file and verifies the write.
func (wv *WriteVerifier) WriteAndVerify(path string, content []byte) error {
	// Calculate expected checksum
	expectedChecksum, err := wv.verifier.CalculateContent(content)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Write the file
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Verify the write
	match, err := wv.verifier.VerifyFile(path, expectedChecksum)
	if err != nil {
		return fmt.Errorf("failed to verify file: %w", err)
	}

	if !match {
		return fmt.Errorf("write verification failed: checksum mismatch")
	}

	return nil
}

// AtomicWriteAndVerify performs an atomic write with verification.
func (wv *WriteVerifier) AtomicWriteAndVerify(path string, content []byte) error {
	// Calculate expected checksum
	expectedChecksum, err := wv.verifier.CalculateContent(content)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Write to temp file
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Verify temp file
	match, err := wv.verifier.VerifyFile(tempPath, expectedChecksum)
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to verify temp file: %w", err)
	}

	if !match {
		os.Remove(tempPath)
		return fmt.Errorf("write verification failed: checksum mismatch")
	}

	// Rename to final path
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
