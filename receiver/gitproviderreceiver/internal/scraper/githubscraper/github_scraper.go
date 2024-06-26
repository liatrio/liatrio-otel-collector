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
		now := now

		go func() {
			defer wg.Done()

			branches, count, err := ghs.getBranches(ctx, genClient, name, trunk)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting branch count: %v", zap.Error(err))
			}

			// Create a mutual exclusion lock to prevent the recordDataPoint
			// from having a nil pointer error passing in the SetStartTimestamp
			mux.Lock()
			ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(count), name)

			for _, branch := range branches {
				// See
				// https://github.com/liatrio/liatrio-otel-collector/blob/main/receiver/gitproviderreceiver/internal/scraper/githubscraper/README.md#github-limitations
				// for more information as to why we do not emit metrics for
				// the default branch (trunk) nor any branch with no changes to
				// it.
				if branch.Name == branch.Repository.DefaultBranchRef.Name || branch.Compare.BehindBy == 0 {
					continue
				}

				// See
				// https://github.com/liatrio/liatrio-otel-collector/blob/main/receiver/gitproviderreceiver/internal/scraper/githubscraper/README.md#github-limitations
				// for more information as to why `BehindBy` and `AheadBy` are
				// swapped.
				ghs.mb.RecordGitRepositoryBranchCommitAheadbyCountDataPoint(now, int64(branch.Compare.BehindBy), branch.Repository.Name, branch.Name)
				ghs.mb.RecordGitRepositoryBranchCommitBehindbyCountDataPoint(now, int64(branch.Compare.AheadBy), branch.Repository.Name, branch.Name)

				adds, dels, age, err := ghs.evalCommits(ctx, genClient, branch.Repository.Name, branch)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting commit info: %v", zap.Error(err))
					continue
				}

				ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, age, branch.Repository.Name, branch.Name)
				ghs.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(adds), branch.Repository.Name, branch.Name)
				ghs.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(dels), branch.Repository.Name, branch.Name)
			}

			// Get the contributor count for each of the repositories
			contribs, err := ghs.getContributorCount(ctx, restClient, name)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting contributor: %v", zap.Error(err))
			}
			ghs.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contribs), name)

			prs, err := ghs.getPullRequests(ctx, genClient, name)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting pull requests: %v", zap.Error(err))
			}

			// When enabled, process any CVEs for the repository
			if ghs.cfg.Metrics.GitRepositoryCveCount.Enabled {
				cves, err := ghs.getCVEs(ctx, genClient, restClient, name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting cves: %v", zap.Error(err))
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

					ghs.mb.RecordGitRepositoryPullRequestTimeToMergeDataPoint(now, age, name, pr.HeadRefName)
				} else {
					open++

					age := getAge(pr.CreatedAt, now.AsTime())

					ghs.mb.RecordGitRepositoryPullRequestTimeOpenDataPoint(now, age, name, pr.HeadRefName)

					if pr.Reviews.TotalCount > 0 {
						age := getAge(pr.CreatedAt, pr.Reviews.Nodes[0].CreatedAt)

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
	ghs.rb.SetTeamName(ghs.cfg.GitHubTeam)

	res := ghs.rb.Emit()
	return ghs.mb.Emit(metadata.WithResource(res)), nil
}
