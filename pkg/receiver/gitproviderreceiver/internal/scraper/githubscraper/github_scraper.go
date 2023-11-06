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

// Not sure if this needs to be here after the refactor
type PullRequest struct {
	Title       string
	CreatedDate time.Time
	ClosedDate  time.Time
}

type Repo struct {
	Name          string
	Owner         string
	DefaultBranch string
	PullRequests  []PullRequest
}

type githubScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
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

func (ghs *githubScraper) getPullRequests(
	ctx context.Context,
	client graphql.Client,
	repoName string,
) ([]PullRequestNode, error) {
	var prCursor *string
	var pullRequests []PullRequestNode

	for hasNextPage := true; hasNextPage; {
		prs, err := getPullRequestData(ctx, client, repoName, ghs.cfg.GitHubOrg, 100, prCursor, []PullRequestState{"OPEN", "MERGED"})
		if err != nil {
			return nil, err
		}
		pullRequests = append(pullRequests, prs.Repository.PullRequests.Nodes...)
		prCursor = &prs.Repository.PullRequests.PageInfo.EndCursor
		hasNextPage = prs.Repository.PullRequests.PageInfo.HasNextPage
	}
	return pullRequests, nil
}

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
	for _, pr := range pullRequests {
		if pr.Merged {
			mergedCount++
			prMergedTime := pr.MergedAt
			mergeAge := int64(prMergedTime.Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestMergeTimeDataPoint(now, mergeAge, repoName, pr.HeadRefName)
			// only exists if the pr is merged
			if pr.MergeCommit.Deployments.TotalCount > 0 {
				deploymentAgeUpperBound := pr.MergeCommit.Deployments.Nodes[0].CreatedAt
				deploymentAge := int64(deploymentAgeUpperBound.Sub(pr.CreatedAt).Hours())
				ghs.mb.RecordGitRepositoryPullRequestDeploymentTimeDataPoint(now, deploymentAge, repoName, pr.HeadRefName)
			}
		} else {
			openCount++
			prAge := int64(now.AsTime().Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, prAge, repoName, pr.HeadRefName)
		}
		if pr.Reviews.TotalCount > 0 {
			approvalAgeUpperBound := pr.Reviews.Nodes[0].CreatedAt
			approvalAge := int64(approvalAgeUpperBound.Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestApprovalTimeDataPoint(now, approvalAge, repoName, pr.HeadRefName)
		}
	}
	ghs.mb.RecordGitRepositoryPullRequestOpenCountDataPoint(now, int64(openCount), repoName)
	ghs.mb.RecordGitRepositoryPullRequestMergedCountDataPoint(now, int64(mergedCount), repoName)
}

func (ghs *githubScraper) getCommitInfo(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	now pcommon.Timestamp,
	comPages int,
	branch BranchNode,
) (int, int, int64, error) {
	comCount := 100
	var cc *string
	var adds int = 0
	var dels int = 0
	var age int64 = 0

	for nPage := 1; nPage <= comPages; nPage++ {
		if nPage == comPages {
			comCount = branch.Compare.BehindBy % 100
			// When the last page is full
			if comCount == 0 {
				comCount = 100
			}
		}
		c, err := ghs.getCommitData(context.Background(), client, repoName, ghs.cfg.GitHubOrg, comCount, cc, branch.Name)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting commit data", zap.Error(err))
			return 0, 0, 0, err
		}

		if len(c.Edges) == 0 {
			break
		}
		cc = &c.PageInfo.EndCursor
		if nPage == comPages {
			e := c.GetEdges()
			oldest := e[len(e)-1].Node.GetCommittedDate()
			age = int64(time.Since(oldest).Hours())
		}
		for b := 0; b < len(c.Edges); b++ {
			adds = add(adds, c.Edges[b].Node.Additions)
			dels = add(dels, c.Edges[b].Node.Deletions)
		}

	}
	return adds, dels, age, nil
}

func (ghs *githubScraper) getBranches(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	defaultBranch string,
) ([]BranchNode, error) {

	var branchCursor *string
	var branches []BranchNode
	for hasNextPage := true; hasNextPage; {
		r, err := getBranchData(ctx, client, repoName, ghs.cfg.GitHubOrg, 50, defaultBranch, branchCursor)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting branch data", zap.Error(err))
			return nil, err
		}
		branches = append(branches, r.Repository.Refs.Nodes...)
		branchCursor = &r.Repository.Refs.PageInfo.EndCursor
		hasNextPage = r.Repository.Refs.PageInfo.HasNextPage
	}
	return branches, nil
}

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
		cp := getNumPages(float64(100), float64(branch.Compare.BehindBy))
		adds, dels, age, err := ghs.getCommitInfo(ctx, client, branch.Repository.Name, now, cp, branch)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting commit info", zap.Error(err))
			continue
		}

		ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, age, branch.Repository.Name, branch.Name)
		ghs.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(adds), branch.Repository.Name, branch.Name)
		ghs.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(dels), branch.Repository.Name, branch.Name)
	}
}

