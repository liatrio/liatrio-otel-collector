package gitlabscraper

import (
	"context"

	"github.com/xanzy/go-gitlab"

	"github.com/Khan/genqlient/graphql"
)

func (gls *gitlabScraper) getBranchNames(ctx context.Context, client graphql.Client, projectPath string) (*getBranchNamesProjectRepository, error) {
	branches, err := getBranchNames(ctx, client, projectPath)
	if err != nil {
		return nil, err
	}
	return &branches.Project.Repository, nil
}

func (gls *gitlabScraper) getInitialCommit(client *gitlab.Client, projectPath string, defaultBranch string, branch string) (*gitlab.Commit, error) {
	diff, _, err := client.Repositories.Compare(projectPath, &gitlab.CompareOptions{From: &defaultBranch, To: &branch})
	if err != nil {
		return nil, err
	}
	if len(diff.Commits) == 0 {
		return nil, nil
	}
	return diff.Commits[0], nil
}
