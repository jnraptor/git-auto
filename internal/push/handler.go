package push

import (
	"fmt"

	"github.com/git-automate/git-auto/internal/git"
)

type Handler struct {
	git *git.Runner
}

func NewHandler(gitRunner *git.Runner) *Handler {
	return &Handler{git: gitRunner}
}

type PushResult struct {
	Success     bool
	NeedsMerge  bool
	HasConflict bool
	Message     string
}

func (h *Handler) Push(force bool) *PushResult {
	err := h.git.Push()
	if err == nil {
		return &PushResult{
			Success: true,
			Message: "Push successful",
		}
	}

	gitErr, ok := err.(*git.GitError)
	if !ok {
		return &PushResult{
			Success: false,
			Message: err.Error(),
		}
	}

	if isRejectedError(gitErr) {
		return h.handleRejected()
	}

	if isAuthError(gitErr) {
		return &PushResult{
			Success: false,
			Message: "Authentication failed. Ensure you have push access and proper credentials.",
		}
	}

	return &PushResult{
		Success: false,
		Message: gitErr.Error(),
	}
}

func (h *Handler) handleRejected() *PushResult {
	fmt.Println("Push rejected (non-fast-forward). Attempting to pull and merge...")

	if err := h.git.Pull(); err != nil {
		gitErr, ok := err.(*git.GitError)
		if ok && isAuthError(gitErr) {
			return &PushResult{
				Success:    false,
				NeedsMerge: true,
				Message:    "Pull failed due to authentication issues.",
			}
		}
		return &PushResult{
			Success:    false,
			NeedsMerge: true,
			Message:    "Pull failed: " + err.Error(),
		}
	}

	hasConflicts, err := h.git.HasConflicts()
	if err != nil {
		return &PushResult{
			Success:    false,
			NeedsMerge: true,
			Message:    "Failed to check for conflicts: " + err.Error(),
		}
	}

	if hasConflicts {
		return &PushResult{
			Success:     false,
			NeedsMerge:  true,
			HasConflict: true,
			Message:     "Merge conflict detected. Please resolve conflicts manually using `git mergetool`, then run `git-auto` again.",
		}
	}

	fmt.Println("Merge successful. Retrying push...")
	if err := h.git.Push(); err != nil {
		return &PushResult{
			Success:    false,
			NeedsMerge: true,
			Message:    "Push failed after merge: " + err.Error(),
		}
	}

	return &PushResult{
		Success: true,
		Message: "Push successful after merge",
	}
}

func isRejectedError(err *git.GitError) bool {
	stderr := err.Stderr
	return containsAny(stderr, []string{
		"! [rejected]",
		"non-fast-forward",
		"fetch first",
		"Updates were rejected",
	})
}

func isAuthError(err *git.GitError) bool {
	stderr := err.Stderr
	return containsAny(stderr, []string{
		"authentication failed",
		"permission denied",
		"could not authenticate",
		"401",
		"403",
	})
}

func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
