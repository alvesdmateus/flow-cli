package planner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// CorrectionType represents the type of correction made.
type CorrectionType string

const (
	CorrectionUndo       CorrectionType = "undo"        // User undid an action
	CorrectionModify     CorrectionType = "modify"      // User modified the result
	CorrectionReject     CorrectionType = "reject"      // User rejected the result
	CorrectionApprove    CorrectionType = "approve"     // User approved (positive signal)
	CorrectionRetry      CorrectionType = "retry"       // User requested retry
	CorrectionSkip       CorrectionType = "skip"        // User skipped the task
)

// Correction represents a user correction to an AI action.
type Correction struct {
	ID           string                 `json:"id"`
	SessionID    string                 `json:"session_id"`
	TaskID       string                 `json:"task_id"`
	TaskType     string                 `json:"task_type"`
	Timestamp    time.Time              `json:"timestamp"`
	Type         CorrectionType         `json:"type"`
	Original     interface{}            `json:"original"`     // What the AI did
	Corrected    interface{}            `json:"corrected"`    // What the user wanted
	Context      map[string]interface{} `json:"context"`      // Additional context
	Feedback     string                 `json:"feedback"`     // User feedback
	Tags         []string               `json:"tags"`
}

// Pattern represents a learned pattern from corrections.
type Pattern struct {
	ID          string                 `json:"id"`
	TaskType    string                 `json:"task_type"`
	Context     map[string]interface{} `json:"context"`
	Corrections int                    `json:"corrections"`   // Number of similar corrections
	LastSeen    time.Time              `json:"last_seen"`
	Confidence  float64                `json:"confidence"`    // 0.0 to 1.0
	Rule        string                 `json:"rule"`          // Learned rule description
	Examples    []PatternExample       `json:"examples"`
}

// PatternExample is an example of the pattern.
type PatternExample struct {
	Original  interface{} `json:"original"`
	Corrected interface{} `json:"corrected"`
	Context   string      `json:"context"`
}

// LearningStore stores corrections and learned patterns.
type LearningStore struct {
	storePath   string
	corrections []*Correction
	patterns    map[string]*Pattern
	mu          sync.RWMutex
}

// NewLearningStore creates a new learning store.
func NewLearningStore(storePath string) (*LearningStore, error) {
	ls := &LearningStore{
		storePath:   storePath,
		corrections: make([]*Correction, 0),
		patterns:    make(map[string]*Pattern),
	}

	// Create directory if needed
	dir := filepath.Dir(storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Load existing data
	if err := ls.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return ls, nil
}

// RecordCorrection records a user correction.
func (ls *LearningStore) RecordCorrection(correction *Correction) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if correction.ID == "" {
		correction.ID = generateID("corr")
	}
	if correction.Timestamp.IsZero() {
		correction.Timestamp = time.Now()
	}

	ls.corrections = append(ls.corrections, correction)

	// Update patterns
	ls.updatePatterns(correction)

	// Save async
	go ls.save()
}

// GetCorrections returns corrections for a session.
func (ls *LearningStore) GetCorrections(sessionID string) []*Correction {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	var result []*Correction
	for _, c := range ls.corrections {
		if c.SessionID == sessionID {
			result = append(result, c)
		}
	}
	return result
}

// GetRecentCorrections returns recent corrections.
func (ls *LearningStore) GetRecentCorrections(limit int) []*Correction {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	// Sort by timestamp descending
	sorted := make([]*Correction, len(ls.corrections))
	copy(sorted, ls.corrections)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.After(sorted[j].Timestamp)
	})

	if limit > len(sorted) {
		limit = len(sorted)
	}
	return sorted[:limit]
}

// GetPatterns returns learned patterns for a task type.
func (ls *LearningStore) GetPatterns(taskType string) []*Pattern {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	var result []*Pattern
	for _, p := range ls.patterns {
		if p.TaskType == taskType {
			result = append(result, p)
		}
	}

	// Sort by confidence
	sort.Slice(result, func(i, j int) bool {
		return result[i].Confidence > result[j].Confidence
	})

	return result
}

// GetTopPatterns returns the most confident patterns.
func (ls *LearningStore) GetTopPatterns(limit int) []*Pattern {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	patterns := make([]*Pattern, 0, len(ls.patterns))
	for _, p := range ls.patterns {
		patterns = append(patterns, p)
	}

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})

	if limit > len(patterns) {
		limit = len(patterns)
	}
	return patterns[:limit]
}

