// Package review provides AI-powered code review functionality.
package review

// Severity represents the severity level of an issue.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Verdict represents the overall review verdict.
type Verdict string

const (
	VerdictApprove        Verdict = "approve"
	VerdictRequestChanges Verdict = "request_changes"
	VerdictComment        Verdict = "comment"
)

// ReviewResult contains the complete review output.
type ReviewResult struct {
	Summary     string       `json:"summary"`
	Verdict     Verdict      `json:"verdict"`
	Issues      []Issue      `json:"issues"`
	Suggestions []string     `json:"suggestions"`
	FileReviews []FileReview `json:"file_reviews"`
}

// Issue represents a single issue found during review.
type Issue struct {
	Severity   Severity `json:"severity"`
	Category   string   `json:"category"` // security, performance, bug, style, etc.
	Message    string   `json:"message"`
	File       string   `json:"file,omitempty"`
	Line       int      `json:"line,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// FileReview contains review feedback for a specific file.
type FileReview struct {
	File    string `json:"file"`
	Summary string `json:"summary"`
	Notes   []Note `json:"notes"`
}

// Note is a single comment/note about code.
type Note struct {
	Line    int    `json:"line,omitempty"`
	Comment string `json:"comment"`
}

// DiffInfo contains parsed information about a diff.
type DiffInfo struct {
	FilesChanged int
	Additions    int
	Deletions    int
	Files        []FileDiff
}

// FileDiff represents changes to a single file.
type FileDiff struct {
	Path       string
	OldPath    string // For renames
	Status     string // added, modified, deleted, renamed
	Additions  int
	Deletions  int
	Hunks      []Hunk
	IsBinary   bool
	Language   string // Detected language
}

// Hunk represents a single diff hunk.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Header   string
	Lines    []DiffLine
}

// DiffLine represents a single line in a diff.
type DiffLine struct {
	Type    LineType
	Content string
	OldLine int
	NewLine int
}

// LineType represents the type of a diff line.
type LineType string

const (
	LineContext  LineType = "context"
	LineAddition LineType = "addition"
	LineDeletion LineType = "deletion"
)
