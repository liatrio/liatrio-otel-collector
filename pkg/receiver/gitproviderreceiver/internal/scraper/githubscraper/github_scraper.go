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

func getNumBranchPages(
	ghs *githubScraper,
	ctx context.Context,
	client graphql.Client,
	repoName string,
	now pcommon.Timestamp,
) (int, error) {
	branchCount, err := ghs.getBranchCount(ctx, client, repoName, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting branch count", zap.Error(err))
		return 0, err
	}
	ghs.logger.Sugar().Debugf("%v repo has %v branches", repoName, branchCount)

	ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(branchCount), repoName)

	bp := getNumPages(float64(50), float64(branchCount))
	ghs.logger.Sugar().Debugf("branch pages: %v for repo %v", bp, repoName)
	return bp, nil
}

func getBranchInfo(
	ghs *githubScraper,
	ctx context.Context,
	client graphql.Client,
	repoName string,
	owner string,
	branchPages int,
	defaultBranch string,
) ([]BranchNode, error) {
	var branchCursor *string
	var branches []BranchNode
	for i := 0; i < branchPages; i++ {
		r, err := getBranchData(ctx, client, repoName, ghs.cfg.GitHubOrg, 50, defaultBranch, branchCursor)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting branch data", zap.Error(err))
			return nil, err
		}

		branches = append(branches, r.Repository.Refs.Nodes...)
		branchCursor = &r.Repository.Refs.PageInfo.EndCursor
		if !r.Repository.Refs.PageInfo.HasNextPage {
			break
		}
	}
	return branches, nil
}

func getNumPrPages(
	ghs *githubScraper,
	ctx context.Context,
	client graphql.Client,
	repoName string,
	now pcommon.Timestamp,
) (int, error) {
	prOpenCount, err := ghs.getPrCount(ctx, client, repoName, ghs.cfg.GitHubOrg, []PullRequestState{PullRequestStateOpen})
	if err != nil {
		return 0, err
	}
	ghs.logger.Sugar().Debugf("open pull request count: %v for repo %v", prOpenCount, repoName)
	ghs.mb.RecordGitRepositoryPullRequestOpenCountDataPoint(now, int64(prOpenCount), repoName)

	prMergedCount, err := ghs.getPrCount(ctx, client, repoName, ghs.cfg.GitHubOrg, []PullRequestState{PullRequestStateMerged})
	if err != nil {
		return 0, err
	}
	ghs.logger.Sugar().Debugf("merged pull request count: %v for repo %v", prMergedCount, repoName)
	ghs.mb.RecordGitRepositoryPullRequestMergedCountDataPoint(now, int64(prMergedCount), repoName)

	totalPrCount := add(prOpenCount, prMergedCount)
	prPages := getNumPages(float64(100), float64(totalPrCount))
	ghs.logger.Sugar().Debugf("pull request pages: %v for repo %v", prPages, repoName)
	return prPages, err
}

func getPrData(
	ghs *githubScraper,
	ctx context.Context,
	client graphql.Client,
	prPages int,
	repoName string,
	owner string,
) ([]PullRequestNode, error) {
	var prCursor *string
	var pullRequests []PullRequestNode

	for i := 0; i < prPages; i++ {
		prs, err := getPullRequestData(ctx, client, repoName, ghs.cfg.GitHubOrg, 100, prCursor)
		if err != nil {
			return nil, err
		}

		pullRequests = append(pullRequests, prs.Repository.PullRequests.Nodes...)
		prCursor = &prs.Repository.PullRequests.PageInfo.EndCursor

		if !prs.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
	}
	return pullRequests, nil
}
func getPullRequests(
	ghs *githubScraper,
	ctx context.Context,
	client graphql.Client,
	repo SearchNodeRepository,
	now pcommon.Timestamp,
) ([]PullRequestNode, error) {

	prPages, err := getNumPrPages(ghs, ctx, client, repo.Name, now)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting total pr pages", zap.Error(err))
		return nil, err
	}
	pullRequests, err := getPrData(ghs, ctx, client, prPages, repo.Name, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting pr data", zap.Error(err))
		return nil, err
	}

	if pullRequests != nil {
		return pullRequests, nil
	}
	return nil, nil
}

