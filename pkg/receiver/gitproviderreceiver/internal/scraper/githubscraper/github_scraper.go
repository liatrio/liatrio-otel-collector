package githubscraper

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/go-github/v53/github"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

// A struct representing the GitHubScraper.
type githubScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
}

// A struct to hold the data for a commit.
type commitInfo struct {
	additions int
	deletions int
	age       int64
}

func (ghs *githubScraper) start(_ context.Context, host component.Host) (err error) {
	ghs.logger.Sugar().Info("Starting the scraper inside scraper.go")
	// TODO: Fix the ToClient configuration
	ghs.client, err = ghs.cfg.ToClient(host, ghs.settings)
	return
}

func newGitHubScraper(
	_ context.Context,
	settings receiver.CreateSettings,
	cfg *Config,
) *githubScraper {
	return &githubScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}
}

// Retrieves the pull request data for a given repository.
func (ghs *githubScraper) getPullRequests(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	pullRequestStates []PullRequestState,
) ([]PullRequestNode, error) {
	var pullRequests []PullRequestNode
	var pullRequestCursor *string

	for hasNextPage := true; hasNextPage; {
		pullRequestData, err := getPullRequestData(ctx, client, repoName, ghs.cfg.GitHubOrg, 100, pullRequestCursor, pullRequestStates)
		if err != nil {
			return nil, err
		}

		pullRequests = append(pullRequests, pullRequestData.Repository.PullRequests.Nodes...)
		hasNextPage = pullRequestData.Repository.PullRequests.PageInfo.HasNextPage
		pullRequestCursor = &pullRequestData.Repository.PullRequests.PageInfo.EndCursor
	}

	return pullRequests, nil
}

