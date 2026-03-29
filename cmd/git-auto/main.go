package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/git-automate/git-auto/config"
	"github.com/git-automate/git-auto/internal/git"
	"github.com/git-automate/git-auto/internal/interactive"
	"github.com/git-automate/git-auto/internal/llm"
	"github.com/git-automate/git-auto/internal/push"
	"github.com/git-automate/git-auto/internal/security"
)

var (
	allFlag             = flag.Bool("a", false, "Stage all changed files")
	allFlagLong         = flag.Bool("all", false, "Stage all changed files")
	messageFlag         = flag.String("m", "", "Commit message (if not provided, generate via LLM)")
	dryRunFlag          = flag.Bool("dry-run", false, "Show what would be done without executing")
	forcePushFlag       = flag.Bool("force-push", false, "Force push (use with caution)")
	tagFlag             = flag.String("tag", "", "Create and push a tag after successful push")
	interactiveFlag     = flag.Bool("interactive", false, "Interactive mode: select files and confirm commit message")
	customPromptFlag    = flag.String("prompt", "", "Custom prompt template for LLM (use %s for diff placeholder)")
	maxDiffFlag         = flag.Int("max-diff", 0, "Maximum characters of diff to send to LLM (0 = unlimited)")
	diffThreshold       = flag.Int("diff-threshold", 0, "Character threshold for intelligent diff summarization (0 = no summarization)")
	noSecurity          = flag.Bool("no-security", false, "Disable security checks (blocklist and redaction)")
	verboseSecurityFlag = flag.Bool("verbose-security", false, "Show security warnings when redaction occurs")
)

