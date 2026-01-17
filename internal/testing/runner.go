package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RunnerConfig contains configuration for the test runner.
type RunnerConfig struct {
	WorkDir     string
	ProjectType ProjectType
	Coverage    bool
	Timeout     string
}

// Runner executes tests for a project.
type Runner struct {
	workDir     string
	projectType ProjectType
	coverage    bool
	timeout     time.Duration
}

// NewRunner creates a new test runner.
func NewRunner(cfg RunnerConfig) *Runner {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = 5 * time.Minute
	}

	return &Runner{
		workDir:     cfg.WorkDir,
		projectType: cfg.ProjectType,
		coverage:    cfg.Coverage,
		timeout:     timeout,
	}
}

// Run executes tests and returns the results.
func (r *Runner) Run(ctx context.Context, args []string) (*RunResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Get the test command
	cmdName, cmdArgs := GetTestCommand(r.projectType)
	if cmdName == "" {
		return nil, fmt.Errorf("no test command for project type: %s", r.projectType)
	}

	// Add coverage args if requested
	if r.coverage {
		cmdArgs = append(cmdArgs, GetCoverageArgs(r.projectType)...)
	}

	// Add user-specified args
	cmdArgs = append(cmdArgs, args...)

	// Add verbose/json output for better parsing
	switch r.projectType {
	case ProjectGo:
		cmdArgs = append(cmdArgs, "-v", "-json")
	case ProjectPython:
		cmdArgs = append(cmdArgs, "-v")
	}

	// Execute the command
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = r.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	// Parse the results based on project type
	result := r.parseOutput(output, r.projectType)
	result.Duration = duration
	result.Output = output

	// Determine pass/fail
	if err != nil {
		result.Passed = false
	} else {
		result.Passed = result.TestsFailed == 0
	}

	return result, nil
}

func (r *Runner) parseOutput(output string, projectType ProjectType) *RunResult {
	switch projectType {
	case ProjectGo:
		return r.parseGoOutput(output)
	case ProjectPython:
		return r.parsePytestOutput(output)
	case ProjectNode:
		return r.parseNodeOutput(output)
	case ProjectRust:
		return r.parseCargoOutput(output)
	default:
		return &RunResult{Output: output}
	}
}

// Go test JSON output parsing
type goTestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

func (r *Runner) parseGoOutput(output string) *RunResult {
	result := &RunResult{
		Failures: []TestFailure{},
	}

	lines := strings.Split(output, "\n")
	testOutputs := make(map[string]string)

	for _, line := range lines {
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		testKey := event.Package + "/" + event.Test

		switch event.Action {
		case "pass":
			if event.Test != "" {
				result.TestsPassed++
			}
		case "fail":
			if event.Test != "" {
				result.TestsFailed++
				result.Failures = append(result.Failures, TestFailure{
					Name:   event.Test,
					Output: testOutputs[testKey],
				})
			}
		case "skip":
			if event.Test != "" {
				result.TestsSkipped++
			}
		case "output":
			if event.Test != "" {
				testOutputs[testKey] += event.Output
			}
		}
	}

	// Parse coverage from output
	coverageRegex := regexp.MustCompile(`coverage: (\d+\.?\d*)%`)
	if matches := coverageRegex.FindStringSubmatch(output); matches != nil {
		if cov, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Coverage = cov
		}
	}

	return result
}

func (r *Runner) parsePytestOutput(output string) *RunResult {
	result := &RunResult{
		Failures: []TestFailure{},
	}

	// Parse pytest summary line: "X passed, Y failed, Z skipped"
	summaryRegex := regexp.MustCompile(`(\d+) passed`)
	if matches := summaryRegex.FindStringSubmatch(output); matches != nil {
		result.TestsPassed, _ = strconv.Atoi(matches[1])
	}

	failedRegex := regexp.MustCompile(`(\d+) failed`)
	if matches := failedRegex.FindStringSubmatch(output); matches != nil {
		result.TestsFailed, _ = strconv.Atoi(matches[1])
	}

	skippedRegex := regexp.MustCompile(`(\d+) skipped`)
	if matches := skippedRegex.FindStringSubmatch(output); matches != nil {
		result.TestsSkipped, _ = strconv.Atoi(matches[1])
	}

	// Parse coverage
	coverageRegex := regexp.MustCompile(`TOTAL\s+\d+\s+\d+\s+(\d+)%`)
	if matches := coverageRegex.FindStringSubmatch(output); matches != nil {
		if cov, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Coverage = cov
		}
	}

	// Parse failures
	failureRegex := regexp.MustCompile(`FAILED (.+?)::(.+?) - (.+)`)
	for _, matches := range failureRegex.FindAllStringSubmatch(output, -1) {
		if len(matches) >= 4 {
			result.Failures = append(result.Failures, TestFailure{
				Name:    matches[2],
				File:    matches[1],
				Message: matches[3],
			})
		}
	}

	return result
}

func (r *Runner) parseNodeOutput(output string) *RunResult {
	result := &RunResult{
		Failures: []TestFailure{},
	}

	// Parse Jest output
	// Tests: X passed, Y failed, Z total
	passedRegex := regexp.MustCompile(`Tests:.*?(\d+) passed`)
	if matches := passedRegex.FindStringSubmatch(output); matches != nil {
		result.TestsPassed, _ = strconv.Atoi(matches[1])
	}

	failedRegex := regexp.MustCompile(`Tests:.*?(\d+) failed`)
	if matches := failedRegex.FindStringSubmatch(output); matches != nil {
		result.TestsFailed, _ = strconv.Atoi(matches[1])
	}

	// Parse coverage
	coverageRegex := regexp.MustCompile(`All files\s+\|\s+(\d+\.?\d*)\s+\|`)
	if matches := coverageRegex.FindStringSubmatch(output); matches != nil {
		if cov, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Coverage = cov
		}
	}

	// Parse failures
	failRegex := regexp.MustCompile(`FAIL\s+(.+?)\s*\n`)
	for _, matches := range failRegex.FindAllStringSubmatch(output, -1) {
		if len(matches) >= 2 {
			result.Failures = append(result.Failures, TestFailure{
				File: matches[1],
			})
		}
	}

	return result
}

func (r *Runner) parseCargoOutput(output string) *RunResult {
	result := &RunResult{
		Failures: []TestFailure{},
	}

	// Parse cargo test output
	// test result: ok. X passed; Y failed; Z ignored
	resultRegex := regexp.MustCompile(`test result: (?:ok|FAILED)\. (\d+) passed; (\d+) failed; (\d+) (?:ignored|filtered out)`)
	if matches := resultRegex.FindStringSubmatch(output); matches != nil {
		result.TestsPassed, _ = strconv.Atoi(matches[1])
		result.TestsFailed, _ = strconv.Atoi(matches[2])
		result.TestsSkipped, _ = strconv.Atoi(matches[3])
	}

	// Parse failures
	failRegex := regexp.MustCompile(`---- (.+?) stdout ----`)
	for _, matches := range failRegex.FindAllStringSubmatch(output, -1) {
		if len(matches) >= 2 {
			result.Failures = append(result.Failures, TestFailure{
				Name: matches[1],
			})
		}
	}

	return result
}
