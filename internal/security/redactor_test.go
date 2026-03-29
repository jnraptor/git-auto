package security

import (
	"fmt"
	"strings"
	"testing"
)

func TestRedactor_RedactsSensitivePatterns(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantCount   int
		wantContain string
	}{
		{
			name:        "OpenAI API key",
			content:     `sk-abcdefghijklmnopqrstuvwxyz123456`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "AWS Access Key ID",
			content:     `AKIAIOSFODNN7EXAMPLE`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "GitHub token",
			content:     `ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "Bearer token",
			content:     `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "API key assignment",
			content:     `API_KEY=sk-abcdefghijklmnopqrstuvwxyz123456`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "password assignment",
			content:     `password=SuperSecret123`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "token assignment",
			content:     `token=abc123def456ghi789`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "private key header",
			content:     `-----BEGIN RSA PRIVATE KEY-----`,
			wantCount:   1,
			wantContain: "[REDACTED]",
		},
		{
			name:        "safe content",
			content:     `func main() { fmt.Println("hello") }`,
			wantCount:   0,
		},
		{
			name: "multiple secrets",
			content: `api_key=sk-abcdefghijklmnopqrstuvwxyz123456
password=SuperSecret123
token=abc123def456ghi789`,
			wantCount:   3,
			wantContain: "[REDACTED]",
		},
	}

	redactor := NewRedactor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.RedactContent(tt.content)
			if result.RedactedCount != tt.wantCount {
				t.Errorf("RedactContent() redacted count = %d, want %d", result.RedactedCount, tt.wantCount)
			}
			if tt.wantCount > 0 && !strings.Contains(result.Content, tt.wantContain) {
				t.Errorf("RedactContent() content does not contain %q", tt.wantContain)
			}
			if tt.wantCount > 0 && strings.Contains(result.Content, "sk-abc") {
				t.Errorf("RedactContent() content still contains original secret")
			}
		})
	}
}

func TestRedactor_IsSensitive(t *testing.T) {
	tests := []struct {
		content   string
		sensitive bool
	}{
		{`sk-abcdefghijklmnopqrstuvwxyz123456`, true},
		{`AKIAIOSFODNN7EXAMPLE`, true},
		{`password=secret123`, true},
		{`func main() {}`, false},
		{`README.md content`, false},
	}

	redactor := NewRedactor()
	for _, tt := range tests {
		t.Run(tt.content[:min(20, len(tt.content))], func(t *testing.T) {
			got := redactor.IsSensitive(tt.content)
			if got != tt.sensitive {
				t.Errorf("IsSensitive(%q) = %v, want %v", tt.content, got, tt.sensitive)
			}
		})
	}
}

func TestRedactor_GetSensitivePatterns(t *testing.T) {
	content := `sk-abcdefghijklmnopqrstuvwxyz123456
AKIAIOSFODNN7EXAMPLE`

	redactor := NewRedactor()
	patterns := redactor.GetSensitivePatterns(content)

	if len(patterns) < 2 {
		t.Errorf("GetSensitivePatterns() returned %d patterns, want at least 2", len(patterns))
	}
}

func TestRedactor_RedactLine(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		line  string
		want  string
	}{
		{
			line: `api_key=sk-abcdefghijklmnopqrstuvwxyz123456`,
			want: `api_key=[REDACTED]`,
		},
		{
			line: `password=SuperSecret123`,
			want: `[REDACTED]`,
		},
		{
			line: `func main() {}`,
			want: `func main() {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.line[:min(30, len(tt.line))], func(t *testing.T) {
			got := redactor.RedactLine(tt.line)
			if got != tt.want {
				t.Errorf("RedactLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestRedactor_PreservesNonSensitiveContent(t *testing.T) {
	content := `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`

	redactor := NewRedactor()
	result := redactor.RedactContent(content)

	if result.Content != content {
		t.Errorf("RedactContent() modified non-sensitive content")
	}
	if result.RedactedCount != 0 {
		t.Errorf("RedactContent() redacted count = %d, want 0", result.RedactedCount)
	}
}

func TestRedactor_LargeDiff(t *testing.T) {
	// Simulate a large diff with one secret
	var sb strings.Builder
	sb.WriteString("diff --git a/main.go b/main.go\n")
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("+func function%d() {}\n", i))
	}
	sb.WriteString("+const API_KEY = sk-abcdefghijklmnopqrstuvwxyz123456\n")

	redactor := NewRedactor()
	result := redactor.RedactContent(sb.String())

	if result.RedactedCount != 1 {
		t.Errorf("RedactContent() redacted count = %d, want 1", result.RedactedCount)
	}
	if strings.Contains(result.Content, "sk-abc") {
		t.Errorf("RedactContent() still contains original secret")
	}
	if !strings.Contains(result.Content, "[REDACTED]") {
		t.Errorf("RedactContent() does not contain [REDACTED]")
	}
}

func TestGetDefaultPatterns(t *testing.T) {
	patterns := GetDefaultPatterns()
	if len(patterns) == 0 {
		t.Error("GetDefaultPatterns() returned empty patterns")
	}

	// Verify pattern names
	expectedNames := []string{
		"OpenAI API Key",
		"AWS Access Key ID",
		"Generic Bearer Token",
		"Generic Password Assignment",
	}

	for _, name := range expectedNames {
		found := false
		for _, p := range patterns {
			if p.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetDefaultPatterns() missing expected pattern: %s", name)
		}
	}
}

func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "CRLF to LF",
			input: "line1\r\nline2\r\nline3",
			want:  "line1\nline2\nline3",
		},
		{
			name:  "LF unchanged",
			input: "line1\nline2\nline3",
			want:  "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLineEndings(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeLineEndings() = %q, want %q", got, tt.want)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
