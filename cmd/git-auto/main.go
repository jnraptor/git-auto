package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/git-automate/git-auto/config"
	"github.com/git-automate/git-auto/internal/git"
	"github.com/git-automate/git-auto/internal/interactive"
	"github.com/git-automate/git-auto/internal/llm"
	"github.com/git-automate/git-auto/internal/push"
)

var (
	allFlag          = flag.Bool("a", false, "Stage all changed files")
	allFlagLong      = flag.Bool("all", false, "Stage all changed files")
	messageFlag      = flag.String("m", "", "Commit message (if not provided, generate via LLM)")
	dryRunFlag       = flag.Bool("dry-run", false, "Show what would be done without executing")
	forcePushFlag    = flag.Bool("force-push", false, "Force push (use with caution)")
	tagFlag          = flag.String("tag", "", "Create and push a tag after successful push")
	interactiveFlag  = flag.Bool("interactive", false, "Interactive mode: select files and confirm commit message")
	customPromptFlag = flag.String("prompt", "", "Custom prompt template for LLM (use %s for diff placeholder)")
	maxDiffFlag      = flag.Int("max-diff", 0, "Maximum characters of diff to send to LLM (0 = unlimited)")
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

	diff, err := gitRunner.Diff()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Failed to get diff:", err)
		os.Exit(1)
	}

	commitMessage := *messageFlag
	if commitMessage == "" {
		fmt.Println("Generating commit message via LLM...")

		if *customPromptFlag != "" {
			llm.SetPromptTemplate(*customPromptFlag)
		}

		var llmClient *llm.Client
		if *maxDiffFlag > 0 {
			llmClient = llm.NewClientWithMaxDiff(cfg.APIKey, cfg.BaseURL, cfg.Model, *maxDiffFlag)
		} else {
			llmClient = llm.NewClient(cfg.APIKey, cfg.BaseURL, cfg.Model)
		}

		commitMessage, err = llmClient.GenerateCommitMessage(diff)
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
