package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level represents the logging level
type Level int

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

func (l Level) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelTrace:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

// Logger is a simple structured logger
type Logger struct {
	mu       sync.Mutex
	level    Level
	output   io.Writer
	file     *os.File
	prefix   string
	colorize bool
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Default returns the default logger instance
func Default() *Logger {
	once.Do(func() {
		defaultLogger = New(LevelInfo)
	})
	return defaultLogger
}

// New creates a new logger with the specified level
func New(level Level) *Logger {
	return &Logger{
		level:    level,
		output:   os.Stderr,
		colorize: true,
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetPrefix sets the logger prefix
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// EnableColor enables/disables colorized output
func (l *Logger) EnableColor(enable bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.colorize = enable
}

// SetLogFile sets up file logging
func (l *Logger) SetLogFile(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Close previous file if any
	if l.file != nil {
		l.file.Close()
	}

	l.file = f
	l.output = io.MultiWriter(os.Stderr, f)
	return nil
}

// Close closes the log file if open
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// log writes a log message at the specified level
func (l *Logger) log(level Level, format string, args ...any) {
	if level > l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	levelStr := level.String()

	// Apply colors if enabled
	if l.colorize {
		levelStr = colorize(level, levelStr)
	}

	message := fmt.Sprintf(format, args...)

	var prefix string
	if l.prefix != "" {
		prefix = fmt.Sprintf("[%s] ", l.prefix)
	}

	fmt.Fprintf(l.output, "%s %s %s%s\n", timestamp, levelStr, prefix, message)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...any) {
	l.log(LevelWarn, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...any) {
	l.log(LevelDebug, format, args...)
}

// Trace logs a trace message
func (l *Logger) Trace(format string, args ...any) {
	l.log(LevelTrace, format, args...)
}

// colorize applies ANSI colors to the level string
func colorize(level Level, s string) string {
	var color string
	switch level {
	case LevelError:
		color = "\033[1;31m" // Bold red
	case LevelWarn:
		color = "\033[1;33m" // Bold yellow
	case LevelInfo:
		color = "\033[1;36m" // Bold cyan
	case LevelDebug:
		color = "\033[1;35m" // Bold magenta
	case LevelTrace:
		color = "\033[90m" // Gray
	default:
		return s
	}
	return color + s + "\033[0m"
}

// Package-level convenience functions using the default logger

// Error logs an error message
func Error(format string, args ...any) {
	Default().Error(format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...any) {
	Default().Warn(format, args...)
}

// Info logs an info message
func Info(format string, args ...any) {
	Default().Info(format, args...)
}

// Debug logs a debug message
func Debug(format string, args ...any) {
	Default().Debug(format, args...)
}

// Trace logs a trace message
func Trace(format string, args ...any) {
	Default().Trace(format, args...)
}

// SetLevel sets the default logger's level
func SetLevel(level Level) {
	Default().SetLevel(level)
}

// SetVerbose enables verbose (debug) logging
func SetVerbose(verbose bool) {
	if verbose {
		SetLevel(LevelDebug)
	}
}

// SetDebug enables debug/trace logging
func SetDebug(debug bool) {
	if debug {
		SetLevel(LevelTrace)
	}
}

// GetLogDir returns the default log directory
func GetLogDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".flow", "logs"), nil
}

// SetupFileLogging configures file logging with rotation
func SetupFileLogging() error {
	logDir, err := GetLogDir()
	if err != nil {
		return err
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("flow_%s.log", time.Now().Format("2006-01-02")))
	return Default().SetLogFile(logFile)
}
