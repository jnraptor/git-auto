package push

import (
	"testing"

	"github.com/git-automate/git-auto/internal/git"
)

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		substrs   []string
		wantFound bool
	}{
		{
			name:      "found at start",
			s:         "! [rejected] some error",
			substrs:   []string{"! [rejected]"},
			wantFound: true,
		},
		{
			name:      "found at end",
			s:         "some error Updates were rejected",
			substrs:   []string{"Updates were rejected"},
			wantFound: true,
		},
		{
			name:      "not found",
			s:         "some error message",
			substrs:   []string{"! [rejected]"},
			wantFound: false,
		},
		{
			name:      "empty string",
			s:         "",
			substrs:   []string{"test"},
			wantFound: false,
		},
		{
			name:      "empty substrings",
			s:         "some string",
			substrs:   []string{},
			wantFound: false,
		},
		{
			name:      "matches any substring",
			s:         "authentication failed for some reason",
			substrs:   []string{"permission denied", "authentication failed", "could not authenticate"},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAny(tt.s, tt.substrs)
			if got != tt.wantFound {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrs, got, tt.wantFound)
			}
		})
	}
}

func TestIsRejectedError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "rejected message",
			stderr: "error: failed to push some refs\n! [rejected] master -> master (non-fast-forward)",
			want:   true,
		},
		{
			name:   "updates were rejected",
			stderr: "error: Updates were rejected because the tip of your current branch is behind",
			want:   true,
		},
		{
			name:   "fetch first",
			stderr: "error: failed to push due to fetch first",
			want:   true,
		},
		{
			name:   "auth error",
			stderr: "error: authentication failed",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &git.GitError{Stderr: tt.stderr}
			got := isRejectedError(err)
			if got != tt.want {
				t.Errorf("isRejectedError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "authentication failed",
			stderr: "error: authentication failed",
			want:   true,
		},
		{
			name:   "permission denied",
			stderr: "error: permission denied (publickey)",
			want:   true,
		},
		{
			name:   "could not authenticate",
			stderr: "error: could not authenticate",
			want:   true,
		},
		{
			name:   "401 status",
			stderr: "HTTP 401",
			want:   true,
		},
		{
			name:   "403 status",
			stderr: "HTTP 403",
			want:   true,
		},
		{
			name:   "rejected error",
			stderr: "! [rejected] non-fast-forward",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &git.GitError{Stderr: tt.stderr}
			got := isAuthError(err)
			if got != tt.want {
				t.Errorf("isAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}
