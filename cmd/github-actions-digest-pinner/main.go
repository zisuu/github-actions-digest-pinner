package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zisuu/github-actions-digest-pinner/internal/finder"
	"github.com/zisuu/github-actions-digest-pinner/internal/ghclient"
	"github.com/zisuu/github-actions-digest-pinner/internal/parser"
	"github.com/zisuu/github-actions-digest-pinner/internal/updater"
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
)

var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

type WorkflowFinder interface {
	FindWorkflowFiles(fsys fs.FS) ([]string, error)
}

type WorkflowParser interface {
	ParseWorkflowActions(content []byte) ([]types.ActionRef, error)
}

type WorkflowUpdater interface {
	UpdateWorkflows(ctx context.Context, fsys fs.FS) (int, error)
}

type App struct {
	Out      io.Writer
	Err      io.Writer
	Client   ghclient.GitHubClient
	Finder   WorkflowFinder
	Parser   WorkflowParser
	Updater  WorkflowUpdater
	OpenFile func(name string) (fs.File, error)
	ReadFile func(fsys fs.FS, name string) ([]byte, error)
}

func NewApp(out, err io.Writer) *App {
	return &App{
		Out:      out,
		Err:      err,
		Client:   ghclient.NewGitHubClient(),
		Finder:   finder.DefaultFinder{},
		Parser:   parser.DefaultParser{},
		Updater:  updater.NewUpdater(ghclient.NewGitHubClient(), "."),
		OpenFile: func(name string) (fs.File, error) { return os.Open(name) },
		ReadFile: fs.ReadFile,
	}
}

func (a *App) scanCommand(dir string, verbose bool) error {
	if verbose {
		log.SetOutput(a.Err)
		log.Println("Starting GitHub Actions digest pinner utility")
		log.Printf("Scanning directory: %s", dir)
	}

	fsys := os.DirFS(dir)

	if verbose {
		log.Println("Finding workflow files...")
	}

	files, err := a.Finder.FindWorkflowFiles(fsys)
	if err != nil {
		return fmt.Errorf("failed to find workflow files: %w", err)
	}

	if verbose {
		log.Printf("Found %d workflow files", len(files))
		log.Println("Parsing actions in workflow files...")
	}

	for _, file := range files {
		if verbose {
			log.Printf("Processing file: %s", file)
		}

		// Open file just to check if it exists
		if _, err := a.OpenFile(file); err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		fileContent, err := a.ReadFile(fsys, file)
		if err != nil {
			return fmt.Errorf("failed to read content of file %s: %w", file, err)
		}

		actions, err := a.Parser.ParseWorkflowActions(fileContent)
		if err != nil {
			return fmt.Errorf("failed to parse actions in file %s: %w", file, err)
		}

		if verbose {
			log.Printf("Found %d actions in file %s", len(actions), file)
			for _, action := range actions {
				fmt.Fprintf(a.Out, "- Action: %s/%s@%s\n", action.Owner, action.Repo, action.Ref)
			}
		} else if len(actions) > 0 {
			fmt.Fprintf(a.Out, "%s: %d actions found\n", file, len(actions))
		}
	}

	return nil
}

func (a *App) updateCommand(dir string, timeout int, verbose bool) error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if verbose {
		log.SetOutput(a.Err)
		log.Println("Starting GitHub Actions digest pinner utility")
		log.Printf("Scanning directory: %s", dir)
	}

	fsys := os.DirFS(dir)

	if verbose {
		log.Println("Finding workflow files...")
	}

	files, err := a.Finder.FindWorkflowFiles(fsys)
	if err != nil {
		return fmt.Errorf("failed to find workflow files: %w", err)
	}

	if verbose {
		log.Printf("Found %d workflow files", len(files))
	}

	totalUpdates, err := a.Updater.UpdateWorkflows(ctx, fsys)
	if err != nil {
		return fmt.Errorf("failed to update workflows: %w", err)
	}

	if verbose {
		log.Printf("Updated %d action references in %v", totalUpdates, time.Since(start).Round(time.Millisecond))
		for _, file := range files {
			fmt.Fprintf(a.Out, "- Processed: %s\n", file)
		}
	} else {
		fmt.Fprintf(a.Out, "Updated %d action references in %v\n", totalUpdates, time.Since(start).Round(time.Millisecond))
	}

	return nil
}

func (a *App) versionCommand() {
	fmt.Fprintf(a.Out, "Version: %s\nCommit: %s\nDate: %s\n", version, commit, date)
}

func newRootCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github-actions-digest-pinner",
		Short: "A tool to pin GitHub Actions to specific digests",
		Long:  "GitHub Actions Digest Pinner is a tool to help you pin GitHub Actions to specific digests for better security and reliability.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				log.Printf("Failed to display help: %v", err)
				os.Exit(1)
			}
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show the version information",
		Run: func(cmd *cobra.Command, args []string) {
			app.versionCommand()
		},
	})

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the repository for GitHub Actions workflows",
		Run: func(cmd *cobra.Command, args []string) {
			dir, _ := cmd.Flags().GetString("dir")
			verbose, _ := cmd.Flags().GetBool("verbose")
			if err := app.scanCommand(dir, verbose); err != nil {
				log.Printf("Scan failed: %v", err)
				os.Exit(1)
			}
		},
	}

	scanCmd.Flags().String("dir", ".", "Directory containing GitHub workflows")
	scanCmd.Flags().Bool("verbose", false, "Verbose output")
	cmd.AddCommand(scanCmd)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update GitHub Actions workflows to use pinned digests",
		Run: func(cmd *cobra.Command, args []string) {
			dir, _ := cmd.Flags().GetString("dir")
			timeout, _ := cmd.Flags().GetInt("timeout")
			verbose, _ := cmd.Flags().GetBool("verbose")
			if err := app.updateCommand(dir, timeout, verbose); err != nil {
				log.Printf("Update failed: %v", err)
				os.Exit(1)
			}
		},
	}

	updateCmd.Flags().String("dir", ".", "Directory containing GitHub workflows")
	updateCmd.Flags().Int("timeout", 30, "API timeout in seconds")
	updateCmd.Flags().Bool("verbose", false, "Verbose output")
	cmd.AddCommand(updateCmd)

	return cmd
}

func main() {
	app := NewApp(os.Stdout, os.Stderr)
	rootCmd := newRootCommand(app)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(app.Err, err)
		os.Exit(1)
	}
}
