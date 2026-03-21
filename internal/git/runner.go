package git

import (
	"bytes"
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

	return strings.TrimSpace(string(out)), nil
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
	output, err := r.Run("status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}

	status := &Status{
		Files: make([]FileStatus, 0),
	}

	if output == "" {
		return status, nil
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		indexStatus := line[0]
		workTreeStatus := line[1]
		path := strings.TrimSpace(line[3:])

		status.Files = append(status.Files, FileStatus{
			IndexStatus:    indexStatus,
			WorkTreeStatus: workTreeStatus,
			Path:           path,
		})
	}

	return status, nil
}

func (r *Runner) Diff() (string, error) {
	return r.Run("diff", "--staged")
}

func (r *Runner) DiffAll() (string, error) {
	return r.Run("diff")
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

func (r *Runner) Commit(message string) error {
	_, err := r.Run("commit", "-m", message)
	return err
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
	return output != "", nil
}

func (r *Runner) CurrentBranch() (string, error) {
	output, err := r.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}
