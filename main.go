package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zisuu/github-actions-digest-pinner/internal/finder"
	"github.com/zisuu/github-actions-digest-pinner/internal/ghclient"
	"github.com/zisuu/github-actions-digest-pinner/internal/updater"
)

func main() {
	dir := flag.String("dir", ".", "Directory containing GitHub workflows")
	timeout := flag.Int("timeout", 30, "API timeout in seconds")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	if *verbose {
		log.Println("Starting GitHub Actions digest pinner utility")
		log.Printf("Scanning directory: %s", *dir)
	}

	client := ghclient.NewGitHubClient()
	newupdater := updater.NewUpdater(client, ".")
	fsys := os.DirFS(*dir)

	if *verbose {
		log.Println("Finding workflow files...")
	}
	files, err := finder.FindWorkflowFiles(fsys)
	if err != nil {
		log.Fatalf("Failed to find workflow files: %v", err)
	}

	if *verbose {
		log.Printf("Found %d workflow files", len(files))
	}

	totalUpdates, err := newupdater.UpdateWorkflows(ctx, fsys)
	if err != nil {
		log.Fatalf("Failed to update workflows: %v", err)
	}

	log.Printf("Updated %d action references in %v", totalUpdates, time.Since(start).Round(time.Millisecond))

	if *verbose {
		for _, file := range files {
			fmt.Printf("- Processed: %s\n", file)
		}
	}
}
