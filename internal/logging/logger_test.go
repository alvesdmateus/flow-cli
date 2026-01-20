package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Note: sync is used for concurrent access tests

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelError, "ERROR"},
		{LevelWarn, "WARN"},
		{LevelInfo, "INFO"},
		{LevelDebug, "DEBUG"},
		{LevelTrace, "TRACE"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestLevel_Ordering(t *testing.T) {
	// Verify level ordering (lower = more severe)
	if LevelError >= LevelWarn {
		t.Error("LevelError should be less than LevelWarn")
	}
	if LevelWarn >= LevelInfo {
		t.Error("LevelWarn should be less than LevelInfo")
	}
	if LevelInfo >= LevelDebug {
		t.Error("LevelInfo should be less than LevelDebug")
	}
	if LevelDebug >= LevelTrace {
		t.Error("LevelDebug should be less than LevelTrace")
	}
}

func TestNew(t *testing.T) {
	logger := New(LevelDebug)

	if logger == nil {
		t.Fatal("New() returned nil")
	}

	if logger.level != LevelDebug {
		t.Errorf("level = %v, want %v", logger.level, LevelDebug)
	}

	if logger.output == nil {
		t.Error("output should not be nil")
	}

	if !logger.colorize {
		t.Error("colorize should be true by default")
	}
}

func TestLogger_SetLevel(t *testing.T) {
	logger := New(LevelInfo)

	logger.SetLevel(LevelDebug)

	if logger.level != LevelDebug {
		t.Errorf("level = %v, want %v", logger.level, LevelDebug)
	}
}

func TestLogger_SetOutput(t *testing.T) {
	logger := New(LevelInfo)
	var buf bytes.Buffer

	logger.SetOutput(&buf)
	logger.EnableColor(false)
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("output should contain 'test message', got: %q", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("output should contain 'INFO', got: %q", output)
	}
}

func TestLogger_SetPrefix(t *testing.T) {
	logger := New(LevelInfo)
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.EnableColor(false)

	logger.SetPrefix("MyApp")
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[MyApp]") {
		t.Errorf("output should contain '[MyApp]', got: %q", output)
	}
}

func TestLogger_EnableColor(t *testing.T) {
	logger := New(LevelInfo)
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// With color
	logger.EnableColor(true)
	logger.Error("colored")
	coloredOutput := buf.String()

	buf.Reset()

	// Without color
	logger.EnableColor(false)
	logger.Error("plain")
	plainOutput := buf.String()

	// Colored output should contain ANSI escape codes
	if !strings.Contains(coloredOutput, "\033[") {
		t.Error("colored output should contain ANSI escape codes")
	}

	// Plain output should not contain ANSI escape codes
	if strings.Contains(plainOutput, "\033[") {
		t.Error("plain output should not contain ANSI escape codes")
	}
}

func TestLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name       string
		logLevel   Level
		logFunc    func(*Logger)
		shouldLog  bool
		levelInOut string
	}{
		{"Error at Info level", LevelInfo, func(l *Logger) { l.Error("msg") }, true, "ERROR"},
		{"Warn at Info level", LevelInfo, func(l *Logger) { l.Warn("msg") }, true, "WARN"},
		{"Info at Info level", LevelInfo, func(l *Logger) { l.Info("msg") }, true, "INFO"},
		{"Debug at Info level", LevelInfo, func(l *Logger) { l.Debug("msg") }, false, "DEBUG"},
		{"Trace at Info level", LevelInfo, func(l *Logger) { l.Trace("msg") }, false, "TRACE"},
		{"Debug at Debug level", LevelDebug, func(l *Logger) { l.Debug("msg") }, true, "DEBUG"},
		{"Trace at Debug level", LevelDebug, func(l *Logger) { l.Trace("msg") }, false, "TRACE"},
		{"Trace at Trace level", LevelTrace, func(l *Logger) { l.Trace("msg") }, true, "TRACE"},
		{"Error at Error level", LevelError, func(l *Logger) { l.Error("msg") }, true, "ERROR"},
		{"Warn at Error level", LevelError, func(l *Logger) { l.Warn("msg") }, false, "WARN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.logLevel)
			var buf bytes.Buffer
			logger.SetOutput(&buf)
			logger.EnableColor(false)

			tt.logFunc(logger)

			output := buf.String()
			hasLevel := strings.Contains(output, tt.levelInOut)

			if tt.shouldLog && !hasLevel {
				t.Errorf("expected log output with %s, got: %q", tt.levelInOut, output)
			}
			if !tt.shouldLog && hasLevel {
				t.Errorf("expected no log output, got: %q", output)
			}
		})
	}
}