func processPullRequests(
	ghs *githubScraper,
	ctx context.Context,
	client graphql.Client,
	now pcommon.Timestamp,
	pullRequests []PullRequestNode,
) {
	for _, pr := range pullRequests {
		if pr.Merged {
			prMergedTime := pr.MergedAt
			mergeAge := int64(prMergedTime.Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestMergeTimeDataPoint(now, mergeAge, pr.Repository.Name, pr.HeadRefName)
			//only exists if the pr is merged
			if pr.MergeCommit.Deployments.TotalCount > 0 {
				deploymentAgeUpperBound := pr.MergeCommit.Deployments.Nodes[0].CreatedAt
				deploymentAge := int64(deploymentAgeUpperBound.Sub(pr.CreatedAt).Hours())
				ghs.mb.RecordGitRepositoryPullRequestDeploymentTimeDataPoint(now, deploymentAge, pr.Repository.Name, pr.HeadRefName)
			}
		} else {
			prAge := int64(now.AsTime().Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, prAge, pr.Repository.Name, pr.HeadRefName)
		}
		if pr.Reviews.TotalCount > 0 {
			approvalAgeUpperBound := pr.Reviews.Nodes[0].CreatedAt
			approvalAge := int64(approvalAgeUpperBound.Sub(pr.CreatedAt).Hours())
			ghs.mb.RecordGitRepositoryPullRequestApprovalTimeDataPoint(now, approvalAge, pr.Repository.Name, pr.HeadRefName)
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
		c, err := getCommitData(context.Background(), client, repoName, ghs.cfg.GitHubOrg, 1, comCount, cc, branch.Name)
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
	repo SearchNodeRepository,
	now pcommon.Timestamp,
) ([]BranchNode, error) {

	var defaultBranch string = repo.DefaultBranchRef.Name

	bp, err := getNumBranchPages(ghs, ctx, client, repo.Name, now)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting number of pages for branch data", zap.Error(err))
		return nil, err
	}

	branches, err := getBranchInfo(ghs, ctx, client, repo.Name, ghs.cfg.GitHubOrg, bp, defaultBranch)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting branch info", zap.Error(err))
		return nil, err
	}

	if branches != nil {
		return branches, nil
	}
	return nil, nil

}

func (ghs *githubScraper) processBranches(
	ctx context.Context,
	client graphql.Client,
	now pcommon.Timestamp,
	branches []BranchNode,
) {

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
		ghs.processCommits(ctx, client, branch.Repository.Name, now, cp, branch)
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

	restClient := github.NewClient(ghs.client)

	// Enable the ability to override the endpoint for self-hosted github instances
	if ghs.cfg.HTTPClientSettings.Endpoint != "" {
		restCURL := ghs.cfg.HTTPClientSettings.Endpoint

		restClient, err = github.NewEnterpriseClient(restCURL, restCURL, ghs.client)
		if err != nil {
			ghs.logger.Sugar().Errorf("error: %v", err)
		}
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
	var searchRepos []SearchNodeRepository

	if searchData, ok := data.(*getRepoDataBySearchResponse); ok {
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
			searchData, err = getRepoData(ctx, genClient, sq, ownertype, repoCursor)
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

		//pullrequest information
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				prs, err := getPullRequests(ghs, ctx, genClient, searchRepos[i], now)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting pull requests", zap.Error(err))
				}
				if prs != nil {
					processPullRequests(ghs, ctx, genClient, now, prs)
				}
				<-sem
			}()
		}

		//branch information
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				branches, err := ghs.getBranches(ctx, genClient, searchRepos[i], now)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting branches", zap.Error(err))
				}
				if branches != nil {
					ghs.processBranches(ctx, genClient, now, branches)
				}
				<-sem
			}()
		}

		//contributor count
		for i := 0; i < len(searchRepos); i++ {
			i := i
			sem <- 1
			go func() {
				contribCount, err := ghs.getContributorCount(ctx, restClient, searchRepos[i], now)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
				}
				ghs.logger.Sugar().Debugf("contributor count: %v for repo %v", contribCount, searchRepos[i].Name)
				ghs.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contribCount), searchRepos[i].Name)
				<-sem
			}()
		}

		//wait until all goroutines are finished
		for i := 0; i < maxProcesses; i++ {
			sem <- 1
		}
	}

	return ghs.mb.Emit(), nil
}
