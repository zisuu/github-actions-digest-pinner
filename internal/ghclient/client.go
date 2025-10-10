package ghclient

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v75/github"
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
	"golang.org/x/oauth2"
)

type GitHubClient interface {
	ResolveActionSHA(ctx context.Context, action types.ActionRef) (string, error)
}

// githubClient is a wrapper around the GitHub client.
type githubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub client.
func NewGitHubClient() GitHubClient {
	token := os.Getenv("GITHUB_TOKEN")

	var client *github.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(context.Background(), ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	return &githubClient{client: client}
}

// ResolveActionSHA resolves the SHA of a GitHub Action reference.
func (g *githubClient) ResolveActionSHA(ctx context.Context, action types.ActionRef) (string, error) {
	if isSHA(action.Ref) {
		return action.Ref, nil
	}

	ref, _, err := g.client.Git.GetRef(ctx, action.Owner, action.Repo, "tags/"+action.Ref)
	if err != nil {
		ref, _, err = g.client.Git.GetRef(ctx, action.Owner, action.Repo, "heads/"+action.Ref)
		if err != nil {
			return "", fmt.Errorf("failed to resolve ref %s: %w", action.Ref, err)
		}
	}

	return ref.GetObject().GetSHA(), nil
}

func isSHA(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, c := range ref {
		if c < '0' || (c > '9' && c < 'A') || (c > 'F' && c < 'a') || c > 'f' {
			return false
		}
	}
	return true
}
