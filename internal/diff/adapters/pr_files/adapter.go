package prfiles

import (
	"context"
	"fmt"

	"github.com/google/go-github/v68/github"
)

// Adapter implements ports.FileChangesPort by querying the GitHub API
// for files changed in a pull request.
type Adapter struct {
	client *github.Client
}

// New creates a new PR files adapter.
func New(client *github.Client) *Adapter {
	return &Adapter{client: client}
}

// GetChangedFiles returns a list of files modified in the PR.
func (a *Adapter) GetChangedFiles(ctx context.Context, owner, repo string, prNumber int) ([]string, error) {
	client := a.client

	var changedFiles []string
	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		files, resp, err := client.PullRequests.ListFiles(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("listing PR files: %w", err)
		}

		for _, file := range files {
			changedFiles = append(changedFiles, file.GetFilename())
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return changedFiles, nil
}
