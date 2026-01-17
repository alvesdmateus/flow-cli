// Package security provides security utilities for command and path sanitization.
package security

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"unicode"
)

// CommandSanitizer provides command injection prevention.
type CommandSanitizer struct {
	// DangerousPatterns are regex patterns that indicate potential injection
	DangerousPatterns []*regexp.Regexp

	// DangerousChars are characters that need escaping
	DangerousChars map[rune]bool

	// BlockedEnvVars are environment variable patterns to block
	BlockedEnvVars []*regexp.Regexp
}

// NewCommandSanitizer creates a new command sanitizer with default patterns.
func NewCommandSanitizer() *CommandSanitizer {
	cs := &CommandSanitizer{
		DangerousChars: map[rune]bool{
			';':  true, // Command separator
			'&':  true, // Background/AND
			'|':  true, // Pipe
			'`':  true, // Command substitution
			'$':  true, // Variable expansion
			'(':  true, // Subshell
			')':  true, // Subshell
			'{':  true, // Brace expansion
			'}':  true, // Brace expansion
			'<':  true, // Redirection
			'>':  true, // Redirection
			'\n': true, // Newline (command separator)
			'\r': true, // Carriage return
			'!':  true, // History expansion
		},
	}

	// Dangerous command patterns
	dangerousPatterns := []string{
		`\$\(.*\)`,           // Command substitution $(...)
		`\x60.*\x60`,         // Backtick command substitution
		`\$\{.*\}`,           // Variable expansion ${...}
		`;\s*\w+`,            // Command chaining with ;
		`\|\s*\w+`,           // Piping
		`&&\s*\w+`,           // AND chaining
		`\|\|\s*\w+`,         // OR chaining
		`>\s*/`,              // Redirect to root paths
		`>>\s*/`,             // Append to root paths
		`<\s*/etc/`,          // Read from /etc
		`\beval\b`,           // eval command
		`\bexec\b`,           // exec command
		`\bsource\b`,         // source command
		`\b\.\s+/`,           // dot source
		`\\x[0-9a-fA-F]{2}`,  // Hex escape sequences
		`\\[0-7]{1,3}`,       // Octal escape sequences
		`%[0-9a-fA-F]{2}`,    // URL encoding
		`\bnc\s+-`,           // netcat
		`\bcurl\s+.*\|\s*sh`, // curl pipe to shell
		`\bwget\s+.*\|\s*sh`, // wget pipe to shell
	}

	for _, p := range dangerousPatterns {
		if re, err := regexp.Compile(p); err == nil {
			cs.DangerousPatterns = append(cs.DangerousPatterns, re)
		}
	}

	// Blocked environment variable patterns
	blockedEnvPatterns := []string{
		`(?i)password`,
		`(?i)secret`,
		`(?i)api.?key`,
		`(?i)token`,
		`(?i)credential`,
		`(?i)private.?key`,
	}

	for _, p := range blockedEnvPatterns {
		if re, err := regexp.Compile(p); err == nil {
			cs.BlockedEnvVars = append(cs.BlockedEnvVars, re)
		}
	}

	return cs
}

// SanitizeResult contains the result of command sanitization.
type SanitizeResult struct {
	Original    string
	Sanitized   string
	IsModified  bool
	Warnings    []string
	IsDangerous bool
	Reason      string
}

// SanitizeCommand checks a command for injection attempts and sanitizes it.
func (cs *CommandSanitizer) SanitizeCommand(command string) *SanitizeResult {
	result := &SanitizeResult{
		Original:  command,
		Sanitized: command,
		Warnings:  make([]string, 0),
	}

	// Check for dangerous patterns
	for _, pattern := range cs.DangerousPatterns {
		if pattern.MatchString(command) {
			result.IsDangerous = true
			result.Reason = fmt.Sprintf("command contains dangerous pattern: %s", pattern.String())
			result.Warnings = append(result.Warnings, result.Reason)
		}
	}

	// Check for null bytes
	if strings.Contains(command, "\x00") {
		result.IsDangerous = true
		result.Reason = "command contains null bytes"
		result.Warnings = append(result.Warnings, result.Reason)
	}

	// Check for excessive length (potential buffer overflow attempt)
	if len(command) > 10000 {
		result.IsDangerous = true
		result.Reason = "command exceeds maximum length"
		result.Warnings = append(result.Warnings, result.Reason)
	}

	return result
}

