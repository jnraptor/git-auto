package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct {
	dir string
}

func NewRunner(dir string) *Runner {
	return &Runner{dir: dir}
}

func (r *Runner) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", &GitError{Command: "git " + strings.Join(args, " "), Stderr: stderr.String(), Err: err}
		}
		return "", &GitError{Command: "git " + strings.Join(args, " "), Err: err}
	}

	return string(out), nil
}

type GitError struct {
	Command string
	Stderr  string
	Err     error
}

func (e *GitError) Error() string {
	if e.Stderr != "" {
		return "git error: " + e.Command + " - " + e.Stderr
	}
	return "git error: " + e.Command + " - " + e.Err.Error()
}

func (e *GitError) Unwrap() error {
	return e.Err
}

func (r *Runner) Status() (*Status, error) {
	output, err := r.Run("status", "--porcelain=v1", "-z")
	if err != nil {
		return nil, err
	}

	return parseStatusPorcelainV1Z(output), nil
}

func parseStatusPorcelainV1Z(output string) *Status {
	status := &Status{Files: make([]FileStatus, 0)}
	if output == "" {
		return status
	}

	records := bytes.Split([]byte(output), []byte{0})
	for i := 0; i < len(records); i++ {
		record := records[i]
		if len(record) == 0 {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			continue
		}

		fileStatus := FileStatus{
			IndexStatus:    record[0],
			WorkTreeStatus: record[1],
			Path:           string(record[3:]),
		}

		if fileStatus.IndexStatus == 'R' || fileStatus.IndexStatus == 'C' || fileStatus.WorkTreeStatus == 'R' || fileStatus.WorkTreeStatus == 'C' {
			if i+1 < len(records) && len(records[i+1]) > 0 {
				fileStatus.Path = string(records[i+1])
				i++
			}
		}

		status.Files = append(status.Files, fileStatus)
	}

	return status
}

func (r *Runner) Diff() (string, error) {
	return r.Run("diff", "--staged")
}

func (r *Runner) DiffAll() (string, error) {
	return r.Run("diff")
}

// DiffStat returns a statistical summary of the staged diff.
// Example output: " 5 files changed, 150 insertions(+), 50 deletions(-)"
func (r *Runner) DiffStat() (string, error) {
	return r.Run("diff", "--staged", "--stat")
}

// DiffStatFormat returns a formatted diff stat with a custom width.
func (r *Runner) DiffStatFormat(width int) (string, error) {
	return r.Run("diff", "--staged", fmt.Sprintf("--stat-width=%d", width))
}

// StagedFiles returns a list of files currently staged for commit.
func (r *Runner) StagedFiles() ([]string, error) {
	output, err := r.Run("diff", "--staged", "--name-only")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSpace(output), "\n"), nil
}

// DiffSummary contains parsed information about a diff.
type DiffSummary struct {
	Stat        string
	Files       []string
	FileCount   int
	Insertions  int
	Deletions   int
	Truncated   bool
}

func (r *Runner) Add(paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := r.Run(args...)
	return err
}

func (r *Runner) AddAll() error {
	_, err := r.Run("add", "-A")
	return err
}

// UnstageFile removes a file from the staging area (but keeps changes in working tree).
func (r *Runner) UnstageFile(path string) error {
	_, err := r.Run("reset", "HEAD", "--", path)
	return err
}

// UnstageAll removes all files from the staging area.
func (r *Runner) UnstageAll() error {
	_, err := r.Run("reset", "HEAD")
	return err
}

func (r *Runner) Commit(message string) (string, error) {
	return r.Run("commit", "-m", message)
}

func (r *Runner) Push() error {
	_, err := r.Run("push")
	return err
}

func (r *Runner) Pull() error {
	_, err := r.Run("pull", "--no-rebase")
	return err
}

func (r *Runner) HasConflicts() (bool, error) {
	output, err := r.Run("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

func (r *Runner) CurrentBranch() (string, error) {
	output, err := r.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (r *Runner) Tag(name string) error {
	_, err := r.Run("tag", name)
	return err
}

func (r *Runner) PushTags() error {
	_, err := r.Run("push", "--tags")
	return err
}

// GetStagedContent returns the content of a file as staged (from the index).
func (r *Runner) GetStagedContent(path string) (string, error) {
	return r.Run("show", ":"+path)
}

// HashObject computes the git object hash for the given content.
func (r *Runner) HashObject(content []byte) (string, error) {
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Dir = r.dir
	cmd.Stdin = bytes.NewReader(content)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", &GitError{Command: "git hash-object -w --stdin", Stderr: stderr.String(), Err: err}
		}
		return "", &GitError{Command: "git hash-object -w --stdin", Err: err}
	}
	
	return strings.TrimSpace(stdout.String()), nil
}

// UpdateIndex updates the index entry for a file with the given object hash.
func (r *Runner) UpdateIndex(path, hash string) error {
	_, err := r.Run("update-index", "--add", "--cacheinfo", "100644", hash, path)
	return err
}

// UpdateIndexFromContent writes content as a blob and updates the index entry.
// This modifies only the index, not the working tree.
func (r *Runner) UpdateIndexFromContent(path string, content []byte) error {
	hash, err := r.HashObject(content)
	if err != nil {
		return fmt.Errorf("failed to create blob: %w", err)
	}
	
	return r.UpdateIndex(path, hash)
}
