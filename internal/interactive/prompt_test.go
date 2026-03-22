package interactive

import (
	"bufio"
	"strings"
	"testing"

	"github.com/git-automate/git-auto/internal/git"
)

func TestSelectFilesEmpty(t *testing.T) {
	status := &git.Status{Files: []git.FileStatus{}}
	selected, err := SelectFiles(status, bufio.NewReader(strings.NewReader("")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 0 {
		t.Errorf("expected empty selection, got %v", selected)
	}
}

func TestSelectFilesNone(t *testing.T) {
	files := []git.FileStatus{
		{Path: "file1.go", IndexStatus: 'M', WorkTreeStatus: ' '},
		{Path: "file2.go", IndexStatus: '?', WorkTreeStatus: '?'},
	}
	status := &git.Status{Files: files}

	selected, err := SelectFiles(status, bufio.NewReader(strings.NewReader("none\n")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != nil {
		t.Errorf("expected nil selection, got %v", selected)
	}
}

func TestSelectFilesAll(t *testing.T) {
	files := []git.FileStatus{
		{Path: "file1.go", IndexStatus: 'M', WorkTreeStatus: ' '},
		{Path: "file2.go", IndexStatus: '?', WorkTreeStatus: '?'},
		{Path: "file3.go", IndexStatus: 'A', WorkTreeStatus: ' '},
	}
	status := &git.Status{Files: files}

	selected, err := SelectFiles(status, bufio.NewReader(strings.NewReader("all\n")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 3 {
		t.Errorf("expected 3 files, got %d", len(selected))
	}
	for i, f := range selected {
		if f != files[i].Path {
			t.Errorf("expected %s, got %s", files[i].Path, f)
		}
	}
}

func TestSelectFilesSpecific(t *testing.T) {
	files := []git.FileStatus{
		{Path: "file1.go", IndexStatus: 'M', WorkTreeStatus: ' '},
		{Path: "file2.go", IndexStatus: '?', WorkTreeStatus: '?'},
		{Path: "file3.go", IndexStatus: 'A', WorkTreeStatus: ' '},
	}
	status := &git.Status{Files: files}

	selected, err := SelectFiles(status, bufio.NewReader(strings.NewReader("1 3\n")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 2 {
		t.Errorf("expected 2 files, got %d", len(selected))
	}
	if selected[0] != "file1.go" {
		t.Errorf("expected file1.go, got %s", selected[0])
	}
	if selected[1] != "file3.go" {
		t.Errorf("expected file3.go, got %s", selected[1])
	}
}

func TestParseSelection(t *testing.T) {
	files := []git.FileStatus{
		{Path: "file1.go"},
		{Path: "file2.go"},
		{Path: "file3.go"},
		{Path: "file4.go"},
		{Path: "file5.go"},
	}

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single", "1", 1},
		{"multiple", "1 3 5", 3},
		{"all", "all", 5},
		{"none", "none", 0},
		{"empty", "", 0},
		{"comma_separated", "1,2,3", 3},
		{"mixed_spaces_commas", "1, 2, 4", 3},
		{"range", "1-3", 3},
		{"range_with_single", "1-3,5", 4},
		{"range_and_comma_mixed", "1-2, 4-5", 4},
		{"overlapping_range", "1-3,2-4", 4},
		{"range_out_of_bounds", "4-10", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSelection(tt.input, files)
			if len(result) != tt.expected {
				t.Errorf("expected %d files, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestParseSelectionPaths(t *testing.T) {
	files := []git.FileStatus{
		{Path: "file1.go"},
		{Path: "file2.go"},
		{Path: "file3.go"},
		{Path: "file4.go"},
		{Path: "file5.go"},
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"range", "2-4", []string{"file2.go", "file3.go", "file4.go"}},
		{"comma_range", "1,3-4", []string{"file1.go", "file3.go", "file4.go"}},
		{"range_and_single", "1-2,5", []string{"file1.go", "file2.go", "file5.go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSelection(tt.input, files)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d files, got %d", len(tt.expected), len(result))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("index %d: expected %s, got %s", i, tt.expected[i], path)
				}
			}
		})
	}
}

func TestParseSelectionInvalid(t *testing.T) {
	files := []git.FileStatus{
		{Path: "file1.go"},
		{Path: "file2.go"},
	}

	tests := []struct {
		name  string
		input string
	}{
		{"out of range high", "10"},
		{"out of range low", "0"},
		{"negative", "-1"},
		{"non-numeric", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSelection(tt.input, files)
			if len(result) != 0 {
				t.Errorf("expected empty result for invalid input %q, got %v", tt.input, result)
			}
		})
	}
}

func TestConfirmCommitMessageAccept(t *testing.T) {
	message, proceed, err := ConfirmCommitMessage("test message", bufio.NewReader(strings.NewReader("y\n")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed {
		t.Error("expected proceed to be true")
	}
	if message != "test message" {
		t.Errorf("expected 'test message', got %s", message)
	}
}

func TestConfirmCommitMessageReject(t *testing.T) {
	message, proceed, err := ConfirmCommitMessage("test message", bufio.NewReader(strings.NewReader("n\n")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proceed {
		t.Error("expected proceed to be false")
	}
	if message != "" {
		t.Errorf("expected empty message, got %s", message)
	}
}

func TestConfirmCommitMessageEdit(t *testing.T) {
	message, proceed, err := ConfirmCommitMessage("test message", bufio.NewReader(strings.NewReader("e\nnew message\n")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed {
		t.Error("expected proceed to be true")
	}
	if message != "new message" {
		t.Errorf("expected 'new message', got %s", message)
	}
}

func TestConfirmCommitMessageEditEmpty(t *testing.T) {
	message, proceed, err := ConfirmCommitMessage("test message", bufio.NewReader(strings.NewReader("e\n\n")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed {
		t.Error("expected proceed to be true")
	}
	if message != "test message" {
		t.Errorf("expected 'test message', got %s", message)
	}
}
