package types

// Package types provides common types and interfaces for the GitHub Actions Digest Pinner application.
type ActionRef struct {
	Owner string
	Repo  string
	Path  string
	Ref   string
}