func TestLogger_FormatArgs(t *testing.T) {
	logger := New(LevelInfo)
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.EnableColor(false)

	logger.Info("Hello %s, count: %d", "World", 42)

	output := buf.String()
	if !strings.Contains(output, "Hello World, count: 42") {
		t.Errorf("output should contain formatted message, got: %q", output)
	}
}

func TestLogger_SetLogFile(t *testing.T) {
	logger := New(LevelInfo)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "logging_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")

	err = logger.SetLogFile(logPath)
	if err != nil {
		t.Fatalf("SetLogFile() error = %v", err)
	}
	defer logger.Close()

	// Write a log message
	logger.EnableColor(false)
	logger.Info("test file logging")

	// Close to flush
	logger.Close()

	// Read the log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test file logging") {
		t.Errorf("log file should contain message, got: %q", string(content))
	}
}

func TestLogger_SetLogFile_CreatesDirectory(t *testing.T) {
	logger := New(LevelInfo)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "logging_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a nested path that doesn't exist
	logPath := filepath.Join(tmpDir, "nested", "dir", "test.log")

	err = logger.SetLogFile(logPath)
	if err != nil {
		t.Fatalf("SetLogFile() error = %v", err)
	}
	defer logger.Close()

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(logPath)); os.IsNotExist(err) {
		t.Error("SetLogFile should create parent directories")
	}
}

func TestLogger_SetLogFile_ReplacesExisting(t *testing.T) {
	logger := New(LevelInfo)

	tmpDir, err := os.MkdirTemp("", "logging_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath1 := filepath.Join(tmpDir, "test1.log")
	logPath2 := filepath.Join(tmpDir, "test2.log")

	// Set first log file
	err = logger.SetLogFile(logPath1)
	if err != nil {
		t.Fatalf("SetLogFile(1) error = %v", err)
	}

	// Set second log file (should close first)
	err = logger.SetLogFile(logPath2)
	if err != nil {
		t.Fatalf("SetLogFile(2) error = %v", err)
	}
	defer logger.Close()

	// Write to logger
	logger.EnableColor(false)
	logger.Info("second file only")
	logger.Close()

	// Second file should have the message
	content2, _ := os.ReadFile(logPath2)
	if !strings.Contains(string(content2), "second file only") {
		t.Error("second log file should contain the message")
	}
}

func TestLogger_Close(t *testing.T) {
	logger := New(LevelInfo)

	// Close without file should not error
	err := logger.Close()
	if err != nil {
		t.Errorf("Close() without file should not error, got: %v", err)
	}

	// Close with file
	tmpDir, err := os.MkdirTemp("", "logging_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	logger.SetLogFile(logPath)

	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should not error
	err = logger.Close()
	if err != nil {
		t.Errorf("second Close() should not error, got: %v", err)
	}
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	logger := New(LevelDebug)
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.EnableColor(false)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				logger.Info("goroutine %d iteration %d", n, j)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic and output should have content
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected log output from concurrent writes")
	}
}

func TestColorize(t *testing.T) {
	tests := []struct {
		level Level
		input string
	}{
		{LevelError, "ERROR"},
		{LevelWarn, "WARN"},
		{LevelInfo, "INFO"},
		{LevelDebug, "DEBUG"},
		{LevelTrace, "TRACE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := colorize(tt.level, tt.input)

			// Should contain the original text
			if !strings.Contains(result, tt.input) {
				t.Errorf("colorize() should contain %q, got: %q", tt.input, result)
			}

			// Should contain ANSI codes
			if !strings.Contains(result, "\033[") {
				t.Errorf("colorize() should contain ANSI codes, got: %q", result)
			}

			// Should end with reset code
			if !strings.HasSuffix(result, "\033[0m") {
				t.Errorf("colorize() should end with reset code, got: %q", result)
			}
		})
	}
}

