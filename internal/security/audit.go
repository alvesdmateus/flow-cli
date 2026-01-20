package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEventType represents the type of audit event.
type AuditEventType string

const (
	AuditFileRead    AuditEventType = "file_read"
	AuditFileWrite   AuditEventType = "file_write"
	AuditFileDelete  AuditEventType = "file_delete"
	AuditFileCreate  AuditEventType = "file_create"
	AuditDirCreate   AuditEventType = "dir_create"
	AuditDirList     AuditEventType = "dir_list"
	AuditCmdExecute  AuditEventType = "cmd_execute"
	AuditNetRequest  AuditEventType = "net_request"
	AuditLLMCall     AuditEventType = "llm_call"
	AuditSecretFound AuditEventType = "secret_found"
	AuditAccessDenied AuditEventType = "access_denied"
	AuditPermission  AuditEventType = "permission_granted"
)

// AuditSeverity represents the severity of an audit event.
type AuditSeverity string

const (
	SeverityInfo     AuditSeverity = "info"
	SeverityWarning  AuditSeverity = "warning"
	SeverityError    AuditSeverity = "error"
	SeverityCritical AuditSeverity = "critical"
)

// AuditEvent represents a single audit log entry.
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   AuditEventType         `json:"event_type"`
	Severity    AuditSeverity          `json:"severity"`
	Actor       string                 `json:"actor"`       // Who triggered the event
	Target      string                 `json:"target"`      // File path, URL, etc.
	Action      string                 `json:"action"`      // Detailed action description
	Result      string                 `json:"result"`      // success, failure, denied
	Details     map[string]interface{} `json:"details"`     // Additional context
	SessionID   string                 `json:"session_id"`  // Session identifier
	Duration    time.Duration          `json:"duration_ns"` // Operation duration
	ErrorMsg    string                 `json:"error,omitempty"`
}

// AuditLogger provides audit logging functionality.
type AuditLogger struct {
	logFile     *os.File
	logPath     string
	sessionID   string
	actor       string
	enabled     bool
	eventChan   chan *AuditEvent
	done        chan struct{}
	mu          sync.RWMutex
	nextID      int64

	// Callbacks for real-time monitoring
	onEvent     func(*AuditEvent)

	// Filtering
	minSeverity AuditSeverity
	eventTypes  map[AuditEventType]bool
}

// AuditConfig contains configuration for the audit logger.
type AuditConfig struct {
	LogPath     string
	SessionID   string
	Actor       string
	Enabled     bool
	BufferSize  int
	MinSeverity AuditSeverity
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(config AuditConfig) (*AuditLogger, error) {
	if config.BufferSize == 0 {
		config.BufferSize = 100
	}
	if config.MinSeverity == "" {
		config.MinSeverity = SeverityInfo
	}

	al := &AuditLogger{
		logPath:     config.LogPath,
		sessionID:   config.SessionID,
		actor:       config.Actor,
		enabled:     config.Enabled,
		eventChan:   make(chan *AuditEvent, config.BufferSize),
		done:        make(chan struct{}),
		minSeverity: config.MinSeverity,
		eventTypes:  make(map[AuditEventType]bool),
	}

	// Enable all event types by default
	for _, et := range []AuditEventType{
		AuditFileRead, AuditFileWrite, AuditFileDelete, AuditFileCreate,
		AuditDirCreate, AuditDirList, AuditCmdExecute, AuditNetRequest,
		AuditLLMCall, AuditSecretFound, AuditAccessDenied, AuditPermission,
	} {
		al.eventTypes[et] = true
	}

	if config.Enabled && config.LogPath != "" {
		// Ensure directory exists
		dir := filepath.Dir(config.LogPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create audit log directory: %w", err)
		}

		// Open log file
		file, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log: %w", err)
		}
		al.logFile = file

		// Start background writer
		go al.writeLoop()
	}

	return al, nil
}

