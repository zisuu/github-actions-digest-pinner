package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zisuu/github-actions-digest-pinner/internal/finder"
	"github.com/zisuu/github-actions-digest-pinner/internal/ghclient"
	"github.com/zisuu/github-actions-digest-pinner/internal/parser"
	"github.com/zisuu/github-actions-digest-pinner/internal/updater"
)

// Build number and versions injected at compile time
var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "github-actions-digest-pinner",
		Short: "A tool to pin GitHub Actions to specific digests",
		Long: `GitHub Actions Digest Pinner is a tool to help you pin
GitHub Actions to specific digests for better security and reliability.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Fatalf("Failed to display help: %v", err)
			}
		},
	}

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\nCommit: %s\nDate: %s\n", version, commit, date)
		},
	})

	// Add scan command
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the repository for GitHub Actions workflows",
		Run: func(cmd *cobra.Command, args []string) {
			dir, _ := cmd.Flags().GetString("dir")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if verbose {
				log.Println("Starting GitHub Actions digest pinner utility")
				log.Printf("Scanning directory: %s", dir)
			}

			fsys := os.DirFS(dir)

			if verbose {
				log.Println("Finding workflow files...")
			}

			files, err := finder.FindWorkflowFiles(fsys)

			if err != nil {
				log.Fatalf("Failed to find workflow files: %v", err)
			}

			if verbose {
				log.Printf("Found %d workflow files", len(files))
				log.Println("Parsing actions in workflow files...")
			}

			for _, file := range files {
				if verbose {
					log.Printf("Processing file: %s", file)
				}

				content, err := fsys.Open(file)
				if err != nil {
					log.Fatalf("Failed to read file %s: %v", file, err)
				}

				fileContent, err := io.ReadAll(content)
				if err != nil {
					log.Fatalf("Failed to read content of file %s: %v", file, err)
				}

				actions, err := parser.ParseWorkflowActions(fileContent)
				if err != nil {
					log.Fatalf("Failed to parse actions in file %s: %v", file, err)
				}

				log.Printf("Found %d actions in file %s", len(actions), file)

				for _, action := range actions {
					fmt.Printf("- Action: %s/%s@%s\n", action.Owner, action.Repo, action.Ref)
				}
			}
		},
	}

	scanCmd.Flags().String("dir", ".", "Directory containing GitHub workflows")
	scanCmd.Flags().Bool("verbose", false, "Verbose output")

	rootCmd.AddCommand(scanCmd)

	// Add update command
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update GitHub Actions workflows to use pinned digests",
		Run: func(cmd *cobra.Command, args []string) {
			dir, _ := cmd.Flags().GetString("dir")
			timeout, _ := cmd.Flags().GetInt("timeout")
			verbose, _ := cmd.Flags().GetBool("verbose")

			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			defer cancel()

			if verbose {
				log.Println("Starting GitHub Actions digest pinner utility")
				log.Printf("Scanning directory: %s", dir)
			}

			client := ghclient.NewGitHubClient()
			workflowUpdater := updater.NewUpdater(client, dir)
			fsys := os.DirFS(dir)

			if verbose {
				log.Println("Finding workflow files...")
			}

			files, err := finder.FindWorkflowFiles(fsys)

			if err != nil {
				log.Fatalf("Failed to find workflow files: %v", err)
			}

			if verbose {
				log.Printf("Found %d workflow files", len(files))
			}

			totalUpdates, err := workflowUpdater.UpdateWorkflows(ctx, fsys)
			if err != nil {
				log.Fatalf("Failed to update workflows: %v", err)
			}

			log.Printf("Updated %d action references in %v", totalUpdates, time.Since(start).Round(time.Millisecond))

			if verbose {
				for _, file := range files {
					fmt.Printf("- Processed: %s\n", file)
				}
			}
		},
	}

	updateCmd.Flags().String("dir", ".", "Directory containing GitHub workflows")
	updateCmd.Flags().Int("timeout", 30, "API timeout in seconds")
	updateCmd.Flags().Bool("verbose", false, "Verbose output")

	rootCmd.AddCommand(updateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
