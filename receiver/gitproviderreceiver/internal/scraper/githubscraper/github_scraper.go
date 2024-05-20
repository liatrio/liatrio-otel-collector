//go:generate ../../../../../.tools/genqlient

package githubscraper

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

type githubScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
	rb       *metadata.ResourceBuilder
}

func (ghs *githubScraper) start(ctx context.Context, host component.Host) (err error) {
	ghs.logger.Sugar().Info("starting the GitHub scraper")
	ghs.client, err = ghs.cfg.ToClient(ctx, host, ghs.settings)
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
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings, metadata.WithStartTime(pcommon.NewTimestampFromTime(time.Now()))),
		rb:       metadata.NewResourceBuilder(cfg.ResourceAttributes),
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

	genClient, restClient, err := ghs.createClients()
	if err != nil {
		ghs.logger.Sugar().Errorf("unable to create clients: %v", zap.Error(err))
	}

	// Do some basic validation to ensure the values provided actually exist in GitHub
	// prior to making queries against that org or user value
	loginType, err := ghs.login(ctx, genClient, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("error logging into GitHub via GraphQL: %v", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	// Generate the search query based on the type, org/user name, and the search_query
	// value if provided
	sq := genDefaultSearchQuery(loginType, ghs.cfg.GitHubOrg)

	if ghs.cfg.SearchQuery != "" {
		sq = ghs.cfg.SearchQuery
		ghs.logger.Sugar().Debugf("using search query where query is: %v", ghs.cfg.SearchQuery)
	}

	repos, count, err := ghs.getRepos(ctx, genClient, sq)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting repo data: %v", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(count))

	var wg sync.WaitGroup
	wg.Add(len(repos))
    var mux sync.Mutex

	for _, repo := range repos {
		repo := repo
		name := repo.Name
		trunk := repo.DefaultBranchRef.Name
		// now := now

		go func() {
			defer wg.Done()

			now := pcommon.NewTimestampFromTime(time.Now())

			branches, count, err := ghs.getBranches(ctx, genClient, name, trunk)
			if err != nil {
				ghs.logger.Sugar().Errorf("error %v getting branch count for repo %s", zap.Error(err), repo.Name)
			}
			// ghs.logger.Sugar().Infof("BRANCH count for repo %s: %d", repo.Name, count)

            mux.Lock()
			ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(count), name)

			for _, branch := range branches {
				// Check if the branch is the default branch or if it is not behind the default branch
				if branch.Name == branch.Repository.DefaultBranchRef.Name || branch.Compare.BehindBy == 0 {
					continue
				}
				// ghs.logger.Sugar().Infof(
				// 	"default branch behind by: %d\n %s branch behind by: %d in repo: %s",
				// 	branch.Compare.BehindBy, branch.Name, branch.Compare.AheadBy, branch.Repository.Name)

				// Yes, this looks weird. The aheadby metric is referring to the number of commits the branch is AHEAD OF the
				// default branch, which in the context of the query is the behind by value. See the above below comment about
				// BehindBy vs AheadBy.
				// ghs.logger.Sugar().Infof("BRANCH %s is ahead by %d commits", branch.Name, branch.Compare.BehindBy)
				ghs.mb.RecordGitRepositoryBranchCommitAheadbyCountDataPoint(now, int64(branch.Compare.BehindBy), branch.Repository.Name, branch.Name)
				// ghs.logger.Sugar().Infof("BRANCH %s is behind by %d commits", branch.Name, branch.Compare.AheadBy)
				ghs.mb.RecordGitRepositoryBranchCommitBehindbyCountDataPoint(now, int64(branch.Compare.AheadBy), branch.Repository.Name, branch.Name)

				adds, dels, age, err := ghs.getCommitInfo(ctx, genClient, branch.Repository.Name, branch)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting commit info: %v", zap.Error(err))
					continue
				}

				// ghs.logger.Sugar().Infof("BRANCH recording branch time %v", age)
				ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, age, branch.Repository.Name, branch.Name)
				// ghs.logger.Sugar().Infof("BRANCH recording branch line adds %d", adds)
				ghs.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(adds), branch.Repository.Name, branch.Name)
				// ghs.logger.Sugar().Infof("BRANCH recording branch line dels %d", dels)
				ghs.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(dels), branch.Repository.Name, branch.Name)
			}

			// Get the contributor count for each of the repositories
			contribs, err := ghs.getContributorCount(ctx, restClient, name)
			if err != nil {
				ghs.logger.Sugar().Errorf("error %v getting contributor count for repo %s", zap.Error(err), repo.Name)
			}
			ghs.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contribs), name)

			prs, err := ghs.getPullRequests(ctx, genClient, name)
			if err != nil {
				ghs.logger.Sugar().Errorf("error %v getting pull requests for repo %s", zap.Error(err), repo.Name)
			}

			// When enabled, process any CVEs for the repository
			if ghs.cfg.Metrics.GitRepositoryCveCount.Enabled {
				cves, err := ghs.getCVEs(ctx, genClient, restClient, name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error %v getting cves from repo %s", zap.Error(err), name)
				}
				for s, c := range cves {
					ghs.mb.RecordGitRepositoryCveCountDataPoint(now, c, name, s)
				}
			}

			var merged int
			var open int

			for _, pr := range prs {
				if pr.Merged {
					merged++

					age := getAge(pr.CreatedAt, pr.MergedAt)

					// ghs.logger.Sugar().Infof("PULL REQUEST: time to merge is %v", age)
					ghs.mb.RecordGitRepositoryPullRequestTimeToMergeDataPoint(now, age, name, pr.HeadRefName)
				} else {
					open++

					age := getAge(pr.CreatedAt, now.AsTime())

					// ghs.logger.Sugar().Infof("PULL REQUEST: open time is %v", age)
					ghs.mb.RecordGitRepositoryPullRequestTimeOpenDataPoint(now, age, name, pr.HeadRefName)

					if pr.Reviews.TotalCount > 0 {
						age := getAge(pr.CreatedAt, pr.Reviews.Nodes[0].CreatedAt)

						// ghs.logger.Sugar().Infof("PULL REQUEST: approval time is %v", age)
						ghs.mb.RecordGitRepositoryPullRequestTimeToApprovalDataPoint(now, age, name, pr.HeadRefName)
					}
				}
			}

			ghs.mb.RecordGitRepositoryPullRequestCountDataPoint(now, int64(open), metadata.AttributePullRequestStateOpen, name)
			ghs.mb.RecordGitRepositoryPullRequestCountDataPoint(now, int64(merged), metadata.AttributePullRequestStateMerged, name)
            mux.Unlock()
		}()
	}

	wg.Wait()

	ghs.rb.SetGitVendorName("github")
	ghs.rb.SetOrganizationName(ghs.cfg.GitHubOrg)

	res := ghs.rb.Emit()
	return ghs.mb.Emit(metadata.WithResource(res)), nil
}
