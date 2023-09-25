package githubscraper

import (
	"context"
	"errors"
	"net/http"
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
	defaultBranch string,
	now pcommon.Timestamp,
) []PullRequestNode {
	var prCursor *string
	var pullRequests []PullRequestNode

	prOpenCount, err := getPullRequestCount(ctx, client, repoName, ghs.cfg.GitHubOrg, []PullRequestState{PullRequestStateOpen})
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting open pull request count", zap.Error(err))
	}
	ghs.logger.Sugar().Debugf("open pull request count: %v for repo %v", prOpenCount, repoName)
	ghs.mb.RecordGitRepositoryPullRequestCountDataPoint(now, int64(prOpenCount.Repository.PullRequests.TotalCount), repoName)

	prMergedCount, err := getPullRequestCount(ctx, client, repoName, ghs.cfg.GitHubOrg, []PullRequestState{PullRequestStateMerged})
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting merged pull request count", zap.Error(err))
	}

	totalPrCount := add(prOpenCount.Repository.PullRequests.TotalCount, prMergedCount.Repository.PullRequests.TotalCount)
	prPages := getNumPages(float64(100), float64(totalPrCount))
	ghs.logger.Sugar().Debugf("pull request pages: %v for repo %v", prPages, repoName)

	for i := 0; i < prPages; i++ {
		pr, err := getPullRequestData(ctx, client, repoName, ghs.cfg.GitHubOrg, 100, prCursor)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting pull request data", zap.Error(err))
		}

		pullRequests = append(pullRequests, pr.Repository.PullRequests.Nodes...)

		prCursor = &pr.Repository.PullRequests.PageInfo.EndCursor
	}
	return pullRequests
}

func (ghs *githubScraper) processPullRequests(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	defaultBranch string,
	now pcommon.Timestamp,
	pullRequests []PullRequestNode,
) {
	for _, pr := range pullRequests {
		if pr.Merged {
			prMergedTime := pr.MergedAt
			mergeAge := int64(prMergedTime.Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestMergeTimeDataPoint(now, mergeAge, repoName, pr.HeadRefName)
			//only exists if the pr is merged
			if pr.MergeCommit.Deployments.TotalCount > 0 {
				deploymentAgeUpperBound := pr.MergeCommit.Deployments.Nodes[0].CreatedAt
				deploymentAge := int64(deploymentAgeUpperBound.Sub(pr.CreatedAt).Hours())
				ghs.mb.RecordGitRepositoryPullRequestDeploymentTimeDataPoint(now, deploymentAge, repoName, pr.HeadRefName)
			}
		} else {
			prAge := int64(now.AsTime().Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, prAge, repoName, pr.HeadRefName)
		}
		if pr.Reviews.TotalCount > 0 {
			approvalAgeUpperBound := pr.Reviews.Nodes[0].CreatedAt
			approvalAge := int64(approvalAgeUpperBound.Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestApprovalTimeDataPoint(now, approvalAge, repoName, pr.HeadRefName)
		}
	}
}

func (ghs *githubScraper) processCommits(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	now pcommon.Timestamp,
	comPages int,
	branch BranchNode,
) {
	comCount := 100
	var cc *string
	var adds int = 0
	var dels int = 0
	for i := 0; i < comPages; i++ {
		if i == comPages-1 {
			comCount = branch.Compare.BehindBy % 100
		}
		c, err := getCommitData(ctx, client, repoName, ghs.cfg.GitHubOrg, 1, comCount, cc, branch.Name)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting commit data", zap.Error(err))
		}

		if len(c.Repository.GetRefs().Nodes) == 0 {
			break
		}
		tar := c.Repository.GetRefs().Nodes[0].GetTarget()
		if ct, ok := tar.(*CommitNodeTargetCommit); ok {
			cc = &ct.History.PageInfo.EndCursor
			if i == comPages-1 {
				e := ct.History.GetEdges()

				oldest := e[len(e)-1].Node.GetCommittedDate()
				age := int64(time.Since(oldest).Hours())

				ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, age, repoName, branch.Name)
			}
			for b := 0; b < len(ct.History.Edges); b++ {
				adds = add(adds, ct.History.Edges[b].Node.Additions)
				dels = add(dels, ct.History.Edges[b].Node.Deletions)
			}
		}
	}
	ghs.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(adds), repoName, branch.Name)
	ghs.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(dels), repoName, branch.Name)
}

func (ghs *githubScraper) getBranches(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	defaultBranch string,
	now pcommon.Timestamp,
) []BranchNode {
	var branchCursor *string
	var branches []BranchNode
	count, err := getBranchCount(ctx, client, repoName, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting branch count", zap.Error(err))
	}
	ghs.logger.Sugar().Debugf("branch count: %v for repo %v", count.Repository.Refs.TotalCount, repoName)

	ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(count.Repository.Refs.TotalCount), repoName)

	bp := getNumPages(float64(50), float64(count.Repository.Refs.TotalCount))
	ghs.logger.Sugar().Debugf("branch pages: %v for repo %v", bp, repoName)

	for i := 0; i < bp; i++ {
		r, err := getBranchData(ctx, client, repoName, ghs.cfg.GitHubOrg, 50, defaultBranch, branchCursor)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting branch data", zap.Error(err))
		}

		branches = append(branches, r.Repository.Refs.Nodes...)

		branchCursor = &r.Repository.Refs.PageInfo.EndCursor
	}
	return branches
}

