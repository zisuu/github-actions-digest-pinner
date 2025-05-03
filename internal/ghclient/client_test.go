package ghclient

import (
	"context"
	"fmt"
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
	"testing"
)

type mockGitHubClient struct {
	sha string
	err error
}

func (m *mockGitHubClient) ResolveActionSHA(ctx context.Context, action types.ActionRef) (string, error) {
	return m.sha, m.err
}

func TestResolveActionSHA(t *testing.T) {
	tests := []struct {
		name    string
		client  GitHubClient
		action  types.ActionRef
		wantSHA string
		wantErr bool
	}{
		{
			name: "successful resolution",
			client: &mockGitHubClient{
				sha: "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			},
			action: types.ActionRef{
				Owner: "actions",
				Repo:  "checkout",
				Ref:   "v3",
			},
			wantSHA: "a81bbbf8298c0fa03ea29cdc473d45769f953675",
		},
		{
			name: "not found",
			client: &mockGitHubClient{
				err: fmt.Errorf("not found"),
			},
			action: types.ActionRef{
				Owner: "nonexistent",
				Repo:  "nonexistent",
				Ref:   "v999",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sha, err := tt.client.ResolveActionSHA(context.Background(), tt.action)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if sha != tt.wantSHA {
				t.Errorf("expected SHA %q, got %q", tt.wantSHA, sha)
			}
		})
	}
}