// EscapeArgument escapes a single argument for safe shell use.
func (cs *CommandSanitizer) EscapeArgument(arg string) string {
	if arg == "" {
		return "''"
	}

	// Check if escaping is needed
	needsEscape := false
	for _, r := range arg {
		if cs.DangerousChars[r] || unicode.IsSpace(r) {
			needsEscape = true
			break
		}
	}

	if !needsEscape {
		return arg
	}

	// Use single quotes for escaping (safest method)
	// Single quotes preserve everything literally except single quotes themselves
	var escaped strings.Builder
	escaped.WriteRune('\'')

	for _, r := range arg {
		if r == '\'' {
			// End quote, add escaped quote, start new quote
			escaped.WriteString("'\"'\"'")
		} else {
			escaped.WriteRune(r)
		}
	}

	escaped.WriteRune('\'')
	return escaped.String()
}

// EscapeArgumentWindows escapes an argument for Windows cmd.exe.
func (cs *CommandSanitizer) EscapeArgumentWindows(arg string) string {
	if arg == "" {
		return `""`
	}

	// Windows uses different escaping rules
	// Double quotes need to be escaped with backslash
	var escaped strings.Builder
	escaped.WriteRune('"')

	backslashCount := 0
	for _, r := range arg {
		switch r {
		case '\\':
			backslashCount++
		case '"':
			// Escape backslashes before quote
			for i := 0; i < backslashCount; i++ {
				escaped.WriteRune('\\')
			}
			escaped.WriteRune('\\')
			escaped.WriteRune('"')
			backslashCount = 0
		default:
			for i := 0; i < backslashCount; i++ {
				escaped.WriteRune('\\')
			}
			escaped.WriteRune(r)
			backslashCount = 0
		}
	}

	// Handle trailing backslashes
	for i := 0; i < backslashCount; i++ {
		escaped.WriteRune('\\')
	}

	escaped.WriteRune('"')
	return escaped.String()
}

// EscapeForPlatform escapes an argument for the current platform.
func (cs *CommandSanitizer) EscapeForPlatform(arg string) string {
	if runtime.GOOS == "windows" {
		return cs.EscapeArgumentWindows(arg)
	}
	return cs.EscapeArgument(arg)
}

// ValidateCommandStructure validates the basic structure of a command.
func (cs *CommandSanitizer) ValidateCommandStructure(command string) error {
	command = strings.TrimSpace(command)

	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Check for unbalanced quotes
	singleQuotes := 0
	doubleQuotes := 0
	escaped := false

	for _, r := range command {
		if escaped {
			escaped = false
			continue
		}

		switch r {
		case '\\':
			escaped = true
		case '\'':
			singleQuotes++
		case '"':
			doubleQuotes++
		}
	}

	if singleQuotes%2 != 0 {
		return fmt.Errorf("unbalanced single quotes")
	}
	if doubleQuotes%2 != 0 {
		return fmt.Errorf("unbalanced double quotes")
	}

	// Check for unbalanced parentheses
	parens := 0
	braces := 0
	brackets := 0

	for _, r := range command {
		switch r {
		case '(':
			parens++
		case ')':
			parens--
		case '{':
			braces++
		case '}':
			braces--
		case '[':
			brackets++
		case ']':
			brackets--
		}

		if parens < 0 || braces < 0 || brackets < 0 {
			return fmt.Errorf("unbalanced brackets")
		}
	}

	if parens != 0 || braces != 0 || brackets != 0 {
		return fmt.Errorf("unbalanced brackets")
	}

	return nil
}

// SplitCommand splits a command into its components safely.
func (cs *CommandSanitizer) SplitCommand(command string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		switch r {
		case '\\':
			if inSingleQuote {
				current.WriteRune(r)
			} else {
				escaped = true
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			} else {
				current.WriteRune(r)
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			} else {
				current.WriteRune(r)
			}
		case ' ', '\t':
			if inSingleQuote || inDoubleQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	if inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("unterminated quote")
	}

	return parts, nil
}

// IsSafeCommand checks if a command is considered safe to execute.
func (cs *CommandSanitizer) IsSafeCommand(command string) (bool, string) {
	result := cs.SanitizeCommand(command)
	if result.IsDangerous {
		return false, result.Reason
	}

	if err := cs.ValidateCommandStructure(command); err != nil {
		return false, err.Error()
	}

	return true, ""
}