func main() {
	flag.Parse()

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Could not determine current directory:", err)
		os.Exit(1)
	}

	gitRunner := git.NewRunner(dir)

	cfg := config.Load()
	if *messageFlag == "" && cfg == nil {
		fmt.Fprintln(os.Stderr, "Error: Either provide -m flag or set OPENAI_API_KEY environment variable")
		flag.Usage()
		os.Exit(1)
	}

	status, err := gitRunner.Status()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Failed to get git status:", err)
		os.Exit(1)
	}

	if !status.HasChanges() {
		if *tagFlag == "" {
			fmt.Println("No changes to commit.")
			os.Exit(0)
		}
		fmt.Println("No changes to commit.")
		fmt.Printf("Creating tag: %s\n", *tagFlag)
		if err := gitRunner.Tag(*tagFlag); err != nil {
			fmt.Fprintln(os.Stderr, "Error: Failed to create tag:", err)
			os.Exit(1)
		}
		fmt.Printf("Pushing tag: %s\n", *tagFlag)
		if err := gitRunner.PushTags(); err != nil {
			fmt.Fprintln(os.Stderr, "Error: Failed to push tag:", err)
			os.Exit(1)
		}
		fmt.Println("Tag pushed successfully.")
		os.Exit(0)
	}

	fmt.Printf("Found %d changed file(s)\n", len(status.Files))
	for _, f := range status.Files {
		fmt.Printf("  %c%c %s\n", f.IndexStatus, f.WorkTreeStatus, f.Path)
	}
	fmt.Println()

	if *dryRunFlag {
		fmt.Println("[Dry run] Would stage files")
		if *allFlag || *allFlagLong {
			fmt.Println("[Dry run] Would stage all files")
		}
		if *messageFlag != "" {
			fmt.Printf("[Dry run] Would commit with message: %s\n", *messageFlag)
		} else {
			fmt.Println("[Dry run] Would generate commit message via LLM")
		}
		fmt.Println("[Dry run] Would push to remote")
		os.Exit(0)
	}

	var filesToStage []string
	var bufReader *bufio.Reader

	if *interactiveFlag {
		bufReader = bufio.NewReader(os.Stdin)
		selected, err := interactive.SelectFiles(status, bufReader)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		if len(selected) == 0 {
			fmt.Println("No files selected. Exiting.")
			os.Exit(0)
		}
		filesToStage = selected
		fmt.Printf("Selected %d file(s):\n", len(filesToStage))
		for _, f := range filesToStage {
			fmt.Printf("  %s\n", f)
		}
	} else if *allFlag || *allFlagLong {
		if err := gitRunner.AddAll(); err != nil {
			fmt.Fprintln(os.Stderr, "Error: Failed to stage files:", err)
			os.Exit(1)
		}
		fmt.Println("Staged all files.")
	} else {
		stagedFiles := []string{}
		for _, f := range status.Files {
			if f.IndexStatus == '?' {
				stagedFiles = append(stagedFiles, f.Path)
			}
		}
		if len(stagedFiles) > 0 {
			if err := gitRunner.Add(stagedFiles...); err != nil {
				fmt.Fprintln(os.Stderr, "Error: Failed to stage files:", err)
				os.Exit(1)
			}
			fmt.Printf("Staged %d untracked file(s).\n", len(stagedFiles))
		} else {
			fmt.Println("No untracked files to stage. Use -a to stage all changes.")
		}
	}

	if len(filesToStage) > 0 {
		if err := gitRunner.Add(filesToStage...); err != nil {
			fmt.Fprintln(os.Stderr, "Error: Failed to stage files:", err)
			os.Exit(1)
		}
		fmt.Printf("Staged %d file(s).\n", len(filesToStage))
	}

	// Security Layer 1: Blocklist check (always on unless --no-security)
	var securityProcessor *security.Processor
	if !*noSecurity {
		securityProcessor = security.NewProcessor()
		
		stagedFiles, err := gitRunner.StagedFiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: Failed to get staged files:", err)
			os.Exit(1)
		}

		blockedFiles := securityProcessor.ProcessStagedFiles(stagedFiles)
		if len(blockedFiles) > 0 {
			fmt.Fprintln(os.Stderr, "\n[Security] Sensitive files detected in staging area:")
			for _, bf := range blockedFiles {
				fmt.Fprintf(os.Stderr, "  - %s (matched pattern: %s)\n", bf.Path, bf.Pattern)
			}
			
			// Get paths to unstage
			var pathsToUnstage []string
			for _, bf := range blockedFiles {
				pathsToUnstage = append(pathsToUnstage, bf.Path)
			}
			
			// Unstage the blocked files
			for _, path := range pathsToUnstage {
				if err := gitRunner.UnstageFile(path); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to unstage %s: %v\n", path, err)
				}
			}
			fmt.Fprintf(os.Stderr, "[Security] Auto-unstaged %d file(s). These files will not be committed.\n", len(pathsToUnstage))
			fmt.Fprintln(os.Stderr, "[Security] Add the following to .gitignore to prevent future accidental staging:")
			fmt.Fprintln(os.Stderr, "  .ssh/ .aws/ .env *.pem id_rsa* secrets.yaml")
			fmt.Fprintln(os.Stderr, "")
		}
	}

	diff, err := gitRunner.Diff()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Failed to get diff:", err)
		os.Exit(1)
	}

	// Security Layer 2: Redact sensitive content in diff (always on unless --no-security)
	if !*noSecurity && securityProcessor != nil {
		redactionResult := securityProcessor.ProcessDiff(diff)
		if redactionResult.RedactedCount > 0 {
			if *verboseSecurityFlag {
				fmt.Fprintf(os.Stderr, "[Security] Redacted %d sensitive pattern(s) in:\n", redactionResult.RedactedCount)
				for _, file := range redactionResult.RedactedFiles {
					fmt.Fprintf(os.Stderr, "  - %s\n", file)
				}
				fmt.Fprintln(os.Stderr, "[Security] Note: This only affects the diff sent to LLM, not your files.")
				fmt.Fprintln(os.Stderr, "")
			}
			diff = redactionResult.Content
		}
	}

	// Diff summarization: if diff exceeds threshold, replace with summary
	commitMessage := *messageFlag
	if commitMessage == "" {
		fmt.Println("Generating commit message via LLM...")

		if *customPromptFlag != "" {
			llm.SetPromptTemplate(*customPromptFlag)
		}

		// Check if we need to summarize the diff
		diffForLLM := diff
		if *diffThreshold > 0 && len(diff) > *diffThreshold {
			fmt.Printf("Diff exceeds threshold (%d > %d chars), generating summary...\n", len(diff), *diffThreshold)
			
			// Get diff stat for summary
			diffStat, err := gitRunner.DiffStat()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to get diff stat: %v\n", err)
			}
			
			// Get list of staged files
			stagedFiles, err := gitRunner.StagedFiles()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to get staged files: %v\n", err)
			}
			
			// Build summary for LLM
			var summaryBuilder strings.Builder
			summaryBuilder.WriteString("## Summary of Changes\n")
			if diffStat != "" {
				summaryBuilder.WriteString(diffStat)
				summaryBuilder.WriteString("\n")
			}
			
			summaryBuilder.WriteString("\n## Files Changed\n")
			for _, file := range stagedFiles {
				summaryBuilder.WriteString("- " + file + "\n")
			}
			
			summaryBuilder.WriteString("\n## Note\n")
			summaryBuilder.WriteString("The diff is too large to include in full. Please generate a commit message based on the summary above.")
			summaryBuilder.WriteString("\nThe changes include modifications to the listed files.")
			
			diffForLLM = summaryBuilder.String()
		}

		var llmClient *llm.Client
		if *maxDiffFlag > 0 {
			llmClient = llm.NewClientWithMaxDiff(cfg.APIKey, cfg.BaseURL, cfg.Model, *maxDiffFlag)
		} else {
			llmClient = llm.NewClient(cfg.APIKey, cfg.BaseURL, cfg.Model)
		}

		commitMessage, err = llmClient.GenerateCommitMessage(diffForLLM)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: Failed to generate commit message:", err)
			os.Exit(1)
		}
		if !*interactiveFlag {
			fmt.Printf("Generated commit message: %s\n", commitMessage)
		}
	}

	if *interactiveFlag && *messageFlag == "" {
		confirmedMessage, proceed, err := interactive.ConfirmCommitMessage(commitMessage, bufReader)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		if !proceed {
			fmt.Println("Commit cancelled.")
			os.Exit(0)
		}
		commitMessage = confirmedMessage
	}

	commitOutput, err := gitRunner.Commit(commitMessage)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Failed to create commit:", err)
		os.Exit(1)
	}
	fmt.Println("Commit created successfully.")
	if commitOutput != "" {
		fmt.Println(commitOutput)
	}

	fmt.Println("Pushing to remote...")
	pushHandler := push.NewHandler(gitRunner)
	result := pushHandler.Push(*forcePushFlag)

	if result.Success {
		fmt.Println(result.Message)
		if *tagFlag != "" {
			fmt.Printf("Creating tag: %s\n", *tagFlag)
			if err := gitRunner.Tag(*tagFlag); err != nil {
				fmt.Fprintln(os.Stderr, "Error: Failed to create tag:", err)
				os.Exit(1)
			}
			fmt.Printf("Pushing tag: %s\n", *tagFlag)
			if err := gitRunner.PushTags(); err != nil {
				fmt.Fprintln(os.Stderr, "Error: Failed to push tag:", err)
				os.Exit(1)
			}
			fmt.Println("Tag pushed successfully.")
		}
	} else {
		fmt.Fprintln(os.Stderr, "Push failed:", result.Message)
		if result.HasConflict {
			fmt.Fprintln(os.Stderr, "\nMerge conflicts must be resolved manually.")
			fmt.Fprintln(os.Stderr, "After resolving, run git-auto again to complete the push.")
		}
		if result.NeedsMerge && !result.HasConflict {
			fmt.Fprintln(os.Stderr, "\nPlease resolve any issues and run git-auto again.")
		}
		os.Exit(1)
	}
}
