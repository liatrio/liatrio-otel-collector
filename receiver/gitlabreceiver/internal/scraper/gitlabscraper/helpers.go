package gitlabscraper

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/zap"

	"github.com/Khan/genqlient/graphql"
)

type gitlabProject struct {
	Name           string
	Path           string
	CreatedAt      time.Time
	LastActivityAt time.Time
}

func (gls *gitlabScraper) getProjects(restClient *gitlab.Client) ([]gitlabProject, error) {
	var projectList []gitlabProject

	for nextPage := 1; nextPage > 0; {
		// TODO: since we pass in a context already, do we need to create a new background context?
		projects, res, err := restClient.Groups.ListGroupProjects(gls.cfg.GitLabOrg, &gitlab.ListGroupProjectsOptions{
			IncludeSubGroups: gitlab.Ptr(true),
			Topic:            gitlab.Ptr(gls.cfg.SearchTopic),
			Search:           gitlab.Ptr(gls.cfg.SearchQuery),
			Archived:         gitlab.Ptr(false),
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 100,
			},
		})
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
			return nil, err
		}

		if len(projects) == 0 {
			errMsg := fmt.Sprintf("no GitLab projects found for the given group/org: %s", gls.cfg.GitLabOrg)
			err = errors.New(errMsg)
			gls.logger.Sugar().Error(err)
			return nil, err
		}

		for _, p := range projects {
			projectList = append(projectList, gitlabProject{
				Name:           p.Name,
				Path:           p.PathWithNamespace,
				CreatedAt:      *p.CreatedAt,
				LastActivityAt: *p.LastActivityAt,
			})
		}

		nextPageHeader := res.Header.Get("x-next-page")
		if len(nextPageHeader) > 0 {
			nextPage, err = strconv.Atoi(nextPageHeader)
			if err != nil {
				gls.logger.Sugar().Errorf("error: %v", err)

				return nil, err
			}
		} else {
			nextPage = 0
		}
	}

	return projectList, nil
}

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

func (gls *gitlabScraper) getContributorCount(
	restClient *gitlab.Client,
	projectPath string,
) (int, error) {
	contributors, _, err := restClient.Repositories.Contributors(projectPath, nil)
	if err != nil {
		gls.logger.Sugar().Errorf("error getting contributors: %v", zap.Error(err))
		return 0, err
	}

	return len(contributors), nil
}

func (gls *gitlabScraper) getMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
	state MergeRequestState,
) ([]MergeRequestNode, error) {
	var mergeRequestData []MergeRequestNode
	var mrCursor *string

	for hasNextPage := true; hasNextPage; {
		// Get the next page of data
		mr, err := getMergeRequests(ctx, graphClient, projectPath, mrCursor, state)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
			return nil, err
		}
		if len(mr.Project.MergeRequests.Nodes) == 0 {
			break
		}

		mrCursor = &mr.Project.MergeRequests.PageInfo.EndCursor
		hasNextPage = mr.Project.MergeRequests.PageInfo.HasNextPage
		mergeRequestData = append(mergeRequestData, mr.Project.MergeRequests.Nodes...)
	}

	return mergeRequestData, nil
}

func (gls *gitlabScraper) getCombinedMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
) ([]MergeRequestNode, error) {
	openMrs, err := gls.getMergeRequests(ctx, graphClient, projectPath, MergeRequestStateOpened)
	if err != nil {
		gls.logger.Sugar().Errorf("error getting open merge requests: %v", zap.Error(err))
		return nil, err
	}
	mergedMrs, err := gls.getMergeRequests(ctx, graphClient, projectPath, MergeRequestStateMerged)
	if err != nil {
		gls.logger.Sugar().Errorf("error getting merged merge requests: %v", zap.Error(err))
		return nil, err
	}
	mrs := append(openMrs, mergedMrs...)
	return mrs, nil
}
