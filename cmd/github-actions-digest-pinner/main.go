package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/zisuu/github-actions-digest-pinner/internal/finder"
	"github.com/zisuu/github-actions-digest-pinner/internal/ghclient"
	"github.com/zisuu/github-actions-digest-pinner/internal/parser"
	"github.com/zisuu/github-actions-digest-pinner/internal/updater"
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
)

// Build number and versions injected at compile time
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

// App represents the main application structure.
type App struct {
	Out      io.Writer
	Err      io.Writer
	Client   ghclient.GitHubClient
	Finder   WorkflowFinder
	Parser   WorkflowParser
	Updater  WorkflowUpdater
	FS       func(dir string) fs.FS
	ReadFile func(fsys fs.FS, name string) ([]byte, error)
}

// NewApp creates a new instance of App with the provided output and error writers.
func NewApp(out, err io.Writer) *App {
	return &App{
		Out:     out,
		Err:     err,
		Client:  ghclient.NewGitHubClient(),
		Finder:  finder.DefaultFinder{},
		Parser:  parser.DefaultParser{},
		Updater: updater.NewUpdater(ghclient.NewGitHubClient()),
		FS: func(dir string) fs.FS {
			return os.DirFS(dir)
		},
		ReadFile: fs.ReadFile,
	}
}

// scanCommand scans the specified directory for GitHub Actions workflows and prints the actions found.
func (a *App) scanCommand(dir string, verbose bool) error {
	if verbose {
		log.SetOutput(a.Err)
		log.Println("Starting GitHub Actions digest pinner utility")
		log.Printf("Scanning directory: %s", dir)
	}

	fsys := a.FS(dir)

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
				_, err := fmt.Fprintf(a.Out, "- Action: %s/%s@%s\n", action.Owner, action.Repo, action.Ref)
				if err != nil {
					return fmt.Errorf("failed to write action output: %w", err)
				}
			}
		} else if len(actions) > 0 {
			_, err := fmt.Fprintf(a.Out, "%s: %d actions found\n", file, len(actions))
			if err != nil {
				return fmt.Errorf("failed to write actions found output: %w", err)
			}
		}
	}

	return nil
}

// updateCommand updates the GitHub Actions workflows in the specified directory to use pinned digests.
func (a *App) updateCommand(dir string, timeout int, verbose bool) error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if verbose {
		log.SetOutput(a.Err)
		log.Println("Starting GitHub Actions digest pinner utility")
		log.Printf("Scanning directory: %s", dir)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fsys := a.FS(absDir)

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

	if upd, ok := a.Updater.(*updater.Updater); ok {
		upd.SetBaseDir(absDir)
	}

	totalUpdates, err := a.Updater.UpdateWorkflows(ctx, fsys)
	if err != nil {
		return fmt.Errorf("failed to update workflows: %w", err)
	}

	if verbose {
		log.Printf("Updated %d action references in %v", totalUpdates, time.Since(start).Round(time.Millisecond))
		for _, file := range files {
			_, err := fmt.Fprintf(a.Out, "- Processed: %s\n", file)
			if err != nil {
				return fmt.Errorf("failed to write processed file output: %w", err)
			}
		}
	} else {
		_, err := fmt.Fprintf(a.Out, "Updated %d action references in %v\n", totalUpdates, time.Since(start).Round(time.Millisecond))
		if err != nil {
			return fmt.Errorf("failed to write update summary output: %w", err)
		}
	}

	return nil
}

// versionCommand prints the version information of the application.
func (a *App) versionCommand() {
	_, err := fmt.Fprintf(a.Out, "Version: %s\nCommit: %s\nDate: %s\n", version, commit, date)
	if err != nil {
		log.Printf("Failed to write version output: %v", err)
	}
}

// newRootCommand creates the root command for the CLI application.
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

// main is the entry point of the application.
func main() {
	app := NewApp(os.Stdout, os.Stderr)
	rootCmd := newRootCommand(app)

	if err := rootCmd.Execute(); err != nil {
		_, fmtErr := fmt.Fprintln(app.Err, err)
		if fmtErr != nil {
			log.Printf("Failed to write error output: %v", fmtErr)
		}
		os.Exit(1)
	}
}