func (ghs *githubScraper) getContributorCount(
	ctx context.Context,
	client *github.Client,
	repo SearchNodeRepository,
	now pcommon.Timestamp,
) (int, error) {
	var err error

	contribs, _, err := client.Repositories.ListContributors(ctx, ghs.cfg.GitHubOrg, repo.Name, nil)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
		return 0, err
	}

	contribCount := 0
	if len(contribs) > 0 {
		contribCount = len(contribs)
	}
	return contribCount, nil
}

func (ghs *githubScraper) getRepoData(
	ctx context.Context,
	client graphql.Client,
	sq string,
	ownertype string,
) ([]SearchNodeRepository, error) {
	var searchRepos []SearchNodeRepository
	var repoCursor *string
	for hasNextPage := true; hasNextPage; {
		data, err := getRepoData(ctx, client, sq, ownertype, repoCursor)
		if err != nil {
			ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
			return nil, err
		}
		ghs.logger.Sugar().Debug("successful search response")

		for _, repo := range data.Search.Nodes {
			if r, ok := repo.(*SearchNodeRepository); ok {
				searchRepos = append(searchRepos, *r)
			}
		}
		repoCursor = &data.Search.PageInfo.EndCursor
		hasNextPage = data.Search.PageInfo.HasNextPage
	}

	ghs.logger.Sugar().Debugf("repos: %v", searchRepos)
	if len(searchRepos) == 0 {
		return nil, nil
	}
	return searchRepos, nil
}

// scrape and return metrics
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
	graphCURL := "https://api.github.com/graphql"

	if ghs.cfg.HTTPClientSettings.Endpoint != "" {
		var err error

		// GitHub Enterprise (ghe) URL : http(s)://HOSTNAME/api/graphql
		// https://docs.github.com/en/enterprise-server@3.8/graphql/guides/forming-calls-with-graphql#the-graphql-endpoint
		graphCURL, err = url.JoinPath(ghs.cfg.HTTPClientSettings.Endpoint, "api/graphql")
		if err != nil {
			ghs.logger.Sugar().Errorf("error: %v", err)
		}
	}

	genClient := graphql.NewClient(graphCURL, ghs.client)

	exists, ownertype, err := ghs.checkOwnerExists(ctx, genClient, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner exists", zap.Error(err))
	}

	typeValid, err := checkOwnerTypeValid(ownertype)
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
		restCURL := ghs.cfg.HTTPClientSettings.Endpoint

		restClient, err = github.NewEnterpriseClient(restCURL, restCURL, ghs.client)
		if err != nil {
			ghs.logger.Sugar().Errorf("error: %v", err)
		}
	}

	if !exists || !typeValid {
		ghs.logger.Sugar().Error("error logging in and getting data from github")
		return ghs.mb.Emit(), err
	}

	sq := genDefaultSearchQuery(ownertype, ghs.cfg.GitHubOrg)

	if ghs.cfg.SearchQuery != "" {
		sq = ghs.cfg.SearchQuery
		ghs.logger.Sugar().Debugf("using search query where query is: %v", ghs.cfg.SearchQuery)
	}

	searchRepos, err := ghs.getRepoData(ctx, genClient, sq, ownertype)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting repo data", zap.Error(err))
		return ghs.mb.Emit(), err
	}
	ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(len(searchRepos)))

	if searchRepos != nil {
		ghs.logger.Sugar().Debugf("There are %v repos", len(searchRepos))

		var maxProcesses int = 3
		sem := make(chan int, maxProcesses)

		// pullrequest information
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				prs, err := ghs.getPullRequests(ctx, genClient, searchRepos[i].Name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting pull requests", zap.Error(err))
					<-sem
					return
				}
				processPullRequests(ghs, ctx, genClient, now, prs, searchRepos[i].Name)
				<-sem
			}()
		}

		// branch information
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				branches, err := ghs.getBranches(ctx, genClient, searchRepos[i].Name, searchRepos[i].DefaultBranchRef.Name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting branches", zap.Error(err))
					<-sem
					return
				}
				ghs.processBranches(ctx, genClient, now, branches, searchRepos[i].Name)
				<-sem
			}()
		}

		// contributor count
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				contribCount, err := ghs.getContributorCount(ctx, restClient, searchRepos[i], now)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
					<-sem
					return
				}
				ghs.logger.Sugar().Debugf("contributor count: %v for repo %v", contribCount, searchRepos[i].Name)
				ghs.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contribCount), searchRepos[i].Name)
				<-sem
			}()
		}

		// wait until all goroutines are finished
		for i := 0; i < maxProcesses; i++ {
			sem <- 1
		}
	}

	return ghs.mb.Emit(), nil
}
