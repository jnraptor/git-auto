package security

import (
	"testing"
)

func TestSanitizer_BlocksSensitiveFiles(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name     string
		files    []string
		wantLen  int
		wantPath string
	}{
		{
			name:     "blocks .ssh directory",
			files:    []string{".ssh/id_rsa", ".ssh/authorized_keys"},
			wantLen:  2,
			wantPath: ".ssh/id_rsa",
		},
		{
			name:     "blocks .aws directory",
			files:    []string{".aws/credentials", ".aws/config"},
			wantLen:  2,
			wantPath: ".aws/credentials",
		},
		{
			name:     "blocks .env files",
			files:    []string{".env", ".env.local", "config/app.go"},
			wantLen:  2,
			wantPath: ".env",
		},
		{
			name:     "blocks .pem files",
			files:    []string{"cert.pem", "private/key.pem", "main.go"},
			wantLen:  2,
			wantPath: "cert.pem",
		},
		{
			name:     "blocks id_rsa",
			files:    []string{"id_rsa", "id_rsa.pub", "src/main.go"},
			wantLen:  2,
			wantPath: "id_rsa",
		},
		{
			name:     "blocks secrets.yaml",
			files:    []string{"secrets.yaml", "k8s/secrets.yaml", "config.yaml"},
			wantLen:  2,
			wantPath: "secrets.yaml",
		},
		{
			name:     "safe files pass through",
			files:    []string{"main.go", "README.md", "src/utils.js"},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := sanitizer.CheckStagedFiles(tt.files)
			if len(blocked) != tt.wantLen {
				t.Errorf("CheckStagedFiles() returned %d files, want %d", len(blocked), tt.wantLen)
			}
			if tt.wantLen > 0 && blocked[0].Path != tt.wantPath {
				t.Errorf("CheckStagedFiles() first blocked path = %v, want %v", blocked[0].Path, tt.wantPath)
			}
		})
	}
}

func TestSanitizer_PatternMatching(t *testing.T) {
	tests := []struct {
		path    string
		blocked bool
	}{
		{".ssh/id_rsa", true},
		{"home/.ssh/config", true},
		{".aws/credentials", true},
		{".env", true},
		{".env.local", true},
		{"src/.env", true},
		{"cert.pem", true},
		{"keys/private.pem", true},
		{"id_rsa", true},
		{"secrets.yaml", true},
		{"main.go", false},
		{"README.md", false},
		{"package.json", false},
	}

	sanitizer := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			blocked := sanitizer.CheckStagedFiles([]string{tt.path})
			gotBlocked := len(blocked) > 0
			if gotBlocked != tt.blocked {
				t.Errorf("CheckStagedFiles(%q) blocked = %v, want %v", tt.path, gotBlocked, tt.blocked)
			}
		})
	}
}

func TestSanitizer_CustomBlocklist(t *testing.T) {
	customBlocklist := []string{"*.secret", "private/"}
	sanitizer := NewSanitizerWithBlocklist(customBlocklist)

	tests := []struct {
		path    string
		blocked bool
	}{
		{"config.secret", true},
		{"private/keys.txt", true},
		{".env", false},
		{"main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			blocked := sanitizer.CheckStagedFiles([]string{tt.path})
			gotBlocked := len(blocked) > 0
			if gotBlocked != tt.blocked {
				t.Errorf("CheckStagedFiles(%q) blocked = %v, want %v", tt.path, gotBlocked, tt.blocked)
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{".ssh/id_rsa", ".ssh/", true},
		{"home/.ssh/config", ".ssh/", true},
		{".env", ".env", true},
		{".env.local", ".env", true},
		{"cert.pem", "*.pem", true},
		{"keys/private.pem", "*.pem", true},
		{"secrets.yaml", "secrets.yaml", true},
		{"main.go", "*.pem", false},
		{"main.go", ".env", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchesPattern(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestGetDefaultBlocklist(t *testing.T) {
	blocklist := GetDefaultBlocklist()
	if len(blocklist) == 0 {
		t.Error("GetDefaultBlocklist() returned empty blocklist")
	}

	// Check that common patterns are present
	expectedPatterns := []string{".ssh/", ".aws/", ".env", "*.pem", "id_rsa", "secrets.yaml"}
	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range blocklist {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetDefaultBlocklist() missing expected pattern: %s", expected)
		}
	}
}

func TestIsSensitiveFileName(t *testing.T) {
	tests := []struct {
		path      string
		sensitive bool
	}{
		{".env", true},
		{".env.local", true},
		{".env.production", true},
		{"credentials.json", true},
		{"secrets.yaml", true},
		{"secrets.json", true},
		{"main.go", false},
		{"config.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsSensitiveFileName(tt.path)
			if got != tt.sensitive {
				t.Errorf("IsSensitiveFileName(%q) = %v, want %v", tt.path, got, tt.sensitive)
			}
		})
	}
}