// FindMatchingPatterns finds patterns that match the current context.
func (ls *LearningStore) FindMatchingPatterns(taskType string, context map[string]interface{}) []*Pattern {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	var matches []*Pattern
	for _, p := range ls.patterns {
		if p.TaskType != taskType {
			continue
		}
		if matchContext(p.Context, context) {
			matches = append(matches, p)
		}
	}

	// Sort by confidence
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}

func matchContext(pattern, actual map[string]interface{}) bool {
	for key, pval := range pattern {
		aval, ok := actual[key]
		if !ok {
			return false
		}
		// Simple string matching
		if pstr, ok := pval.(string); ok {
			if astr, ok := aval.(string); ok {
				if !strings.Contains(astr, pstr) {
					return false
				}
			}
		}
	}
	return true
}

func (ls *LearningStore) updatePatterns(correction *Correction) {
	// Generate pattern ID based on task type and context
	patternID := generatePatternID(correction.TaskType, correction.Context)

	pattern, exists := ls.patterns[patternID]
	if !exists {
		pattern = &Pattern{
			ID:       patternID,
			TaskType: correction.TaskType,
			Context:  extractPatternContext(correction.Context),
			Examples: make([]PatternExample, 0),
		}
		ls.patterns[patternID] = pattern
	}

	pattern.Corrections++
	pattern.LastSeen = correction.Timestamp
	pattern.Confidence = calculateConfidence(pattern.Corrections, correction.Type)

	// Add example (limit to 5)
	if len(pattern.Examples) < 5 {
		pattern.Examples = append(pattern.Examples, PatternExample{
			Original:  correction.Original,
			Corrected: correction.Corrected,
			Context:   correction.Feedback,
		})
	}

	// Generate rule description
	pattern.Rule = generateRuleDescription(pattern)
}

func generatePatternID(taskType string, context map[string]interface{}) string {
	// Simple hash based on task type and key context values
	key := taskType
	if context != nil {
		if op, ok := context["operation"].(string); ok {
			key += "-" + op
		}
		if target, ok := context["target"].(string); ok {
			// Use extension or type
			ext := filepath.Ext(target)
			if ext != "" {
				key += "-" + ext
			}
		}
	}
	return key
}

func extractPatternContext(context map[string]interface{}) map[string]interface{} {
	// Extract generalizable context
	result := make(map[string]interface{})
	if context == nil {
		return result
	}

	// Extract operation type
	if op, ok := context["operation"]; ok {
		result["operation"] = op
	}

	// Extract file type (from extension)
	if target, ok := context["target"].(string); ok {
		ext := filepath.Ext(target)
		if ext != "" {
			result["file_type"] = ext
		}
	}

	// Extract language
	if lang, ok := context["language"]; ok {
		result["language"] = lang
	}

	return result
}

func calculateConfidence(corrections int, correctionType CorrectionType) float64 {
	// Base confidence from number of corrections
	base := float64(corrections) / 10.0
	if base > 0.5 {
		base = 0.5
	}

	// Adjust based on correction type
	typeWeight := 0.0
	switch correctionType {
	case CorrectionReject:
		typeWeight = 0.3 // Strong negative signal
	case CorrectionUndo:
		typeWeight = 0.2
	case CorrectionModify:
		typeWeight = 0.15
	case CorrectionRetry:
		typeWeight = 0.1
	case CorrectionApprove:
		typeWeight = -0.1 // Positive signal reduces "problem" confidence
	}

	confidence := base + typeWeight
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

func generateRuleDescription(pattern *Pattern) string {
	if len(pattern.Examples) == 0 {
		return ""
	}

	// Generate a simple rule description
	var sb strings.Builder
	sb.WriteString("When performing ")
	sb.WriteString(pattern.TaskType)

	if len(pattern.Context) > 0 {
		sb.WriteString(" on ")
		first := true
		for k, v := range pattern.Context {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v.(string))
			first = false
		}
	}

	sb.WriteString(": User corrections suggest different approach")

	return sb.String()
}

func (ls *LearningStore) load() error {
	data, err := os.ReadFile(ls.storePath)
	if err != nil {
		return err
	}

	var stored struct {
		Corrections []*Correction       `json:"corrections"`
		Patterns    map[string]*Pattern `json:"patterns"`
	}

	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	ls.corrections = stored.Corrections
	ls.patterns = stored.Patterns

	return nil
}