// Log records an audit event.
func (al *AuditLogger) Log(event *AuditEvent) {
	if !al.enabled {
		return
	}

	// Check if event type is enabled
	al.mu.RLock()
	if !al.eventTypes[event.EventType] {
		al.mu.RUnlock()
		return
	}

	// Check severity
	if !al.severityMeetsThreshold(event.Severity) {
		al.mu.RUnlock()
		return
	}
	al.mu.RUnlock()

	// Generate ID if not set
	if event.ID == "" {
		al.mu.Lock()
		al.nextID++
		event.ID = fmt.Sprintf("%s-%d", al.sessionID, al.nextID)
		al.mu.Unlock()
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Set session ID
	if event.SessionID == "" {
		event.SessionID = al.sessionID
	}

	// Set actor
	if event.Actor == "" {
		event.Actor = al.actor
	}

	// Initialize details map if nil
	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}

	// Send to channel (non-blocking)
	select {
	case al.eventChan <- event:
	default:
		// Channel full, event dropped
	}

	// Call event callback if set
	al.mu.RLock()
	callback := al.onEvent
	al.mu.RUnlock()

	if callback != nil {
		callback(event)
	}
}

func (al *AuditLogger) severityMeetsThreshold(severity AuditSeverity) bool {
	severityOrder := map[AuditSeverity]int{
		SeverityInfo:     0,
		SeverityWarning:  1,
		SeverityError:    2,
		SeverityCritical: 3,
	}

	return severityOrder[severity] >= severityOrder[al.minSeverity]
}

func (al *AuditLogger) writeLoop() {
	for {
		select {
		case event := <-al.eventChan:
			al.writeEvent(event)
		case <-al.done:
			// Drain remaining events
			for {
				select {
				case event := <-al.eventChan:
					al.writeEvent(event)
				default:
					return
				}
			}
		}
	}
}

func (al *AuditLogger) writeEvent(event *AuditEvent) {
	if al.logFile == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	al.logFile.Write(data)
	al.logFile.Write([]byte("\n"))
}

// Close closes the audit logger.
func (al *AuditLogger) Close() error {
	al.mu.Lock()
	al.enabled = false
	al.mu.Unlock()

	close(al.done)

	if al.logFile != nil {
		return al.logFile.Close()
	}
	return nil
}

// SetOnEvent sets a callback for real-time event monitoring.
func (al *AuditLogger) SetOnEvent(callback func(*AuditEvent)) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.onEvent = callback
}

// EnableEventType enables logging for a specific event type.
func (al *AuditLogger) EnableEventType(eventType AuditEventType) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.eventTypes[eventType] = true
}

// DisableEventType disables logging for a specific event type.
func (al *AuditLogger) DisableEventType(eventType AuditEventType) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.eventTypes[eventType] = false
}

// SetMinSeverity sets the minimum severity level for logging.
func (al *AuditLogger) SetMinSeverity(severity AuditSeverity) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.minSeverity = severity
}

// Helper methods for common audit events

// LogFileRead logs a file read operation.
func (al *AuditLogger) LogFileRead(path string, success bool, err error) {
	event := &AuditEvent{
		EventType: AuditFileRead,
		Severity:  SeverityInfo,
		Target:    path,
		Action:    "Read file",
		Result:    resultString(success),
	}
	if err != nil {
		event.ErrorMsg = err.Error()
	}
	al.Log(event)
}

// LogFileWrite logs a file write operation.
func (al *AuditLogger) LogFileWrite(path string, bytes int64, success bool, err error) {
	event := &AuditEvent{
		EventType: AuditFileWrite,
		Severity:  SeverityWarning,
		Target:    path,
		Action:    "Write file",
		Result:    resultString(success),
		Details: map[string]interface{}{
			"bytes_written": bytes,
		},
	}
	if err != nil {
		event.ErrorMsg = err.Error()
	}
	al.Log(event)
}

// LogFileDelete logs a file deletion operation.
func (al *AuditLogger) LogFileDelete(path string, success bool, err error) {
	event := &AuditEvent{
		EventType: AuditFileDelete,
		Severity:  SeverityWarning,
		Target:    path,
		Action:    "Delete file",
		Result:    resultString(success),
	}
	if err != nil {
		event.ErrorMsg = err.Error()
	}
	al.Log(event)
}

