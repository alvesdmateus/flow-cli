// Package testing provides test generation and execution functionality.
package testing

import "time"

// ProjectType represents the type of project being tested.
type ProjectType string

const (
	ProjectUnknown ProjectType = "unknown"
	ProjectGo      ProjectType = "go"
	ProjectPython  ProjectType = "python"
	ProjectNode    ProjectType = "node"
	ProjectRust    ProjectType = "rust"
)

// String returns the string representation of the project type.
func (p ProjectType) String() string {
	return string(p)
}

// RunResult contains the results of a test run.
type RunResult struct {
	Passed      bool
	TestsPassed int
	TestsFailed int
	TestsSkipped int
	Duration    time.Duration
	Coverage    float64
	Failures    []TestFailure
	Output      string
}

// TestFailure represents a single test failure.
type TestFailure struct {
	Name    string
	File    string
	Line    int
	Message string
	Output  string
}

// GenerateResult contains the results of test generation.
type GenerateResult struct {
	TestCode    string
	TestFile    string
	TestCount   int
	Description string
}

// AnalysisResult contains the results of failure analysis.
type AnalysisResult struct {
	Suggestions []FixSuggestion
}

// FixSuggestion represents a suggested fix for a failing test.
type FixSuggestion struct {
	TestName string
	Issue    string
	Fix      string
	Code     string
}
