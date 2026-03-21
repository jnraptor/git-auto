package git

import (
	"errors"
	"testing"
)

func TestParseStatusPorcelainV1Z(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []FileStatus
	}{
		{
			name:  "preserves spaces in file names",
			input: " M config.yaml\x00?? file with spaces.yaml\x00",
			want: []FileStatus{
				{IndexStatus: ' ', WorkTreeStatus: 'M', Path: "config.yaml"},
				{IndexStatus: '?', WorkTreeStatus: '?', Path: "file with spaces.yaml"},
			},
		},
		{
			name:  "uses destination path for renames",
			input: "R  old name.txt\x00new name.txt\x00",
			want: []FileStatus{
				{IndexStatus: 'R', WorkTreeStatus: ' ', Path: "new name.txt"},
			},
		},
		{
			name:  "empty input returns empty status",
			input: "",
			want:  []FileStatus{},
		},
		{
			name:  "handles copy status",
			input: "C  original.txt\x00copy.txt\x00",
			want: []FileStatus{
				{IndexStatus: 'C', WorkTreeStatus: ' ', Path: "copy.txt"},
			},
		},
		{
			name:  "skips malformed records",
			input: "M\x00config.yaml\x00?? untracked.txt\x00",
			want: []FileStatus{
				{IndexStatus: '?', WorkTreeStatus: '?', Path: "untracked.txt"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStatusPorcelainV1Z(tt.input)
			if len(got.Files) != len(tt.want) {
				t.Fatalf("got %d files, want %d", len(got.Files), len(tt.want))
			}
			for i := range tt.want {
				if got.Files[i] != tt.want[i] {
					t.Fatalf("file %d = %#v, want %#v", i, got.Files[i], tt.want[i])
				}
			}
		})
	}
}

func TestStatusHasChanges(t *testing.T) {
	tests := []struct {
		name   string
		files  []FileStatus
		hasChg bool
	}{
		{
			name:   "empty files",
			files:  []FileStatus{},
			hasChg: false,
		},
		{
			name:   "has staged changes",
			files:  []FileStatus{{IndexStatus: 'M', WorkTreeStatus: ' ', Path: "foo.txt"}},
			hasChg: true,
		},
		{
			name:   "has unstaged changes",
			files:  []FileStatus{{IndexStatus: ' ', WorkTreeStatus: 'M', Path: "foo.txt"}},
			hasChg: true,
		},
		{
			name:   "has untracked files",
			files:  []FileStatus{{IndexStatus: '?', WorkTreeStatus: '?', Path: "foo.txt"}},
			hasChg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Status{Files: tt.files}
			if got := s.HasChanges(); got != tt.hasChg {
				t.Errorf("HasChanges() = %v, want %v", got, tt.hasChg)
			}
		})
	}
}

func TestStatusStagedCount(t *testing.T) {
	tests := []struct {
		name  string
		files []FileStatus
		count int
	}{
		{
			name:  "empty files",
			files: []FileStatus{},
			count: 0,
		},
		{
			name:  "staged modifications count",
			files: []FileStatus{{IndexStatus: 'M', WorkTreeStatus: ' '}},
			count: 1,
		},
		{
			name:  "staged additions count",
			files: []FileStatus{{IndexStatus: 'A', WorkTreeStatus: ' '}},
			count: 1,
		},
		{
			name:  "staged deletions count",
			files: []FileStatus{{IndexStatus: 'D', WorkTreeStatus: ' '}},
			count: 1,
		},
		{
			name:  "untracked files not counted",
			files: []FileStatus{{IndexStatus: '?', WorkTreeStatus: '?'}},
			count: 0,
		},
		{
			name:  "worktree changes not counted",
			files: []FileStatus{{IndexStatus: ' ', WorkTreeStatus: 'M'}},
			count: 0,
		},
		{
			name: "mixed changes",
			files: []FileStatus{
				{IndexStatus: 'M', WorkTreeStatus: ' '},
				{IndexStatus: ' ', WorkTreeStatus: 'M'},
				{IndexStatus: '?', WorkTreeStatus: '?'},
			},
			count: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Status{Files: tt.files}
			if got := s.StagedCount(); got != tt.count {
				t.Errorf("StagedCount() = %v, want %v", got, tt.count)
			}
		})
	}
}

func TestStatusUntrackedCount(t *testing.T) {
	tests := []struct {
		name  string
		files []FileStatus
		count int
	}{
		{
			name:  "empty files",
			files: []FileStatus{},
			count: 0,
		},
		{
			name:  "untracked files count",
			files: []FileStatus{{IndexStatus: '?', WorkTreeStatus: '?'}},
			count: 1,
		},
		{
			name:  "staged files not counted",
			files: []FileStatus{{IndexStatus: 'M', WorkTreeStatus: ' '}},
			count: 0,
		},
		{
			name:  "worktree changes not counted",
			files: []FileStatus{{IndexStatus: ' ', WorkTreeStatus: 'M'}},
			count: 0,
		},
		{
			name: "multiple untracked",
			files: []FileStatus{
				{IndexStatus: '?', WorkTreeStatus: '?'},
				{IndexStatus: '?', WorkTreeStatus: '?'},
			},
			count: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Status{Files: tt.files}
			if got := s.UntrackedCount(); got != tt.count {
				t.Errorf("UntrackedCount() = %v, want %v", got, tt.count)
			}
		})
	}
}

func TestGitError(t *testing.T) {
	t.Run("error with stderr", func(t *testing.T) {
		err := &GitError{
			Command: "git status",
			Stderr:  "fatal: not a git repository",
			Err:     errors.New("exit status 128"),
		}
		want := "git error: git status - fatal: not a git repository"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("error without stderr", func(t *testing.T) {
		err := &GitError{
			Command: "git status",
			Stderr:  "",
			Err:     errors.New("exit status 1"),
		}
		want := "git error: git status - exit status 1"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap returns underlying error", func(t *testing.T) {
		innerErr := errors.New("inner error")
		err := &GitError{
			Command: "git status",
			Stderr:  "",
			Err:     innerErr,
		}
		if got := err.Unwrap(); got != innerErr {
			t.Errorf("Unwrap() = %v, want %v", got, innerErr)
		}
	})
}
