package updater

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zisuu/github-actions-digest-pinner/internal/finder"
	"github.com/zisuu/github-actions-digest-pinner/internal/ghclient"
	"github.com/zisuu/github-actions-digest-pinner/internal/parser"
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
)

var shaRegex = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// Updater is responsible for updating GitHub Actions workflow files
type Updater struct {
	Client  ghclient.GitHubClient
	baseDir string
}

// NewUpdater creates a new Updater instance with the provided GitHub client
func NewUpdater(client ghclient.GitHubClient) *Updater {
	return &Updater{
		Client: client,
	}
}

// SetBaseDir sets the base directory for file operations
func (u *Updater) SetBaseDir(dir string) {
	u.baseDir = dir
}

// UpdateWorkflows scans for workflow files, parses them, and updates action references
func (u *Updater) UpdateWorkflows(ctx context.Context, fsys fs.FS) (int, error) {
	files, err := finder.FindWorkflowFiles(fsys)
	if err != nil {
		return 0, fmt.Errorf("failed to find workflow files: %w", err)
	}

	totalUpdates := 0
	for _, file := range files {
		updates, err := u.processWorkflowFile(ctx, fsys, file)
		if err != nil {
			return totalUpdates, err
		}
		totalUpdates += updates
	}
	return totalUpdates, nil
}

// processWorkflowFile reads a workflow file, parses it for action references
func (u *Updater) processWorkflowFile(ctx context.Context, fsys fs.FS, file string) (int, error) {
	log.Printf("Processing file: %s", file)

	content, err := fs.ReadFile(fsys, file)
	if err != nil {
		return 0, fmt.Errorf("failed to read file %s: %w", file, err)
	}

	actions, err := parser.ParseWorkflowActions(content)
	if err != nil {
		return 0, fmt.Errorf("failed to parse actions in file %s: %w", file, err)
	}

	debugActions(actions)
	log.Printf("Found %d actions in file %s", len(actions), file)

	updatedContent, fileUpdates, err := u.updateActionReferences(ctx, string(content), actions)
	if err != nil {
		return 0, err
	}

	if fileUpdates > 0 {
		return fileUpdates, u.writeUpdatedFile(fsys, file, updatedContent)
	}

	log.Printf("No changes made to file: %s", file)
	return 0, nil
}

// updateActionReferences updates action references in the content
func (u *Updater) updateActionReferences(ctx context.Context, content string, actions []types.ActionRef) (string, int, error) {
	updatedContent := content
	fileUpdates := 0

	for _, action := range actions {
		newContent, updated, err := u.updateSingleActionReference(ctx, updatedContent, action)
		if err != nil {
			return "", 0, err
		}
		if updated {
			updatedContent = newContent
			fileUpdates++
		}
	}

	return updatedContent, fileUpdates, nil
}

// updateSingleActionReference updates a single action reference in the content
func (u *Updater) updateSingleActionReference(ctx context.Context, content string, action types.ActionRef) (string, bool, error) {
	if isSHA(action.Ref) {
		log.Printf("Skipping %s/%s@%s (already a SHA)", action.Owner, action.Repo, action.Ref)
		return content, false, nil
	}

	log.Printf("Processing action: %s/%s@%s", action.Owner, action.Repo, action.Ref)
	sha, err := u.Client.ResolveActionSHA(ctx, action)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve SHA for action %s/%s@%s: %w",
			action.Owner, action.Repo, action.Ref, err)
	}
	log.Printf("Resolved SHA for %s/%s@%s: %s", action.Owner, action.Repo, action.Ref, sha)

	// Build the exact reference string that appears in the workflow file
	var oldRef string
	if action.Path != "" {
		oldRef = fmt.Sprintf("%s/%s/%s@%s", action.Owner, action.Repo, action.Path, action.Ref)
	} else {
		oldRef = fmt.Sprintf("%s/%s@%s", action.Owner, action.Repo, action.Ref)
	}

	newRef := strings.Replace(oldRef, action.Ref, sha, 1)

	if !strings.Contains(content, oldRef) {
		log.Printf("Warning: reference %s not found in content", oldRef)
		return content, false, nil
	}

	updated := strings.Replace(content, oldRef, newRef, 1)
	if updated == content {
		log.Printf("Warning: no changes made for %s (reference not found or already updated)", oldRef)
		return content, false, nil
	}

	return updated, true, nil
}

// writeUpdatedFile writes the updated content back to the file system
func (u *Updater) writeUpdatedFile(fsys fs.FS, file string, content string) error {
	// First try if the filesystem supports writing (for tests)
	if writeFS, ok := fsys.(interface {
		WriteFile(name string, data []byte, perm fs.FileMode) error
	}); ok {
		return writeFS.WriteFile(file, []byte(content), 0644)
	}

	// For real filesystem operations, use the base directory
	if u.baseDir == "" {
		return fmt.Errorf("base directory not set for real filesystem operations")
	}
	fullPath := filepath.Join(u.baseDir, file)
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// isSHA checks if a string is a valid Git SHA-1 hash (40 hex characters)
func isSHA(ref string) bool {
	return shaRegex.MatchString(ref)
}

// debugActions logs the action references for debugging purposes
func debugActions(actions []types.ActionRef) {
	for i, action := range actions {
		log.Printf("Action %d: %s/%s@%s", i+1, action.Owner, action.Repo, action.Ref)
	}
}
