package finder

import (
	"testing"
	"testing/fstest"
)

func TestFindWorkflowFiles(t *testing.T) {
	tests := []struct {
		name          string
		fs            fstest.MapFS
		expectedCount int
		expectError   bool
	}{
		{
			name: "finds workflow files in .github/workflows",
			fs: fstest.MapFS{
				".github/workflows/ci.yml":      &fstest.MapFile{},
				".github/workflows/deploy.yaml": &fstest.MapFile{},
				"README.md":                     &fstest.MapFile{},
			},
			expectedCount: 2,
		},
		{
			name: "ignores non-workflow files",
			fs: fstest.MapFS{
				".github/workflows/notes.txt": &fstest.MapFile{},
				"main.go":                     &fstest.MapFile{},
			},
			expectedCount: 0,
		},
		{
			name:          "works with empty directory",
			fs:            fstest.MapFS{},
			expectedCount: 0,
		},
		{
			name: "handles nested workflows (should ignore)",
			fs: fstest.MapFS{
				".github/workflows/nested/extra.yml": &fstest.MapFile{},
			},
			expectedCount: 0,
		},
		{
			name: "case insensitive extensions",
			fs: fstest.MapFS{
				".github/workflows/ci.YML": &fstest.MapFile{},
			},
			expectedCount: 1,
		},
		{
			name: "invalid paths",
			fs: fstest.MapFS{
				"invalid/.github/workflows/ci.yml": &fstest.MapFile{},
				".github/deploy.yaml":              &fstest.MapFile{},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := FindWorkflowFiles(tt.fs)

			if tt.expectError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(files) != tt.expectedCount {
				t.Errorf("expected %d workflow files, got %d", tt.expectedCount, len(files))
			}
		})
	}
}
