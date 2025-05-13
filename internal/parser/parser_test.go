package parser

import (
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
	"testing"
)

func TestParseWorkflowActions(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []types.ActionRef
		wantErr  bool
	}{
		{
			name: "simple action reference",
			content: `
name: CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - uses: actions/setup-java@v4.7
      - uses: actions/setup-node@v4.3.0
`,
			expected: []types.ActionRef{
				{Owner: "actions", Repo: "checkout", Ref: "v3"},
				{Owner: "actions", Repo: "setup-go", Ref: "v4"},
				{Owner: "actions", Repo: "setup-java", Ref: "v4.7"},
				{Owner: "actions", Repo: "setup-node", Ref: "v4.3.0"},
			},
		},
		{
			name: "lint workflow",
			content: `
---
name: Lint

on:  # yamllint disable-line rule:truthy
  pull_request: null

env:
  IGNORE_GITIGNORED_FILES: true
  FILTER_REGEX_EXCLUDE: '.*dev-bundle/.*|.*mvnw.*|.*frontend/.*|.*CHANGELOG.md.*|.*LICENSE.md.*|.*README.md.*'
  VALIDATE_GOOGLE_JAVA_FORMAT: false
  VALIDATE_JAVA: false
  VALIDATE_SQLFLUFF: false
  VALIDATE_JSCPD: false

permissions: {}

jobs:
  build:
    name: Lint
    runs-on: ubuntu-latest

    permissions:
      contents: read
      packages: read
      # To report GitHub Actions status checks
      statuses: write

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          # super-linter needs the full git history to get the
          # list of files that changed across commits
          fetch-depth: 0

      - name: Super-linter
        uses: super-linter/super-linter@v6.7.0  # x-release-please-version
        env:
          # To report GitHub Actions status checks
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
`,
			expected: []types.ActionRef{
				{Owner: "actions", Repo: "checkout", Ref: "v4"},
				{Owner: "super-linter", Repo: "super-linter", Ref: "v6.7.0"},
			},
		},
		{
			name: "with whitespace",
			content: `
steps:
  - uses:  actions/checkout  @v3
`,
			expected: nil,
		},
		{
			name: "invalid action reference",
			content: `
steps:
  - uses: invalid-ref
`,
			expected: nil,
		},
		{
			name: "missing version",
			content: `
steps:
  - uses: actions/checkout@
`,
			expected: nil,
		},
		{
			name: "skip local actions",
			content: `---
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
			expected: []types.ActionRef{
				{Owner: "actions", Repo: "checkout", Ref: "v4"},
				{Owner: "actions", Repo: "setup-java", Ref: "v4.0.1"},
			},
		},
		{
			name: "path-based action reference",
			content: `
jobs:
  test:
    steps:
      - uses: myorg/actions-maven-setup/.github/actions/maven-setup@v1.0.1
`,
			expected: []types.ActionRef{
				{
					Owner: "myorg",
					Repo:  "actions-maven-setup",
					Path:  ".github/actions/maven-setup",
					Ref:   "v1.0.1",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := ParseWorkflowActions([]byte(tt.content))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(actions) != len(tt.expected) {
				t.Fatalf("expected %d actions, got %d", len(tt.expected), len(actions))
			}

			for i, action := range actions {
				if action != tt.expected[i] {
					t.Errorf("action %d mismatch:\nexpected: %+v\ngot:      %+v",
						i, tt.expected[i], action)
				}
			}
		})
	}
}
