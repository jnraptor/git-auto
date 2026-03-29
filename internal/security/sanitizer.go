package security

import (
	"path/filepath"
	"strings"
)

// Blocklist patterns for files that should never be committed.
// Patterns are matched against the file path.
var defaultBlocklist = []string{
	".ssh/",
	".aws/",
	".env",
	"*.pem",
	"id_rsa",
	"secrets.yaml",
	"id_dsa",
	"id_ecdsa",
	"id_ed25519",
}

// Sanitizer checks staged files against a blocklist.
type Sanitizer struct {
	blocklist []string
}

// NewSanitizer creates a sanitizer with the default blocklist.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{blocklist: defaultBlocklist}
}

// NewSanitizerWithBlocklist creates a sanitizer with a custom blocklist.
func NewSanitizerWithBlocklist(blocklist []string) *Sanitizer {
	return &Sanitizer{blocklist: blocklist}
}

// BlockedFile represents a file that matched the blocklist.
type BlockedFile struct {
	Path    string
	Pattern string
}

// CheckStagedFiles checks a list of file paths against the blocklist.
// Returns all files that match a blocklist pattern.
func (s *Sanitizer) CheckStagedFiles(files []string) []BlockedFile {
	var blocked []BlockedFile
	for _, file := range files {
		if pattern, isBlocked := s.isBlocked(file); isBlocked {
			blocked = append(blocked, BlockedFile{
				Path:    file,
				Pattern: pattern,
			})
		}
	}
	return blocked
}

// isBlocked checks if a file path matches any blocklist pattern.
// Returns the matching pattern and whether it's blocked.
func (s *Sanitizer) isBlocked(path string) (string, bool) {
	for _, pattern := range s.blocklist {
		if matchesPattern(path, pattern) {
			return pattern, true
		}
	}
	return "", false
}

// matchesPattern checks if a path matches a blocklist pattern.
// Supports:
//   - Exact prefix match (e.g., ".ssh/")
//   - Exact suffix match (e.g., ".env", ".pem")
//   - Wildcard patterns (e.g., "*.pem")
//   - Exact filename match (e.g., "id_rsa", "secrets.yaml")
func matchesPattern(path, pattern string) bool {
	lowerPath := strings.ToLower(path)
	lowerPattern := strings.ToLower(pattern)

	// Handle directory prefix patterns (ends with /)
	if strings.HasSuffix(lowerPattern, "/") {
		return strings.HasPrefix(lowerPath, lowerPattern) ||
			strings.Contains(lowerPath, "/"+lowerPattern) ||
			strings.Contains(lowerPath, lowerPattern)
	}

	// Handle wildcard patterns (*.ext)
	if strings.HasPrefix(lowerPattern, "*") {
		ext := lowerPattern[1:] // Remove the *
		return strings.HasSuffix(lowerPath, ext)
	}

	// Handle exact match or contains match
	if strings.Contains(lowerPath, lowerPattern) {
		return true
	}

	// Try glob matching
	matched, err := filepath.Match(lowerPattern, filepath.Base(lowerPath))
	if err == nil && matched {
		return true
	}

	return false
}

// GetDefaultBlocklist returns the default blocklist patterns.
func GetDefaultBlocklist() []string {
	result := make([]string, len(defaultBlocklist))
	copy(result, defaultBlocklist)
	return result
}
