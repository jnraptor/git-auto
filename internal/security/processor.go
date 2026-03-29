package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Processor combines sanitization and redaction for security checks.
type Processor struct {
	sanitizer *Sanitizer
	redactor  *Redactor
}

// NewProcessor creates a new security processor with default settings.
func NewProcessor() *Processor {
	return &Processor{
		sanitizer: NewSanitizer(),
		redactor:  NewRedactor(),
	}
}

// NewProcessorWithConfig creates a processor with custom settings.
func NewProcessorWithConfig(blocklist []string, patterns []RedactionPattern) *Processor {
	return &Processor{
		sanitizer: NewSanitizerWithBlocklist(blocklist),
		redactor:  NewRedactorWithPatterns(patterns),
	}
}

// CheckResult contains the results of security checks.
type CheckResult struct {
	BlockedFiles   []BlockedFile
	SensitiveFiles []SensitiveFile
	TotalBlocked   int
	TotalRedacted  int
}

// SensitiveFile represents a file with sensitive content.
type SensitiveFile struct {
	Path    string
	Patterns []string
}

// ProcessStagedFiles performs Layer 1 (blocklist) checks on staged files.
// Returns files that should be unstaged.
func (p *Processor) ProcessStagedFiles(files []string) []BlockedFile {
	return p.sanitizer.CheckStagedFiles(files)
}

// ProcessDiff performs Layer 2 (redaction) on diff content.
// Returns the redacted diff and details about what was found.
func (p *Processor) ProcessDiff(diff string) RedactionResult {
	return p.redactor.RedactContent(diff)
}

// GetBlockedFilesForUnstage returns just the file paths that should be unstaged.
func (p *Processor) GetBlockedFilesForUnstage(files []string) []string {
	blocked := p.ProcessStagedFiles(files)
	paths := make([]string, len(blocked))
	for i, f := range blocked {
		paths[i] = f.Path
	}
	return paths
}

// ShouldBlockCommit checks if any critical files are staged that would block the commit.
// Returns true if files should be completely blocked (not just unstaged).
func (p *Processor) ShouldBlockCommit(files []string) bool {
	for _, file := range files {
		lower := strings.ToLower(file)
		// Block private keys completely
		if strings.Contains(lower, "id_rsa") || strings.Contains(lower, "id_dsa") ||
			strings.Contains(lower, "id_ecdsa") || strings.Contains(lower, "id_ed25519") {
			return true
		}
	}
	return false
}

// FormatBlockedFiles returns a human-readable string of blocked files.
func FormatBlockedFiles(blocked []BlockedFile) string {
	if len(blocked) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Blocked %d sensitive file(s):\n", len(blocked)))
	for _, f := range blocked {
		sb.WriteString(fmt.Sprintf("  - %s (matched: %s)\n", f.Path, f.Pattern))
	}
	return sb.String()
}

// FormatRedactionResult returns a human-readable string of redaction results.
func FormatRedactionResult(result RedactionResult) string {
	if result.RedactedCount == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Redacted %d sensitive pattern(s):\n", result.RedactedCount))
	for _, pattern := range result.RedactedPatterns {
		sb.WriteString(fmt.Sprintf("  - %s\n", pattern))
	}
	return sb.String()
}

// IsSensitiveFileName checks if a filename matches sensitive patterns.
func IsSensitiveFileName(path string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(base)

	sensitiveNames := []string{
		".env", ".env.local", ".env.production", ".env.development",
		"credentials", "credentials.json",
		"service-account", "service-account.json",
		"keystore.jks", "truststore.jks",
		"secrets.yaml", "secrets.yml", "secrets.json",
	}

	for _, name := range sensitiveNames {
		if lower == name {
			return true
		}
	}

	return false
}