func TestColorize_UnknownLevel(t *testing.T) {
	input := "UNKNOWN"
	result := colorize(Level(99), input)

	// Unknown level should return unchanged
	if result != input {
		t.Errorf("colorize(unknown) = %q, want %q", result, input)
	}
}

func TestDefault(t *testing.T) {
	logger1 := Default()
	logger2 := Default()

	if logger1 == nil {
		t.Fatal("Default() returned nil")
	}

	// Should return same instance
	if logger1 != logger2 {
		t.Error("Default() should return singleton instance")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Test that package-level functions don't panic
	// We can't easily capture output due to sync.Once singleton pattern,
	// but we can verify they execute without error

	// Set level high enough to allow all logs
	logger := Default()
	logger.SetLevel(LevelTrace)

	// These should not panic
	Error("error msg %d", 1)
	Warn("warn msg %d", 2)
	Info("info msg %d", 3)
	Debug("debug msg %d", 4)
	Trace("trace msg %d", 5)

	// Reset to default level
	logger.SetLevel(LevelInfo)
}

func TestSetLevel_PackageLevel(t *testing.T) {
	// Save original level
	logger := Default()
	origLevel := logger.level
	defer logger.SetLevel(origLevel)

	SetLevel(LevelDebug)

	if logger.level != LevelDebug {
		t.Errorf("SetLevel() should update default logger level")
	}

	SetLevel(LevelTrace)
	if logger.level != LevelTrace {
		t.Errorf("SetLevel() should update to Trace")
	}
}

func TestSetVerbose(t *testing.T) {
	logger := Default()
	origLevel := logger.level
	defer logger.SetLevel(origLevel)

	// Reset to a known state
	logger.SetLevel(LevelInfo)

	SetVerbose(true)

	if logger.level != LevelDebug {
		t.Error("SetVerbose(true) should set level to Debug")
	}

	// SetVerbose(false) should not change level
	logger.SetLevel(LevelError)
	SetVerbose(false)
	if logger.level != LevelError {
		t.Error("SetVerbose(false) should not change level")
	}
}

func TestSetDebug(t *testing.T) {
	logger := Default()
	origLevel := logger.level
	defer logger.SetLevel(origLevel)

	// Reset to a known state
	logger.SetLevel(LevelInfo)

	SetDebug(true)

	if logger.level != LevelTrace {
		t.Error("SetDebug(true) should set level to Trace")
	}

	// SetDebug(false) should not change level
	logger.SetLevel(LevelError)
	SetDebug(false)
	if logger.level != LevelError {
		t.Error("SetDebug(false) should not change level")
	}
}

func TestGetLogDir(t *testing.T) {
	dir, err := GetLogDir()

	if err != nil {
		t.Fatalf("GetLogDir() error = %v", err)
	}

	if dir == "" {
		t.Error("GetLogDir() should return non-empty path")
	}

	if !strings.Contains(dir, ".flow") {
		t.Error("GetLogDir() should contain .flow directory")
	}

	if !strings.Contains(dir, "logs") {
		t.Error("GetLogDir() should contain logs directory")
	}
}

func TestTimestampFormat(t *testing.T) {
	logger := New(LevelInfo)
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.EnableColor(false)

	logger.Info("test")

	output := buf.String()

	// Should contain timestamp in format HH:MM:SS.mmm
	// Example: 15:04:05.000
	if len(output) < 12 {
		t.Fatalf("output too short: %q", output)
	}

	// Check timestamp format (starts with HH:MM:SS)
	timestamp := output[:12]
	if timestamp[2] != ':' || timestamp[5] != ':' || timestamp[8] != '.' {
		t.Errorf("timestamp format incorrect: %q", timestamp)
	}
}