// LogCommandExecute logs a command execution.
func (al *AuditLogger) LogCommandExecute(command string, exitCode int, duration time.Duration, err error) {
	severity := SeverityInfo
	if exitCode != 0 || err != nil {
		severity = SeverityWarning
	}

	event := &AuditEvent{
		EventType: AuditCmdExecute,
		Severity:  severity,
		Target:    command,
		Action:    "Execute command",
		Result:    resultString(err == nil),
		Duration:  duration,
		Details: map[string]interface{}{
			"exit_code": exitCode,
		},
	}
	if err != nil {
		event.ErrorMsg = err.Error()
	}
	al.Log(event)
}

// LogLLMCall logs an LLM API call.
func (al *AuditLogger) LogLLMCall(model string, tokens int, duration time.Duration, success bool, err error) {
	event := &AuditEvent{
		EventType: AuditLLMCall,
		Severity:  SeverityInfo,
		Target:    model,
		Action:    "LLM API call",
		Result:    resultString(success),
		Duration:  duration,
		Details: map[string]interface{}{
			"tokens": tokens,
		},
	}
	if err != nil {
		event.ErrorMsg = err.Error()
	}
	al.Log(event)
}

// LogAccessDenied logs an access denial.
func (al *AuditLogger) LogAccessDenied(target, reason string) {
	event := &AuditEvent{
		EventType: AuditAccessDenied,
		Severity:  SeverityCritical,
		Target:    target,
		Action:    "Access denied",
		Result:    "denied",
		Details: map[string]interface{}{
			"reason": reason,
		},
	}
	al.Log(event)
}

// LogSecretFound logs a detected secret.
func (al *AuditLogger) LogSecretFound(file string, line int, secretType string, confidence float64) {
	event := &AuditEvent{
		EventType: AuditSecretFound,
		Severity:  SeverityCritical,
		Target:    file,
		Action:    "Secret detected",
		Result:    "detected",
		Details: map[string]interface{}{
			"line":        line,
			"secret_type": secretType,
			"confidence":  confidence,
		},
	}
	al.Log(event)
}

// LogPermissionGranted logs a permission grant.
func (al *AuditLogger) LogPermissionGranted(operation, target string, autoApproved bool) {
	event := &AuditEvent{
		EventType: AuditPermission,
		Severity:  SeverityInfo,
		Target:    target,
		Action:    fmt.Sprintf("Permission granted for %s", operation),
		Result:    "granted",
		Details: map[string]interface{}{
			"auto_approved": autoApproved,
		},
	}
	al.Log(event)
}

func resultString(success bool) string {
	if success {
		return "success"
	}
	return "failure"
}

// QueryAuditLog reads and filters audit log entries.
func QueryAuditLog(logPath string, filter AuditFilter) ([]AuditEvent, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []AuditEvent
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var event AuditEvent
		if err := decoder.Decode(&event); err != nil {
			continue // Skip malformed entries
		}

		if filter.matches(&event) {
			events = append(events, event)
		}
	}

	return events, nil
}

// AuditFilter defines criteria for filtering audit events.
type AuditFilter struct {
	EventTypes  []AuditEventType
	Severities  []AuditSeverity
	StartTime   time.Time
	EndTime     time.Time
	SessionID   string
	TargetPath  string
	ResultType  string
}

func (f *AuditFilter) matches(event *AuditEvent) bool {
	// Check event types
	if len(f.EventTypes) > 0 {
		found := false
		for _, et := range f.EventTypes {
			if event.EventType == et {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check severities
	if len(f.Severities) > 0 {
		found := false
		for _, s := range f.Severities {
			if event.Severity == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range
	if !f.StartTime.IsZero() && event.Timestamp.Before(f.StartTime) {
		return false
	}
	if !f.EndTime.IsZero() && event.Timestamp.After(f.EndTime) {
		return false
	}

	// Check session ID
	if f.SessionID != "" && event.SessionID != f.SessionID {
		return false
	}

	// Check target path
	if f.TargetPath != "" && event.Target != f.TargetPath {
		return false
	}

	// Check result type
	if f.ResultType != "" && event.Result != f.ResultType {
		return false
	}

	return true
}