// Iterates over the given pullRequests and records the relevant metrics.
func processPullRequests(
	ghs *githubScraper,
	ctx context.Context,
	client graphql.Client,
	now pcommon.Timestamp,
	pullRequests []PullRequestNode,
	repoName string,
) {
	var mergedCount int
	var openCount int

	for _, pullRequest := range pullRequests {
		if pullRequest.Merged {
			mergedCount++
			mergeAge := int64(pullRequest.MergedAt.Sub(pullRequest.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestMergeTimeDataPoint(now, mergeAge, repoName, pullRequest.HeadRefName)

			// Only exists if the pr is merged
			if pullRequest.MergeCommit.Deployments.TotalCount > 0 {
				deploymentAgeUpperBound := pullRequest.MergeCommit.Deployments.Nodes[0].CreatedAt
				deploymentAge := int64(deploymentAgeUpperBound.Sub(pullRequest.CreatedAt).Hours())
				ghs.mb.RecordGitRepositoryPullRequestDeploymentTimeDataPoint(now, deploymentAge, repoName, pullRequest.HeadRefName)
			}
		} else {
			openCount++
			pullRequestAge := int64(now.AsTime().Sub(pullRequest.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, pullRequestAge, repoName, pullRequest.HeadRefName)
		}

		if pullRequest.Reviews.TotalCount > 0 {
			approvalAgeUpperBound := pullRequest.Reviews.Nodes[0].CreatedAt
			approvalAge := int64(approvalAgeUpperBound.Sub(pullRequest.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestApprovalTimeDataPoint(now, approvalAge, repoName, pullRequest.HeadRefName)
		}
	}

	ghs.mb.RecordGitRepositoryPullRequestOpenCountDataPoint(now, int64(openCount), repoName)
	ghs.mb.RecordGitRepositoryPullRequestMergedCountDataPoint(now, int64(mergedCount), repoName)
}

// Retrieves the commit data for a given branch of a given repository.
func (ghs *githubScraper) getCommitInfo(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	now pcommon.Timestamp,
	commitPages int,
	branch BranchNode,
) (*commitInfo, error) {
	var commitCursor *string
	var commitCount int = 100
	commitInfo := commitInfo{
		additions: 0,
		deletions: 0,
		age:       0,
	}

	for pageNum := 1; pageNum <= commitPages; pageNum++ {
		if pageNum == commitPages {
			commitCount = branch.Compare.BehindBy % 100
			// When the last page is full
			if commitCount == 0 {
				commitCount = 100
			}
		}

		commitData, err := ghs.getCommitData(context.Background(), client, repoName, ghs.cfg.GitHubOrg, commitCount, commitCursor, branch.Name)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting commit data", zap.Error(err))
			return &commitInfo, err
		}

		if len(commitData.Edges) == 0 {
			break
		}

		commitCursor = &commitData.PageInfo.EndCursor
		if pageNum == commitPages {
			edges := commitData.GetEdges()
			oldest := edges[len(edges)-1].Node.GetCommittedDate()
			commitInfo.age = int64(time.Since(oldest).Hours())
		}

		for b := 0; b < len(commitData.Edges); b++ {
			commitInfo.additions = add(commitInfo.additions, commitData.Edges[b].Node.Additions)
			commitInfo.deletions = add(commitInfo.deletions, commitData.Edges[b].Node.Deletions)
		}
	}

	return &commitInfo, nil
}

// Retrieves data for the given branch of the given repo.
func (ghs *githubScraper) getBranches(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	defaultBranch string,
) ([]BranchNode, error) {
	var branchCursor *string
	var branches []BranchNode
	for hasNextPage := true; hasNextPage; {
		branchData, err := getBranchData(ctx, client, repoName, ghs.cfg.GitHubOrg, 50, defaultBranch, branchCursor)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting branch data", zap.Error(err))
			return nil, err
		}

		branchCursor = &branchData.Repository.Refs.PageInfo.EndCursor
		hasNextPage = branchData.Repository.Refs.PageInfo.HasNextPage
		branches = append(branches, branchData.Repository.Refs.Nodes...)
	}

	return branches, nil
}

// Iterates over the given branches of a given repo and records the relevant
// metrics.
func (ghs *githubScraper) processBranches(
	ctx context.Context,
	client graphql.Client,
	now pcommon.Timestamp,
	branches []BranchNode,
	repoName string,
) {
	ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(branches)), repoName)

	for _, branch := range branches {
		if branch.Name == branch.Repository.DefaultBranchRef.Name || branch.Compare.BehindBy == 0 {
			continue
		}
		ghs.logger.Sugar().Debugf(
			"default branch behind by: %d\n %s branch behind by: %d in repo: %s",
			branch.Compare.BehindBy, branch.Name, branch.Compare.AheadBy, branch.Repository.Name)

		// Yes, this looks weird. The aheadby metric is referring to the number of commits the branch is AHEAD OF the
		// default branch, which in the context of the query is the behind by value. See the above below comment about
		// BehindBy vs AheadBy.
		ghs.mb.RecordGitRepositoryBranchCommitAheadbyCountDataPoint(now, int64(branch.Compare.BehindBy), branch.Repository.Name, branch.Name)
		ghs.mb.RecordGitRepositoryBranchCommitBehindbyCountDataPoint(now, int64(branch.Compare.AheadBy), branch.Repository.Name, branch.Name)

		// We're using BehindBy here because we're comparing against the target
		// branch, which is the default branch. In essence the response is saying
		// the default branch is behind the queried branch by X commits which is
		// the number of commits made to the queried branch but not merged into
		// the default branch. Doing it this way involves less queries because
		// we don't have to know the queried branch name ahead of time.
		commitPages := getNumPages(float64(100), float64(branch.Compare.BehindBy))
		commitInfo, err := ghs.getCommitInfo(ctx, client, branch.Repository.Name, now, commitPages, branch)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting commit info", zap.Error(err))
			continue
		}

		ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, commitInfo.age, branch.Repository.Name, branch.Name)
		ghs.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(commitInfo.additions), branch.Repository.Name, branch.Name)
		ghs.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(commitInfo.deletions), branch.Repository.Name, branch.Name)
	}
}

// Retrieves the amount of contributors for a given repository.
func (ghs *githubScraper) getContributorCount(
	ctx context.Context,
	client *github.Client,
	repo SearchNodeRepository,
) (int, error) {
	var err error

	contributors, _, err := client.Repositories.ListContributors(ctx, ghs.cfg.GitHubOrg, repo.Name, nil)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
		return 0, err
	}

	return len(contributors), nil
}