func (ghs *githubScraper) processBranches(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	defaultBranch string,
	now pcommon.Timestamp,
	branches []BranchNode,
) {
	for _, branch := range branches {
		if branch.Name == defaultBranch || branch.Compare.BehindBy == 0 {
			continue
		}

		ghs.logger.Sugar().Debugf(
			"default branch behind by: %d\n %s branch behind by: %d in repo: %s",
			branch.Compare.BehindBy, branch.Name, branch.Compare.AheadBy, repoName)

		// Yes, this looks weird. The aheadby metric is referring to the number of commits the branch is AHEAD OF the
		// default branch, which in the context of the query is the behind by value. See the above below comment about
		// BehindBy vs AheadBy.
		ghs.mb.RecordGitRepositoryBranchCommitAheadbyCountDataPoint(now, int64(branch.Compare.BehindBy), repoName, branch.Name)
		ghs.mb.RecordGitRepositoryBranchCommitBehindbyCountDataPoint(now, int64(branch.Compare.AheadBy), repoName, branch.Name)

		// We're using BehindBy here because we're comparing against the target
		// branch, which is the default branch. In essence the response is saying
		// the default branch is behind the queried branch by X commits which is
		// the number of commits made to the queried branch but not merged into
		// the default branch. Doing it this way involves less queries because
		// we don't have to know the queried branch name ahead of time.
		cp := getNumPages(float64(100), float64(branch.Compare.BehindBy))
		ghs.processCommits(ctx, client, repoName, now, cp, branch)
	}
}

// scrape and return metrics
func (ghs *githubScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	// ghs.logger.Sugar().Debug("checking if client is initialized")
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
	genClient := graphql.NewClient("https://api.github.com/graphql", ghs.client)

	exists, ownertype, err := ghs.checkOwnerExists(ctx, genClient, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner exists", zap.Error(err))
	}

	typeValid, err := checkOwnerTypeValid(ownertype)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner type is valid", zap.Error(err))
	}

	var data interface{}
	var repoCursor *string

	if !exists || !typeValid {
		ghs.logger.Sugar().Error("error logging in and getting data from github")
		return ghs.mb.Emit(), err
	}

	sq := genDefaultSearchQuery(ownertype, ghs.cfg.GitHubOrg)

	if ghs.cfg.SearchQuery != "" {
		sq = ghs.cfg.SearchQuery
		ghs.logger.Sugar().Debugf("using search query where query is: %v", ghs.cfg.SearchQuery)
	}

	data, err = getRepoData(ctx, genClient, sq, ownertype, repoCursor)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	// TODO: setting this here for access from the proceeding for statement
	// gathering repo data
	var searchRepos []getRepoDataBySearchSearchSearchResultItemConnectionEdgesSearchResultItemEdge

	if searchData, ok := data.(*getRepoDataBySearchResponse); ok {
		ghs.logger.Sugar().Debug("successful search response")
		ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(searchData.Search.RepositoryCount))

		pages := getNumPages(float64(100), float64(searchData.Search.RepositoryCount))
		ghs.logger.Sugar().Debugf("pages: %v", pages)

		for i := 0; i < pages; i++ {
			results := searchData.GetSearch()
			searchRepos = append(searchRepos, results.Edges...)

			repoCursor = &searchData.Search.PageInfo.EndCursor
			data, err = getRepoData(ctx, genClient, sq, ownertype, repoCursor)
			if err != nil {
				ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
			}
		}

		ghs.logger.Sugar().Debugf("repos: %v", searchRepos)

	}

	if _, ok := data.(*getRepoDataBySearchResponse); ok {
		for _, repo := range searchRepos {
			var name string
			var defaultBranch string
			// var branchCursor *string
			// var branches []BranchNode

			if n, ok := repo.Node.(*SearchNodeRepository); ok {
				name = n.Name
				defaultBranch = n.DefaultBranchRef.Name
			}

			// Getting contributor count via the graphql api is very process heavy
			// as you have to get all commits on the default branch and then
			// iterate through each commit to get the author and committer, and remove
			// duplicate values. The default branch could be thousands of commits,
			// which would require tons of pagination and requests to the api. Doing
			// so via the rest api is much more efficient as it's a direct endpoint
			// with limited pagination.
			// Due to the above, we'll only run this actual code when the metric
			// is explicitly enabled.
			if ghs.cfg.MetricsBuilderConfig.Metrics.GitRepositoryContributorCount.Enabled {
				gc := github.NewClient(ghs.client)
				contribs, _, err := gc.Repositories.ListContributors(ctx, ghs.cfg.GitHubOrg, name, nil)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
				}

				contribCount := 0
				if len(contribs) > 0 {
					contribCount = len(contribs)
				}

				ghs.logger.Sugar().Debugf("contributor count: %v for repo %v", contribCount, repo)

				ghs.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contribCount), name)
			}
      
			branches := ghs.getBranches(ctx, genClient, name, defaultBranch, now)
			ghs.processBranches(ctx, genClient, name, defaultBranch, now, branches)
			pullRequests := ghs.getPullRequests(ctx, genClient, name, defaultBranch, now)
			ghs.processPullRequests(ctx, genClient, name, defaultBranch, now, pullRequests)

		}
	}

	return ghs.mb.Emit(), nil
}
