package gitlabscraper

import (
	"context"
	"time"

	"github.com/xanzy/go-gitlab"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/Khan/genqlient/graphql"
)

type gitlabProject struct {
	Name           string
	Path           string
	CreatedAt      time.Time
	LastActivityAt time.Time
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

func (gls *gitlabScraper) processBranches(client *gitlab.Client, branches *getBranchNamesProjectRepository, projectPath string, now pcommon.Timestamp) {
	gls.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(branches.BranchNames)), projectPath)
	gls.logger.Sugar().Debugf("%s branch count: %v", projectPath, int64(len(branches.BranchNames)))

	for _, branch := range branches.BranchNames {
		if branch == branches.RootRef {
			continue
		}

		commit, err := gls.getInitialCommit(client, projectPath, branches.RootRef, branch)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		if commit != nil {
			branchAge := time.Since(*commit.CreatedAt).Hours()
			gls.logger.Sugar().Debugf("%v age: %v hours, commit name: %s", branch, branchAge, commit.Title)
			gls.mb.RecordGitRepositoryBranchTimeDataPoint(now, int64(branchAge), projectPath, branch)
		}
	}
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

func (gls *gitlabScraper) processMergeRequests(mrs []MergeRequestNode, projectPath string, now pcommon.Timestamp) {
	for _, mr := range mrs {
		gls.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(mr.DiffStatsSummary.Additions), projectPath, mr.SourceBranch)
		gls.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(mr.DiffStatsSummary.Deletions), projectPath, mr.SourceBranch)

		// Checks if the merge request has been merged. This is done with IsZero() which tells us if the
		// time is or isn't  January 1, year 1, 00:00:00 UTC, which is what null in graphql date values
		// get returned as in Go.
		if mr.MergedAt.IsZero() {
			mrAge := int64(time.Since(mr.CreatedAt).Hours())
			gls.mb.RecordGitRepositoryPullRequestTimeOpenDataPoint(now, mrAge, projectPath, mr.SourceBranch)
			gls.logger.Sugar().Debugf("%s merge request for branch %v, age: %v", projectPath, mr.SourceBranch, mrAge)
		} else {
			mergedAge := int64(mr.MergedAt.Sub(mr.CreatedAt).Hours())
			gls.mb.RecordGitRepositoryPullRequestTimeToMergeDataPoint(now, mergedAge, projectPath, mr.SourceBranch)
			gls.logger.Sugar().Debugf("%s merge request for branch %v, merged age: %v", projectPath, mr.SourceBranch, mergedAge)
		}
	}
}