func (ls *LearningStore) save() error {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	stored := struct {
		Corrections []*Correction       `json:"corrections"`
		Patterns    map[string]*Pattern `json:"patterns"`
	}{
		Corrections: ls.corrections,
		Patterns:    ls.patterns,
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ls.storePath, data, 0644)
}

// Clear removes all corrections and patterns.
func (ls *LearningStore) Clear() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.corrections = make([]*Correction, 0)
	ls.patterns = make(map[string]*Pattern)

	return os.Remove(ls.storePath)
}

// SessionLearner tracks learning within a session.
type SessionLearner struct {
	sessionID   string
	store       *LearningStore
	corrections []*Correction
	preferences map[string]interface{}
	mu          sync.RWMutex
}

// NewSessionLearner creates a new session learner.
func NewSessionLearner(sessionID string, store *LearningStore) *SessionLearner {
	return &SessionLearner{
		sessionID:   sessionID,
		store:       store,
		corrections: make([]*Correction, 0),
		preferences: make(map[string]interface{}),
	}
}

// RecordCorrection records a correction in the session.
func (sl *SessionLearner) RecordCorrection(taskID, taskType string, corrType CorrectionType, original, corrected interface{}, context map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	correction := &Correction{
		SessionID: sl.sessionID,
		TaskID:    taskID,
		TaskType:  taskType,
		Timestamp: time.Now(),
		Type:      corrType,
		Original:  original,
		Corrected: corrected,
		Context:   context,
	}

	sl.corrections = append(sl.corrections, correction)

	// Update preferences
	sl.updatePreferences(correction)

	// Store for long-term learning
	if sl.store != nil {
		sl.store.RecordCorrection(correction)
	}
}

// SetFeedback sets user feedback on a correction.
func (sl *SessionLearner) SetFeedback(correctionID, feedback string) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	for _, c := range sl.corrections {
		if c.ID == correctionID {
			c.Feedback = feedback
			break
		}
	}
}

func (sl *SessionLearner) updatePreferences(correction *Correction) {
	// Track user preferences based on corrections
	key := correction.TaskType

	// Track rejection count
	if correction.Type == CorrectionReject || correction.Type == CorrectionUndo {
		countKey := key + "_rejections"
		count, _ := sl.preferences[countKey].(int)
		sl.preferences[countKey] = count + 1
	}

	// Track modification patterns
	if correction.Type == CorrectionModify {
		modsKey := key + "_modifications"
		mods, _ := sl.preferences[modsKey].([]interface{})
		sl.preferences[modsKey] = append(mods, correction.Corrected)
	}
}

// GetPreferences returns learned preferences for the session.
func (sl *SessionLearner) GetPreferences() map[string]interface{} {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range sl.preferences {
		result[k] = v
	}
	return result
}

// GetSuggestions returns suggestions based on session learning.
func (sl *SessionLearner) GetSuggestions(taskType string, context map[string]interface{}) []string {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	var suggestions []string

	// Check for high rejection rate
	rejections, _ := sl.preferences[taskType+"_rejections"].(int)
	if rejections >= 2 {
		suggestions = append(suggestions, "User has rejected similar actions; consider alternative approach")
	}

	// Check stored patterns
	if sl.store != nil {
		patterns := sl.store.FindMatchingPatterns(taskType, context)
		for _, p := range patterns {
			if p.Confidence > 0.5 {
				suggestions = append(suggestions, p.Rule)
			}
		}
	}

	return suggestions
}

// GetSessionCorrections returns all corrections in the session.
func (sl *SessionLearner) GetSessionCorrections() []*Correction {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	result := make([]*Correction, len(sl.corrections))
	copy(result, sl.corrections)
	return result
}

// Stats returns statistics about session learning.
func (sl *SessionLearner) Stats() SessionLearningStats {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	stats := SessionLearningStats{
		TotalCorrections: len(sl.corrections),
		ByType:           make(map[CorrectionType]int),
	}

	for _, c := range sl.corrections {
		stats.ByType[c.Type]++
	}

	return stats
}

// SessionLearningStats contains statistics about session learning.
type SessionLearningStats struct {
	TotalCorrections int
	ByType           map[CorrectionType]int
}

func generateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405.000")
}
