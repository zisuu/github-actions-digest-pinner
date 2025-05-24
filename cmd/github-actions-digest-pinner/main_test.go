package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
)

type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) ResolveActionSHA(ctx context.Context, action types.ActionRef) (string, error) {
	args := m.Called(ctx, action)
	return args.String(0), args.Error(1)
}

type MockFinder struct {
	mock.Mock
}

func (m *MockFinder) FindWorkflowFiles(fsys fs.FS) ([]string, error) {
	args := m.Called(fsys)
	return args.Get(0).([]string), args.Error(1)
}

type MockParser struct {
	mock.Mock
}

func (m *MockParser) ParseWorkflowActions(content []byte) ([]types.ActionRef, error) {
	args := m.Called(content)
	return args.Get(0).([]types.ActionRef), args.Error(1)
}

type MockUpdater struct {
	mock.Mock
}

func (m *MockUpdater) UpdateWorkflows(ctx context.Context, fsys fs.FS) (int, error) {
	args := m.Called(ctx, fsys)
	return args.Int(0), args.Error(1)
}

type MockFile struct {
	content []byte
	offset  int64
}

func (m *MockFile) Read(p []byte) (n int, err error) {
	if m.offset >= int64(len(m.content)) {
		return 0, io.EOF
	}
	n = copy(p, m.content[m.offset:])
	m.offset += int64(n)
	return n, nil
}

func (m *MockFile) Stat() (fs.FileInfo, error) {
	return &MockFileInfo{name: "mockfile", size: int64(len(m.content))}, nil
}

func (m *MockFile) Close() error {
	return nil
}

type MockFileInfo struct {
	name string
	size int64
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() fs.FileMode  { return 0644 }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return false }
func (m *MockFileInfo) Sys() interface{}   { return nil }

type MockFS struct {
	files map[string]*MockFile
}

