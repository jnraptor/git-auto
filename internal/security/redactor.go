package security

import (
	"regexp"
	"strings"
)

// Sensitive patterns to redact from diffs.
// These patterns are matched and replaced with [REDACTED].
var defaultPatterns = []RedactionPattern{
	{
		Name:    "OpenAI API Key",
		Pattern: regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),
	},
	{
		Name:    "AWS Access Key ID",
		Pattern: regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	},
	{
		Name:    "AWS Secret Access Key",
		Pattern: regexp.MustCompile(`(?i)aws[_\-]?secret[_\-]?access[_\-]?key\s*[=:]\s*["']?([A-Za-z0-9/+=]{40})["']?`),
	},
	{
		Name:    "GitHub Token",
		Pattern: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,}`),
	},
	{
		Name:    "Generic Bearer Token",
		Pattern: regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`),
	},
	{
		Name:    "Generic API Key Assignment",
		Pattern: regexp.MustCompile(`(?i)(api[_\-]?key|apikey)\s*[=:]\s*["']?[A-Za-z0-9\-._~]{8,}["']?`),
	},
	{
		Name:    "Generic Token Assignment",
		Pattern: regexp.MustCompile(`(?i)(token|access[_\-]?token)\s*[=:]\s*["']?[A-Za-z0-9\-._~]{8,}["']?`),
	},
	{
		Name:    "Generic Password Assignment",
		Pattern: regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*["']?([^\s"']{4,})["']?`),
	},
	{
		Name:    "Private Key Header",
		Pattern: regexp.MustCompile(`-----BEGIN\s+(RSA\s+|EC\s+|OPENSSH\s+)?PRIVATE\s+KEY-----`),
	},
	{
		Name:    "Generic Secret Assignment",
		Pattern: regexp.MustCompile(`(?i)(secret|secret[_\-]?key)\s*[=:]\s*["']?[A-Za-z0-9\-._~]{8,}["']?`),
	},
}

// RedactionPattern defines a pattern that should be redacted.
type RedactionPattern struct {
	Name    string
	Pattern *regexp.Regexp
}

// Redactor masks sensitive data in text content.
type Redactor struct {
	patterns []RedactionPattern
}

// NewRedactor creates a redactor with default patterns.
func NewRedactor() *Redactor {
	patterns := make([]RedactionPattern, len(defaultPatterns))
	copy(patterns, defaultPatterns)
	return &Redactor{patterns: patterns}
}

// NewRedactorWithPatterns creates a redactor with custom patterns.
func NewRedactorWithPatterns(patterns []RedactionPattern) *Redactor {
	return &Redactor{patterns: patterns}
}

// RedactionResult contains the redacted content and information about what was found.
type RedactionResult struct {
	Content          string
	RedactedCount    int
	RedactedPatterns []string
	RedactedFiles    []string
}

// RedactContent redacts sensitive patterns from the given content.
// Returns the redacted content and details about what was found.
func (r *Redactor) RedactContent(content string) RedactionResult {
	result := RedactionResult{
		Content:          content,
		RedactedCount:    0,
		RedactedPatterns: []string{},
		RedactedFiles:    []string{},
	}

	patternsFound := make(map[string]bool)

	for _, pattern := range r.patterns {
		if pattern.Pattern.MatchString(result.Content) {
			result.Content = pattern.Pattern.ReplaceAllString(result.Content, r.getReplacement(pattern.Name))
			result.RedactedCount++
			if !patternsFound[pattern.Name] {
				patternsFound[pattern.Name] = true
				result.RedactedPatterns = append(result.RedactedPatterns, pattern.Name)
			}
		}
	}

	// Identify which files had redactions
	result.RedactedFiles = identifyRedactedFiles(result.Content)

	return result
}

// identifyRedactedFiles parses the diff content and returns files that contain [REDACTED].
func identifyRedactedFiles(content string) []string {
	var redactedFiles []string
	var currentFile string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		// Check for diff header line indicating a new file
		if strings.HasPrefix(line, "diff --git a/") {
			// Extract the file path from "diff --git a/path/to/file b/path/to/file"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				// a/file b/file -> extract a/file
				currentFile = strings.TrimPrefix(parts[2], "a/")
			}
		} else if strings.Contains(line, "[REDACTED]") && currentFile != "" {
			// Check if this file is already in the list
			found := false
			for _, f := range redactedFiles {
				if f == currentFile {
					found = true
					break
				}
			}
			if !found {
				redactedFiles = append(redactedFiles, currentFile)
			}
		}
	}

	return redactedFiles
}

// RedactLine redacts a single line of text.
func (r *Redactor) RedactLine(line string) string {
	return r.RedactContent(line).Content
}

// getReplacement returns a replacement string for a matched pattern.
func (r *Redactor) getReplacement(patternName string) string {
	return "[REDACTED]"
}

// IsSensitive checks if content contains any sensitive patterns.
func (r *Redactor) IsSensitive(content string) bool {
	for _, pattern := range r.patterns {
		if pattern.Pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// GetSensitivePatterns returns the names of patterns found in the content.
func (r *Redactor) GetSensitivePatterns(content string) []string {
	var found []string
	patternsFound := make(map[string]bool)

	for _, pattern := range r.patterns {
		if pattern.Pattern.MatchString(content) && !patternsFound[pattern.Name] {
			patternsFound[pattern.Name] = true
			found = append(found, pattern.Name)
		}
	}

	return found
}

// GetDefaultPatterns returns the default redaction patterns.
func GetDefaultPatterns() []RedactionPattern {
	result := make([]RedactionPattern, len(defaultPatterns))
	for i, p := range defaultPatterns {
		result[i] = RedactionPattern{
			Name:    p.Name,
			Pattern: regexp.MustCompile(p.Pattern.String()),
		}
	}
	return result
}

// NormalizeLineEndings normalizes line endings for consistent processing.
func NormalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}
