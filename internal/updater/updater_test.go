package updater_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zisuu/github-actions-digest-pinner/internal/updater"
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
)

type mockGitHubClient struct {
	shaMap map[string]string
}

func (m *mockGitHubClient) ResolveActionSHA(ctx context.Context, action types.ActionRef) (string, error) {
	key := action.Owner + "/" + action.Repo + "@" + action.Ref
	return m.shaMap[key], nil
}

type writableMapFS struct {
	fstest.MapFS
}

func (w *writableMapFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	w.MapFS[name] = &fstest.MapFile{
		Data: data,
		Mode: perm,
	}
	return nil
}

func TestUpdater_UpdateWorkflows(t *testing.T) {
	testCases := []struct {
		name        string
		files       map[string]string
		expected    map[string]string
		shaMap      map[string]string
		wantUpdates int
		wantErr     bool
		errContains string
	}{
		{
			name: "update simple workflow",
			files: map[string]string{
				".github/workflows/test-simple.yml": `name: Test simple
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-java@v4
      - uses: actions/setup-node@v4
`,
			},
			expected: map[string]string{
				".github/workflows/test-simple.yml": `name: Test simple
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675
      - uses: actions/setup-java@b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i
      - uses: actions/setup-node@c9d0e1f2a3b4c5d6e7f8g9h0i1j2k3l4m5n6o7p8
`,
			},
			shaMap: map[string]string{
				"actions/checkout@v4":   "a81bbbf8298c0fa03ea29cdc473d45769f953675",
				"actions/setup-java@v4": "b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i",
				"actions/setup-node@v4": "c9d0e1f2a3b4c5d6e7f8g9h0i1j2k3l4m5n6o7p8",
			},
			wantUpdates: 3,
		},
		{
			name: "update verbose workflow",
			files: map[string]string{
				".github/workflows/test-verbose.yml": `---
name: Build

on:
  pull_request: null

env:
  GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

permissions: {}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: write # to be able to publish a GitHub release
      issues: write # to be able to comment on released issues
      pull-requests: write # to be able to comment on released pull requests
      id-token: write # to enable use of OIDC for npm provenance
    steps:
      - uses: actions/checkout@v4
      - name: Set up JDK 21
        uses: actions/setup-java@v4.0.1
        with:
          java-version: '21'
          distribution: 'adopt'
          cache: maven
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: 'lts/*'
      - name: Build with Maven
        run: mvn -B package -Pproduction --file pom.xml
`,
			},
			expected: map[string]string{
				".github/workflows/test-verbose.yml": `---
name: Build

on:
  pull_request: null

env:
  GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

permissions: {}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: write # to be able to publish a GitHub release
      issues: write # to be able to comment on released issues
      pull-requests: write # to be able to comment on released pull requests
      id-token: write # to enable use of OIDC for npm provenance
    steps:
      - uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675
      - name: Set up JDK 21
        uses: actions/setup-java@b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i
        with:
          java-version: '21'
          distribution: 'adopt'
          cache: maven
      - name: Setup Node.js
        uses: actions/setup-node@c9d0e1f2a3b4c5d6e7f8g9h0i1j2k3l4m5n6o7p8
        with:
          node-version: 'lts/*'
      - name: Build with Maven
        run: mvn -B package -Pproduction --file pom.xml
`,
			},
			shaMap: map[string]string{
				"actions/checkout@v4":       "a81bbbf8298c0fa03ea29cdc473d45769f953675",
				"actions/setup-java@v4.0.1": "b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i",
				"actions/setup-node@v4":     "c9d0e1f2a3b4c5d6e7f8g9h0i1j2k3l4m5n6o7p8",
			},
			wantUpdates: 3,
		},
		{
			name: "skip already pinned actions",
			files: map[string]string{
				".github/workflows/test-already-pinned.yml": `---
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675
      - uses: actions/setup-java@v4
`,
			},
			expected: map[string]string{
				".github/workflows/test-already-pinned.yml": `---
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675
      - uses: actions/setup-java@b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i
`,
			},
			shaMap: map[string]string{
				"actions/setup-java@v4": "b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i",
			},
			wantUpdates: 1,
		},
		{
			name: "skip local action but update others",
			files: map[string]string{
				".github/workflows/test-local-action.yml": `---
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Check if version was updated
        id: version-tag
        uses: ./.github/actions/setup-versions
        with:
          branch_to_compare: ${{ github.base_ref }}
          file_path: ${{ env.file_path }}/cpu/VERSION
          deployment-tag: ${{ github.event.pull_request.merged }}
      - uses: ../parent/local/action
      - uses: docker://alpine:3.14
      - uses: actions/setup-java@v4.0.1
`,
			},
			expected: map[string]string{
				".github/workflows/test-local-action.yml": `---
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675
      - name: Check if version was updated
        id: version-tag
        uses: ./.github/actions/setup-versions
        with:
          branch_to_compare: ${{ github.base_ref }}
          file_path: ${{ env.file_path }}/cpu/VERSION
          deployment-tag: ${{ github.event.pull_request.merged }}
      - uses: ../parent/local/action
      - uses: docker://alpine:3.14
      - uses: actions/setup-java@b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i
`,
			},
			shaMap: map[string]string{
				"actions/checkout@v4":       "a81bbbf8298c0fa03ea29cdc473d45769f953675",
				"actions/setup-java@v4.0.1": "b72c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8g9h0i",
			},
			wantUpdates: 2,
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup writable filesystem
			memFS := &writableMapFS{MapFS: make(fstest.MapFS)}
			for name, content := range tc.files {
				memFS.MapFS[name] = &fstest.MapFile{Data: []byte(content)}
			}

			// Run updater
			client := &mockGitHubClient{shaMap: tc.shaMap}
			u := updater.NewUpdater(client)
			updates, err := u.UpdateWorkflows(context.Background(), memFS)

			// Error handling
			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tc.errContains, err.Error())
				}
				return // Skip further checks for error cases
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify updates count
			if updates != tc.wantUpdates {
				t.Errorf("Expected %d updates, got %d", tc.wantUpdates, updates)
			}

			// Verify file contents (only if no error expected)
			for name, expectedContent := range tc.expected {
				content, err := fs.ReadFile(memFS, name)
				if err != nil {
					t.Fatalf("Failed to read file %s: %v", name, err)
				}
				if string(content) != expectedContent {
					t.Errorf("Content mismatch in %s:\nExpected:\n%s\n\nGot:\n%s",
						name, expectedContent, string(content))
				}
			}
		})
	}
}
