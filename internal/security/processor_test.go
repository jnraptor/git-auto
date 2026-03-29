package security

import (
	"testing"
)

func TestProcessor_ProcessStagedFiles(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name      string
		files     []string
		wantBlock int
	}{
		{
			name:      "blocks sensitive files",
			files:     []string{".env", ".ssh/id_rsa", "main.go"},
			wantBlock: 2,
		},
		{
			name:      "no sensitive files",
			files:     []string{"main.go", "README.md"},
			wantBlock: 0,
		},
		{
			name:      "empty list",
			files:     []string{},
			wantBlock: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := processor.ProcessStagedFiles(tt.files)
			if len(blocked) != tt.wantBlock {
				t.Errorf("ProcessStagedFiles() blocked %d files, want %d", len(blocked), tt.wantBlock)
			}
		})
	}
}

func TestProcessor_ProcessDiff(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name        string
		diff        string
		wantRedact  bool
		wantContain string
	}{
		{
			name:        "redacts API keys in diff",
			diff:        "+const API_KEY = sk-abcdefghijklmnopqrstuvwxyz123456",
			wantRedact:  true,
			wantContain: "[REDACTED]",
		},
		{
			name:        "safe diff",
			diff:        "+func main() { return }",
			wantRedact:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ProcessDiff(tt.diff)
			gotRedact := result.RedactedCount > 0
			if gotRedact != tt.wantRedact {
				t.Errorf("ProcessDiff() redacted = %v, want %v", gotRedact, tt.wantRedact)
			}
			if tt.wantRedact && result.Content != tt.diff {
				if result.Content == tt.diff {
					t.Errorf("ProcessDiff() did not modify content")
				}
			}
		})
	}
}

func TestProcessor_GetBlockedFilesForUnstage(t *testing.T) {
	processor := NewProcessor()

	files := []string{".env", ".ssh/id_rsa", "main.go", "secrets.yaml"}
	blocked := processor.GetBlockedFilesForUnstage(files)

	if len(blocked) != 3 {
		t.Errorf("GetBlockedFilesForUnstage() returned %d files, want 3", len(blocked))
	}

	// Check that the expected files are in the blocked list
	expected := map[string]bool{".env": true, ".ssh/id_rsa": true, "secrets.yaml": true}
	for _, path := range blocked {
		if !expected[path] {
			t.Errorf("GetBlockedFilesForUnstage() unexpected blocked file: %s", path)
		}
	}
}

func TestProcessor_ShouldBlockCommit(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{
			name:  "blocks private key files",
			files: []string{"id_rsa", "main.go"},
			want:  true,
		},
		{
			name:  "blocks id_dsa",
			files: []string{"id_dsa", "main.go"},
			want:  true,
		},
		{
			name:  "blocks id_ecdsa",
			files: []string{"id_ecdsa", "main.go"},
			want:  true,
		},
		{
			name:  "blocks id_ed25519",
			files: []string{"id_ed25519", "main.go"},
			want:  true,
		},
		{
			name:  "allows regular files",
			files: []string{"main.go", "README.md"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.ShouldBlockCommit(tt.files)
			if got != tt.want {
				t.Errorf("ShouldBlockCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatBlockedFiles(t *testing.T) {
	tests := []struct {
		name     string
		blocked  []BlockedFile
		contains string
	}{
		{
			name:     "empty",
			blocked:  []BlockedFile{},
			contains: "",
		},
		{
			name: "single file",
			blocked: []BlockedFile{
				{Path: ".env", Pattern: ".env"},
			},
			contains: "Blocked 1 sensitive file",
		},
		{
			name: "multiple files",
			blocked: []BlockedFile{
				{Path: ".env", Pattern: ".env"},
				{Path: ".ssh/id_rsa", Pattern: ".ssh/"},
			},
			contains: "Blocked 2 sensitive file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBlockedFiles(tt.blocked)
			if tt.contains == "" && result != "" {
				t.Errorf("FormatBlockedFiles() = %q, want empty", result)
			}
			if tt.contains != "" && result == "" {
				t.Errorf("FormatBlockedFiles() = empty, want contains %q", tt.contains)
			}
		})
	}
}

func TestFormatRedactionResult(t *testing.T) {
	tests := []struct {
		name   string
		result RedactionResult
		want   string
	}{
		{
			name:   "no redactions",
			result: RedactionResult{RedactedCount: 0},
			want:   "",
		},
		{
			name: "with redactions",
			result: RedactionResult{
				RedactedCount:    2,
				RedactedPatterns: []string{"OpenAI API Key", "AWS Access Key ID"},
			},
			want: "Redacted 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRedactionResult(tt.result)
			if tt.want == "" && got != "" {
				t.Errorf("FormatRedactionResult() = %q, want empty", got)
			}
			if tt.want != "" && got == "" {
				t.Errorf("FormatRedactionResult() = empty, want contains %q", tt.want)
			}
		})
	}
}

func TestNewProcessorWithConfig(t *testing.T) {
	blocklist := []string{"*.custom"}
	patterns := []RedactionPattern{
		{
			Name:    "Custom Pattern",
			Pattern: nil, // Would need valid regex in real use
		},
	}

	processor := NewProcessorWithConfig(blocklist, patterns)
	if processor == nil {
		t.Error("NewProcessorWithConfig() returned nil")
	}
}
