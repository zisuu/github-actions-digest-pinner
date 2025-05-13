package finder

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// FindWorkflowFiles scans the provided filesystem for GitHub Actions workflow files
func FindWorkflowFiles(fsys fs.FS) ([]string, error) {
	var workflowFiles []string

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		dir := filepath.Dir(path)
		if dir == filepath.Join(".github", "workflows") {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yml" || ext == ".yaml" {
				workflowFiles = append(workflowFiles, path)
			}
		}
		return nil
	})

	return workflowFiles, err
}
