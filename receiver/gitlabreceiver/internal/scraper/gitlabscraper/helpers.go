package gitlabscraper

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/Khan/genqlient/graphql"
	"github.com/cenkalti/backoff/v5"
)

type gitlabProject struct {
	Name           string
	Path           string
	CreatedAt      time.Time
	LastActivityAt time.Time
	URL            string
}

func (gls *gitlabScraper) getProjects(ctx context.Context, restClient *gitlab.Client) ([]gitlabProject, error) {
	var projectList []gitlabProject

	operation := func() (string, error) {
		for nextPage := 1; nextPage > 0; {
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
				if apiErr, ok := err.(*gitlab.ErrorResponse); ok && apiErr.Response.StatusCode == 429 &&
					apiErr.Response.Status == "429 Too Many Requests" {
					return "", backoff.RetryAfter(60)
				}
				return "", backoff.Permanent(err)
			}

			if len(projects) == 0 {
				errMsg := fmt.Sprintf("no GitLab projects found for the given group/org: %s", gls.cfg.GitLabOrg)
				err = errors.New(errMsg)
				gls.logger.Sugar().Error(err)
				return "", backoff.Permanent(err)
			}

			for _, p := range projects {
				projectList = append(projectList, gitlabProject{
					Name:           p.Name,
					Path:           p.PathWithNamespace,
					CreatedAt:      *p.CreatedAt,
					LastActivityAt: *p.LastActivityAt,
					URL:            p.WebURL,
				})
			}

			nextPageHeader := res.Header.Get("x-next-page")
			if len(nextPageHeader) > 0 {
				nextPage, err = strconv.Atoi(nextPageHeader)
				if err != nil {
					gls.logger.Sugar().Errorf("error: %v", err)

					return "", backoff.Permanent(err)
				}
			} else {
				nextPage = 0
			}
		}
		return "success", nil
	}
	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))

	if err != nil {
		return nil, err
	}

	gls.logger.Sugar().Infof("Found %d projects for Gitlab Org %s", len(projectList), gls.cfg.GitLabOrg)
	return projectList, nil
}

func (gls *gitlabScraper) getBranchNames(ctx context.Context, client graphql.Client, projectPath string) (*getBranchNamesProjectRepository, error) {
	var branches *getBranchNamesResponse
	var err error

	operation := func() (string, error) {
		branches, err = getBranchNames(ctx, client, projectPath)
		if err != nil {
			if apiErr, ok := err.(*gitlab.ErrorResponse); ok && apiErr.Response.StatusCode == 429 &&
				apiErr.Response.Status == "429 Too Many Requests" {
				return "", backoff.RetryAfter(60)
			}
			return "", backoff.Permanent(err)
		}
		return "success", nil
	}
	_, err = backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))

	if err != nil {
		return nil, err
	}
	return &branches.Project.Repository, nil
}

func (gls *gitlabScraper) getInitialCommit(ctx context.Context, client *gitlab.Client, projectPath string, defaultBranch string, branch string) (*gitlab.Commit, error) {
	var diff *gitlab.Compare
	var err error

	operation := func() (string, error) {
		diff, _, err = client.Repositories.Compare(projectPath, &gitlab.CompareOptions{From: &defaultBranch, To: &branch})
		if err != nil {
			if apiErr, ok := err.(*gitlab.ErrorResponse); ok && apiErr.Response.StatusCode == 429 &&
				apiErr.Response.Status == "429 Too Many Requests" {
				return "", backoff.RetryAfter(60)
			}
			return "", backoff.Permanent(err)
		}
		return "success", nil
	}

	_, err = backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))

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
	var contributors []*gitlab.Contributor
	var err error

	operation := func() (string, error) {
		contributors, _, err = restClient.Repositories.Contributors(projectPath, nil)
		if err != nil {
			if apiErr, ok := err.(*gitlab.ErrorResponse); ok && apiErr.Response.StatusCode == 429 &&
				apiErr.Response.Status == "429 Too Many Requests" {
				return "", backoff.RetryAfter(60)
			}
			return "", backoff.Permanent(err)
		}
		return "success", nil
	}
	_, err = backoff.Retry(context.Background(), operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))

	if err != nil {
		return 0, err
	}

	return len(contributors), nil
}

func (gls *gitlabScraper) getMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
	state MergeRequestState,
	createdAfter time.Time,
) ([]MergeRequestNode, error) {
	var mergeRequestData []MergeRequestNode
	var mrCursor *string

	for hasNextPage := true; hasNextPage; {
		operation := func() (string, error) {
			mr, err := getMergeRequests(ctx, graphClient, projectPath, mrCursor, state, createdAfter)
			if err != nil {
				if apiErr, ok := err.(*gitlab.ErrorResponse); ok && apiErr.Response.StatusCode == 429 &&
					apiErr.Response.Status == "429 Too Many Requests" {
					return "", backoff.RetryAfter(60)
				}
				return "", backoff.Permanent(err)
			}
			if len(mr.Project.MergeRequests.Nodes) == 0 {
				hasNextPage = false
				return "success", nil
			}
			mrCursor = &mr.Project.MergeRequests.PageInfo.EndCursor
			hasNextPage = mr.Project.MergeRequests.PageInfo.HasNextPage
			mergeRequestData = append(mergeRequestData, mr.Project.MergeRequests.Nodes...)
			return "success", nil
		}
		_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))

		if err != nil {
			return nil, err
		}
	}

	return mergeRequestData, nil
}

func (gls *gitlabScraper) getCombinedMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
	limit int,
) ([]MergeRequestNode, error) {
	createdAfter := time.Time{}

	// If a limit is specified, only retrieve merged MRs from the last X days
	if limit > 0 {
		createdAfter = time.Now().AddDate(0, 0, (-1 * limit))
	}

	// always grab all open MRs
	openMrs, err := gls.getMergeRequests(ctx, graphClient, projectPath, MergeRequestStateOpened, time.Time{})
	if err != nil {
		return nil, err
	}
	mergedMrs, err := gls.getMergeRequests(ctx, graphClient, projectPath, MergeRequestStateMerged, createdAfter)
	if err != nil {
		return nil, err
	}
	mrs := append(openMrs, mergedMrs...)
	return mrs, nil
}
