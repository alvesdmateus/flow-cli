package testing

import (
	"testing"
	"time"
)

func TestNewRunner(t *testing.T) {
	runner := NewRunner(RunnerConfig{
		WorkDir:     "/test",
		ProjectType: ProjectGo,
		Coverage:    true,
		Timeout:     "10m",
	})

	if runner.workDir != "/test" {
		t.Errorf("workDir = %q, expected %q", runner.workDir, "/test")
	}
	if runner.projectType != ProjectGo {
		t.Errorf("projectType = %v, expected %v", runner.projectType, ProjectGo)
	}
	if !runner.coverage {
		t.Error("coverage should be true")
	}
	if runner.timeout != 10*time.Minute {
		t.Errorf("timeout = %v, expected %v", runner.timeout, 10*time.Minute)
	}
}

func TestNewRunner_InvalidTimeout(t *testing.T) {
	runner := NewRunner(RunnerConfig{
		Timeout: "invalid",
	})

	// Should default to 5 minutes
	if runner.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, expected %v", runner.timeout, 5*time.Minute)
	}
}

func TestParseGoOutput(t *testing.T) {
	runner := &Runner{projectType: ProjectGo}

	output := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"pkg","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"pkg","Test":"TestFoo","Output":"=== RUN   TestFoo\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"pkg","Test":"TestFoo","Elapsed":0.001}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"pkg","Test":"TestBar"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"pkg","Test":"TestBar","Output":"=== RUN   TestBar\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"pkg","Test":"TestBar","Output":"    bar_test.go:10: expected 1, got 2\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"pkg","Test":"TestBar","Elapsed":0.002}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"pkg","Test":"TestSkip"}
{"Time":"2024-01-01T00:00:00Z","Action":"skip","Package":"pkg","Test":"TestSkip","Elapsed":0}
coverage: 75.5% of statements
`

	result := runner.parseGoOutput(output)

	if result.TestsPassed != 1 {
		t.Errorf("TestsPassed = %d, expected 1", result.TestsPassed)
	}
	if result.TestsFailed != 1 {
		t.Errorf("TestsFailed = %d, expected 1", result.TestsFailed)
	}
	if result.TestsSkipped != 1 {
		t.Errorf("TestsSkipped = %d, expected 1", result.TestsSkipped)
	}
	if result.Coverage != 75.5 {
		t.Errorf("Coverage = %v, expected 75.5", result.Coverage)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Failures len = %d, expected 1", len(result.Failures))
	}
	if result.Failures[0].Name != "TestBar" {
		t.Errorf("Failure name = %q, expected %q", result.Failures[0].Name, "TestBar")
	}
}

func TestParsePytestOutput(t *testing.T) {
	runner := &Runner{projectType: ProjectPython}

	output := `============================= test session starts ==============================
collected 5 items

test_app.py::test_add PASSED
test_app.py::test_subtract PASSED
test_app.py::test_multiply FAILED
test_app.py::test_divide SKIPPED

FAILED test_app.py::test_multiply - AssertionError: Expected 6, got 5

---------- coverage: platform linux, python 3.10 -----------
Name         Stmts   Miss  Cover
--------------------------------
app.py          20      4    80%
--------------------------------
TOTAL           20      4    80%

========================= 2 passed, 1 failed, 1 skipped ==========================
`

	result := runner.parsePytestOutput(output)

	if result.TestsPassed != 2 {
		t.Errorf("TestsPassed = %d, expected 2", result.TestsPassed)
	}
	if result.TestsFailed != 1 {
		t.Errorf("TestsFailed = %d, expected 1", result.TestsFailed)
	}
	if result.TestsSkipped != 1 {
		t.Errorf("TestsSkipped = %d, expected 1", result.TestsSkipped)
	}
	if result.Coverage != 80 {
		t.Errorf("Coverage = %v, expected 80", result.Coverage)
	}
}

func TestParseNodeOutput(t *testing.T) {
	runner := &Runner{projectType: ProjectNode}

	output := `PASS src/utils.test.ts
FAIL src/app.test.ts
  ● test suite failed to run

Test Suites: 1 failed, 1 passed, 2 total
Tests:       3 passed, 2 failed, 5 total

----------|---------|----------|---------|---------|-------------------
File      | % Stmts | % Branch | % Funcs | % Lines | Uncovered Line #s
----------|---------|----------|---------|---------|-------------------
All files |   72.5  |    60    |   80    |   72.5  |
----------|---------|----------|---------|---------|-------------------
`

	result := runner.parseNodeOutput(output)

	if result.TestsPassed != 3 {
		t.Errorf("TestsPassed = %d, expected 3", result.TestsPassed)
	}
	if result.TestsFailed != 2 {
		t.Errorf("TestsFailed = %d, expected 2", result.TestsFailed)
	}
	if result.Coverage != 72.5 {
		t.Errorf("Coverage = %v, expected 72.5", result.Coverage)
	}
}

func TestParseCargoOutput(t *testing.T) {
	runner := &Runner{projectType: ProjectRust}

	output := `   Compiling myproject v0.1.0
    Finished test [unoptimized + debuginfo] target(s) in 1.23s
     Running target/debug/deps/myproject-abc123

running 3 tests
test tests::test_add ... ok
test tests::test_subtract ... ok
test tests::test_multiply ... FAILED

failures:

---- tests::test_multiply stdout ----
thread 'tests::test_multiply' panicked at 'assertion failed'

failures:
    tests::test_multiply

test result: FAILED. 2 passed; 1 failed; 0 filtered out; finished in 0.01s
`

	result := runner.parseCargoOutput(output)

	if result.TestsPassed != 2 {
		t.Errorf("TestsPassed = %d, expected 2", result.TestsPassed)
	}
	if result.TestsFailed != 1 {
		t.Errorf("TestsFailed = %d, expected 1", result.TestsFailed)
	}
	if len(result.Failures) < 1 {
		t.Error("expected at least 1 failure")
	}
}
