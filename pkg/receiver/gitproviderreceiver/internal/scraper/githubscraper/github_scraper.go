package githubscraper

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
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

func (ghs *githubScraper) processBranches(
	ctx context.Context,
	client graphql.Client,
	now pcommon.Timestamp,
	branches []BranchNode,
	repoName string,
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

	genClient, restClient, err := ghs.createClients()
	if err != nil {
		ghs.logger.Sugar().Errorf("unable to create clients", zap.Error(err))
	}

	exists, ownertype, err := ghs.checkOwnerExists(ctx, genClient, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner exists", zap.Error(err))
	}

	typeValid, err := checkOwnerTypeValid(ownertype)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner type is valid", zap.Error(err))
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

	repos, count, err := ghs.getRepos(ctx, genClient, sq)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting repo data", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(count))

	if repos != nil {
		ghs.logger.Sugar().Debugf("There are %v repos", len(repos))

		var maxProcesses int = 3
		sem := make(chan int, maxProcesses)

		// pullrequest information
		for i := 0; i < len(repos); i++ {
			i := i
			sem <- 1
			go func() {
				prs, err := ghs.getPullRequests(ctx, genClient, repos[i].Name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting pull requests", zap.Error(err))
					<-sem
					return
				}
				processPullRequests(ghs, ctx, genClient, now, prs, repos[i].Name)
				<-sem
			}()
		}

		// branch information
		for i := 0; i < len(repos); i++ {
			i := i
			sem <- 1
			go func() {
				branches, count, err := ghs.getBranches(ctx, genClient, repos[i].Name, repos[i].DefaultBranchRef.Name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting branches", zap.Error(err))
					<-sem
					return
				}
				ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(count), repos[0].Name)
				ghs.processBranches(ctx, genClient, now, branches, repos[i].Name)
				<-sem
			}()
		}

		// contributor count
		for i := 0; i < len(repos); i++ {
			i := i
			sem <- 1
			go func() {
				contribCount, err := ghs.getContributorCount(ctx, restClient, repos[i].Name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
					<-sem
					return
				}
				ghs.logger.Sugar().Debugf("contributor count: %v for repo %v", contribCount, repos[i].Name)
				ghs.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contribCount), repos[i].Name)
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