// Perform the actual scraping of GitHub data.
func (ghs *githubScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	if ghs.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	ghs.logger.Sugar().Debugf("current time: %v", now)

	currentDate := time.Now().Day()
	ghs.logger.Sugar().Debugf("current date: %v", currentDate)

	ghs.logger.Sugar().Debug("creating a new github client")

	// TODO: Below is the beginning of the refactor to using genqlient
	// This is a secondary instantiation of the GraphQL client for the purpose of
	// using genqlient during the refactor.

	// Enable the ability to override the endpoint for self-hosted github instances
	// GitHub Free URL : https://api.github.com/graphql
	// https://docs.github.com/en/graphql/guides/forming-calls-with-graphql#the-graphql-endpoint
	graphClientURL := "https://api.github.com/graphql"

	if ghs.cfg.HTTPClientSettings.Endpoint != "" {
		var err error

		// GitHub Enterprise (ghe) URL : http(s)://HOSTNAME/api/graphql
		// https://docs.github.com/en/enterprise-server@3.8/graphql/guides/forming-calls-with-graphql#the-graphql-endpoint
		graphClientURL, err = url.JoinPath(ghs.cfg.HTTPClientSettings.Endpoint, "api/graphql")
		if err != nil {
			ghs.logger.Sugar().Errorf("error: %v", err)
		}
	}

	graphClient := graphql.NewClient(graphClientURL, ghs.client)

	exists, ownertype, err := ghs.checkOwnerExists(ctx, graphClient, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner exists", zap.Error(err))
	}

	validOwnerType, err := checkOwnerTypeValid(ownertype)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner type is valid", zap.Error(err))
	}

	// GitHub Free URL : https://api.github.com/octocat
	// https://docs.github.com/en/rest/guides/getting-started-with-the-rest-api?apiVersion=2022-11-28
	// Already managed by Client.initialize(...) with http(s)://HOSTNAME
	// https://github.com/google/go-github/blob/master/github/github.go#L33
	// https://github.com/google/go-github/blob/master/github/github.go#L386
	restClient := github.NewClient(ghs.client)

	// Enable the ability to override the endpoint for self-hosted github instances
	if ghs.cfg.HTTPClientSettings.Endpoint != "" {
		// GitHub Enterprise URL (ghe) : http(s)://HOSTNAME/api/v3/octocat
		// https://docs.github.com/en/enterprise-server@3.8/rest/guides/getting-started-with-the-rest-api#making-a-request
		// Already Managed by Client.WithEnterpriseURLs(...) with http(s)://HOSTNAME
		// https://github.com/google/go-github/blob/master/github/github.go#L351
		restClientURL := ghs.cfg.HTTPClientSettings.Endpoint

		restClient, err = github.NewEnterpriseClient(restClientURL, restClientURL, ghs.client)
		if err != nil {
			ghs.logger.Sugar().Errorf("error: %v", err)
		}
	}

	var repoData interface{}
	var repoCursor *string

	if !exists || !validOwnerType {
		ghs.logger.Sugar().Error("error logging in and getting data from github")
		return ghs.mb.Emit(), err
	}

	searchQuery := genDefaultSearchQuery(ownertype, ghs.cfg.GitHubOrg)

	if ghs.cfg.SearchQuery != "" {
		searchQuery = ghs.cfg.SearchQuery
		ghs.logger.Sugar().Debugf("using search query where query is: %v", ghs.cfg.SearchQuery)
	}

	repoData, err = getRepoData(ctx, graphClient, searchQuery, ownertype, repoCursor)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	var searchRepos []SearchNodeRepository

	if searchData, ok := repoData.(*getRepoDataBySearchResponse); ok {
		ghs.logger.Sugar().Debug("successful search response")
		ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(searchData.Search.RepositoryCount))

		pages := getNumPages(float64(100), float64(searchData.Search.RepositoryCount))
		ghs.logger.Sugar().Debugf("pages: %v", pages)

		for i := 0; i < pages; i++ {
			results := searchData.GetSearch()
			for _, repo := range results.Nodes {
				if r, ok := repo.(*SearchNodeRepository); ok {
					searchRepos = append(searchRepos, *r)
				}
			}

			repoCursor = &searchData.Search.PageInfo.EndCursor
			searchData, err = getRepoData(ctx, graphClient, searchQuery, ownertype, repoCursor)
			if err != nil {
				ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
			}
		}

		ghs.logger.Sugar().Debugf("repos: %v", searchRepos)

	}

	if searchRepos != nil {
		ghs.logger.Sugar().Debugf("There are %v repos", len(searchRepos))

		var maxProcesses int = 3
		sem := make(chan int, maxProcesses)

		// Gather & process PR info.
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				pullRequestStates := []PullRequestState{"OPEN", "MERGED"}
				pullRequests, err := ghs.getPullRequests(ctx, graphClient, searchRepos[i].Name, pullRequestStates)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting pull requests", zap.Error(err))
					<-sem
					return
				}
				processPullRequests(ghs, ctx, graphClient, now, pullRequests, searchRepos[i].Name)
				<-sem
			}()
		}

		// Gather & process branch info.
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				branches, err := ghs.getBranches(ctx, graphClient, searchRepos[i].Name, searchRepos[i].DefaultBranchRef.Name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting branches", zap.Error(err))
					<-sem
					return
				}
				ghs.processBranches(ctx, graphClient, now, branches, searchRepos[i].Name)
				<-sem
			}()
		}

		// Gather & process contributor info.
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				contributorCount, err := ghs.getContributorCount(ctx, restClient, searchRepos[i])
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
					<-sem
					return
				}
				ghs.logger.Sugar().Debugf("contributor count: %v for repo %v", contributorCount, searchRepos[i].Name)
				ghs.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contributorCount), searchRepos[i].Name)
				<-sem
			}()
		}

		// Wait until all goroutines are finished.
		for i := 0; i < maxProcesses; i++ {
			sem <- 1
		}
	}

	return ghs.mb.Emit(), nil
}
