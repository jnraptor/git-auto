package interactive

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/git-automate/git-auto/internal/git"
)

func SelectFiles(status *git.Status, r *bufio.Reader) ([]string, error) {
	if len(status.Files) == 0 {
		return nil, nil
	}

	fmt.Println("\nSelect files to stage:")
	fmt.Println("Enter file numbers separated by spaces/commas (e.g., '1 3 5' or '1-4,6'), 'all', or 'none':")
	fmt.Println()

	for i, f := range status.Files {
		fmt.Printf("  [%d] %c%c %s\n", i+1, f.IndexStatus, f.WorkTreeStatus, f.Path)
	}
	fmt.Println()

	fmt.Print("Selection: ")
	input, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	selected := ParseSelection(input, status.Files)
	if len(selected) == 0 {
		return nil, nil
	}

	return selected, nil
}

func ParseSelection(input string, files []git.FileStatus) []string {
	if input == "" || input == "none" {
		return nil
	}

	if input == "all" || input == "a" {
		var allFiles []string
		for _, f := range files {
			allFiles = append(allFiles, f.Path)
		}
		return allFiles
	}

	selected := make([]string, 0)
	seen := make(map[int]bool)

	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				continue
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err1 != nil || err2 != nil {
				continue
			}
			if start < 1 {
				start = 1
			}
			if end > len(files) {
				end = len(files)
			}
			for i := start; i <= end; i++ {
				if !seen[i-1] && i >= 1 && i <= len(files) {
					selected = append(selected, files[i-1].Path)
					seen[i-1] = true
				}
			}
		} else {
			idx, err := strconv.Atoi(part)
			if err != nil {
				continue
			}
			if idx >= 1 && idx <= len(files) && !seen[idx-1] {
				selected = append(selected, files[idx-1].Path)
				seen[idx-1] = true
			}
		}
	}

	return selected
}

func ConfirmCommitMessage(message string, r *bufio.Reader) (string, bool, error) {
	fmt.Printf("\nGenerated commit message: %s\n", message)
	fmt.Println("\nOptions:")
	fmt.Println("  [y] Accept and commit")
	fmt.Println("  [e] Edit the message")
	fmt.Println("  [n] Cancel commit")
	fmt.Println()

	fmt.Print("Choice [y/e/n]: ")
	input, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "y", "yes":
		return message, true, nil
	case "e", "edit":
		fmt.Println("\nCurrent message:", message)
		fmt.Print("Enter new commit message: ")
		newInput, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", false, fmt.Errorf("failed to read input: %w", err)
		}
		newMessage := strings.TrimSpace(newInput)
		if newMessage == "" {
			return message, true, nil
		}
		return newMessage, true, nil
	default:
		return "", false, nil
	}
}