func (m *MockFS) Open(name string) (fs.File, error) {
	if file, ok := m.files[name]; ok {
		file.offset = 0
		return file, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func TestVersionCommand(t *testing.T) {
	var buf bytes.Buffer
	app := &App{Out: &buf}
	app.versionCommand()

	output := buf.String()
	assert.Contains(t, output, "Version: unknown")
	assert.Contains(t, output, "Commit: unknown")
	assert.Contains(t, output, "Date: unknown")
}

func TestScanCommand(t *testing.T) {
	tests := []struct {
		name          string
		verbose       bool
		mockFiles     []string
		mockActions   []types.ActionRef
		mockError     error
		expectError   bool
		expectOutput  string
		expectVerbose string
	}{
		{
			name:         "successful scan",
			verbose:      false,
			mockFiles:    []string{"test.yml"},
			mockActions:  []types.ActionRef{{Owner: "owner", Repo: "repo", Ref: "v1"}},
			expectOutput: "test.yml: 1 actions found\n",
		},
		{
			name:          "verbose scan",
			verbose:       true,
			mockFiles:     []string{"test.yml"},
			mockActions:   []types.ActionRef{{Owner: "owner", Repo: "repo", Ref: "v1"}},
			expectVerbose: "- Action: owner/repo@v1\n",
		},
		{
			name:        "file read error",
			mockFiles:   []string{"error.yml"},
			mockError:   errors.New("read error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer

			mockFinder := new(MockFinder)
			mockParser := new(MockParser)
			mockFS := &MockFS{files: make(map[string]*MockFile)}

			// Create a mock filesystem with the test files
			for _, file := range tt.mockFiles {
				mockFS.files[file] = &MockFile{content: []byte("dummy content")}
			}

			app := &App{
				Out:    &outBuf,
				Err:    &errBuf,
				Finder: mockFinder,
				Parser: mockParser,
				FS: func(dir string) fs.FS {
					return mockFS
				},
				ReadFile: func(fsys fs.FS, name string) ([]byte, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					file, err := fsys.Open(name)
					if err != nil {
						return nil, err
					}
					return io.ReadAll(file)
				},
			}

			mockFinder.On("FindWorkflowFiles", mock.Anything).Return(tt.mockFiles, nil).Once()

			if tt.mockError == nil {
				for range tt.mockFiles {
					mockParser.On("ParseWorkflowActions", []byte("dummy content")).Return(tt.mockActions, nil).Once()
				}
			}

			err := app.scanCommand(".", tt.verbose)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.verbose {
					assert.Contains(t, errBuf.String(), "Starting GitHub Actions digest pinner utility")
					assert.Contains(t, outBuf.String(), tt.expectVerbose)
				} else {
					assert.Equal(t, tt.expectOutput, outBuf.String())
				}
			}

			mockFinder.AssertExpectations(t)
			if tt.mockError == nil {
				mockParser.AssertExpectations(t)
			}
		})
	}
}

func TestUpdateCommand(t *testing.T) {
	tests := []struct {
		name          string
		timeout       int
		verbose       bool
		mockFiles     []string
		mockUpdates   int
		mockError     error
		expectError   bool
		expectOutput  string
		expectVerbose string
	}{
		{
			name:         "successful update",
			timeout:      30,
			verbose:      false,
			mockFiles:    []string{"test.yml"},
			mockUpdates:  2,
			expectOutput: "Updated 2 action references in",
		},
		{
			name:          "verbose update",
			timeout:       30,
			verbose:       true,
			mockFiles:     []string{"test.yml"},
			mockUpdates:   1,
			expectVerbose: "- Processed: test.yml\n",
		},
		{
			name:        "update error",
			mockFiles:   []string{"test.yml"},
			mockError:   errors.New("update error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer

			mockFinder := new(MockFinder)
			mockUpdater := new(MockUpdater)
			mockFS := &MockFS{files: make(map[string]*MockFile)}

			// Create mock files if needed
			for _, file := range tt.mockFiles {
				mockFS.files[file] = &MockFile{content: []byte("dummy content")}
			}

			app := &App{
				Out:     &outBuf,
				Err:     &errBuf,
				Finder:  mockFinder,
				Updater: mockUpdater,
				FS: func(dir string) fs.FS {
					return mockFS
				},
				ReadFile: func(fsys fs.FS, name string) ([]byte, error) {
					file, err := fsys.Open(name)
					if err != nil {
						return nil, err
					}
					return io.ReadAll(file)
				},
			}

			mockFinder.On("FindWorkflowFiles", mock.Anything).Return(tt.mockFiles, nil).Once()
			mockUpdater.On("UpdateWorkflows", mock.Anything, mock.Anything).Return(tt.mockUpdates, tt.mockError).Once()

			err := app.updateCommand(".", tt.timeout, tt.verbose)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.verbose {
					assert.Contains(t, errBuf.String(), "Starting GitHub Actions digest pinner utility")
					assert.Contains(t, outBuf.String(), tt.expectVerbose)
				} else {
					assert.Contains(t, outBuf.String(), tt.expectOutput)
				}
			}

			mockFinder.AssertExpectations(t)
			mockUpdater.AssertExpectations(t)
		})
	}
}

func TestRootCommand(t *testing.T) {
	app := &App{
		Out:     os.Stdout,
		Err:     os.Stderr,
		Finder:  new(MockFinder),
		Parser:  new(MockParser),
		Updater: &MockUpdater{},
		Client:  new(MockGitHubClient),
	}

	cmd := newRootCommand(app)

	assert.Equal(t, "github-actions-digest-pinner", cmd.Use)
	assert.Len(t, cmd.Commands(), 3)

	var scanCmd, updateCmd *cobra.Command
	for _, c := range cmd.Commands() {
		switch c.Use {
		case "scan":
			scanCmd = c
		case "update":
			updateCmd = c
		}
	}

	assert.NotNil(t, scanCmd)
	dirFlag := scanCmd.Flags().Lookup("dir")
	assert.NotNil(t, dirFlag)
	assert.Equal(t, ".", dirFlag.DefValue)

	verboseFlag := scanCmd.Flags().Lookup("verbose")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "false", verboseFlag.DefValue)

	assert.NotNil(t, updateCmd)
	timeoutFlag := updateCmd.Flags().Lookup("timeout")
	assert.NotNil(t, timeoutFlag)
	assert.Equal(t, "30", timeoutFlag.DefValue)
}
