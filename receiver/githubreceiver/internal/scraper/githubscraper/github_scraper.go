// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate ../../../../../.tools/genqlient

package githubscraper // import "github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/scraper/githubscraper"

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

	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"
)

// const defaultConcurrencyLimit = 5

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
	settings receiver.Settings,
	cfg *Config,
) *githubScraper {
	return &githubScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		rb:       metadata.NewResourceBuilder(cfg.ResourceAttributes),
	}
}

// scrape and return github metrics
func (ghs *githubScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	if ghs.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	ghs.logger.Sugar().Debug("current time", zap.Time("now", now.AsTime()))

	genClient, restClient, err := ghs.createClients()
	if err != nil {
		ghs.logger.Sugar().Errorf("unable to create clients", zap.Error(err))
	}

	// Do some basic validation to ensure the values provided actually exist in github
	// prior to making queries against that org or user value
	loginType, err := ghs.login(ctx, genClient, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("error logging into GitHub via GraphQL", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	// Generate the search query based on the type, org/user name, and the search_query
	// value if provided
	sq := genDefaultSearchQuery(loginType, ghs.cfg.GitHubOrg)

	if ghs.cfg.SearchQuery != "" {
		sq = ghs.cfg.SearchQuery
		ghs.logger.Sugar().Debugf("using search query where query is: %q", ghs.cfg.SearchQuery)
	}

	// Get the repository data based on the search query retrieving a slice of branches
	// and the recording the total count of repositories
	repos, count, err := ghs.getRepos(ctx, genClient, sq)
	if err != nil {
		ghs.logger.Sugar().Errorf("error getting repo data", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	ghs.mb.RecordVcsRepositoryCountDataPoint(now, int64(count))

	// Get the ref (branch) count (future branch data) for each repo and record
	// the given metrics
	var wg sync.WaitGroup
	wg.Add(len(repos))
	var mux sync.Mutex

	// TODO: Cleanup
	// max := defaultConcurrencyLimit
	// if ghs.cfg.ConcurrencyLimit >= 0 {
	// 	max = ghs.cfg.ConcurrencyLimit
	// }
	var max int
	switch {
	case ghs.cfg.ConcurrencyLimit > 0:
		max = ghs.cfg.ConcurrencyLimit
	default:
		max = len(repos)
	}

	limiter := make(chan struct{}, max)

	// TODO: cleanup
	// Creating and filling a dummy structure to limit the amount of concurrent
	// requests to GitHub to avoid secondary rate limits.
	// for range max {
	// 	limiter <- struct{}{}
	// }

	// done := make(chan struct{})
	// go func() {
	// 	wg.Wait()
	// 	close(done)
	// }()
	// select {
	// case <-done:
	// case <-ctx.Done():
	// }

	// waitForAll := make(chan bool)

	for _, repo := range repos {
		repo := repo
		name := repo.Name
		url := repo.Url
		trunk := repo.DefaultBranchRef.Name
		now := now

		limiter <- struct{}{}

		go func() {
			defer func() {
				<-limiter
				wg.Done()
			}()
			// TODO: cleanup
			// defer wg.Done()

			// limiter <- struct{}{}
			branches, count, err := ghs.getBranches(ctx, genClient, name, trunk)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting branch count: %v", zap.Error(err))
			}

			// Create a mutual exclusion lock to prevent the recordDataPoint
			// SetStartTimestamp call from having a nil pointer panic
			mux.Lock()

			refType := metadata.AttributeVcsRefHeadTypeBranch
			ghs.mb.RecordVcsRefCountDataPoint(now, int64(count), url, name, refType)

			// Iterate through the refs (branches) populating the Branch focused
			// metrics
			for _, branch := range branches {
				// See https://github.com/liatrio/liatrio-otel-collector/blob/main/receiver/githubreceiver/internal/scraper/githubscraper/README.md#github-limitations
				// for more information as to why we do not emit metrics for
				// the default branch (trunk) nor any branch with no changes to
				// it.
				if branch.Name == branch.Repository.DefaultBranchRef.Name || branch.Compare.BehindBy == 0 {
					continue
				}

				// See https://github.com/liatrio/liatrio-otel-collector/blob/main/receiver/githubreceiver/internal/scraper/githubscraper/README.md#github-limitations
				// for more information as to why `BehindBy` and `AheadBy` are
				// swapped.
				//nolint:lll
				ghs.mb.RecordVcsRefRevisionsDeltaDataPoint(now, int64(branch.Compare.BehindBy), url, name, branch.Name, refType, metadata.AttributeVcsRevisionDeltaDirectionAhead)
				//nolint:lll
				ghs.mb.RecordVcsRefRevisionsDeltaDataPoint(now, int64(branch.Compare.AheadBy), url, name, branch.Name, refType, metadata.AttributeVcsRevisionDeltaDirectionBehind)

				var additions int
				var deletions int
				var age int64

				additions, deletions, age, err = ghs.evalCommits(ctx, genClient, branch.Repository.Name, branch)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting commit info: %v", zap.Error(err))
					continue
				}

				ghs.mb.RecordVcsRefTimeDataPoint(now, age, url, name, branch.Name, refType)
				ghs.mb.RecordVcsRefLinesDeltaDataPoint(now, int64(additions), url, name, branch.Name, refType, metadata.AttributeVcsLineChangeTypeAdded)
				ghs.mb.RecordVcsRefLinesDeltaDataPoint(now, int64(deletions), url, name, branch.Name, refType, metadata.AttributeVcsLineChangeTypeRemoved)

			}

			// Get the contributor count for each of the repositories
			// if ghs.cfg.Metrics.VcsContributorCount.Enabled {
			// }
			contribs, err := ghs.getContributorCount(ctx, restClient, name)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting contributor count: %v", zap.Error(err))
			}
			ghs.mb.RecordVcsContributorCountDataPoint(now, int64(contribs), url, name)

			// Get change (pull request) data
			prs, err := ghs.getPullRequests(ctx, genClient, name)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting pull requests: %v", zap.Error(err))
			}

			// When enabled, process any CVEs for the repository
			if ghs.cfg.Metrics.VcsCveCount.Enabled {
				cves, err := ghs.getCVEs(ctx, genClient, restClient, name)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting cves: %v", zap.Error(err))
				}
				for s, c := range cves {
					ghs.mb.RecordVcsCveCountDataPoint(now, c, url, name, s)
				}
			}

			var merged int
			var open int

			for _, pr := range prs {
				if pr.Merged {
					merged++

					age := getAge(pr.CreatedAt, pr.MergedAt)

					ghs.mb.RecordVcsChangeTimeToMergeDataPoint(now, age, url, name, pr.HeadRefName)

				} else {
					open++

					age := getAge(pr.CreatedAt, now.AsTime())

					ghs.mb.RecordVcsChangeDurationDataPoint(now, age, url, name, pr.HeadRefName, metadata.AttributeVcsChangeStateOpen)

					if pr.Reviews.TotalCount > 0 {
						age := getAge(pr.CreatedAt, pr.Reviews.Nodes[0].CreatedAt)

						ghs.mb.RecordVcsChangeTimeToApprovalDataPoint(now, age, url, name, pr.HeadRefName)
					}
				}
			}

			ghs.mb.RecordVcsChangeCountDataPoint(now, int64(open), url, metadata.AttributeVcsChangeStateOpen, name)
			ghs.mb.RecordVcsChangeCountDataPoint(now, int64(merged), url, metadata.AttributeVcsChangeStateMerged, name)
			mux.Unlock()
			// TODO: cleanup
			// <-limiter
		}()
	}

	wg.Wait()

	// Set the resource attributes and emit metrics with those resources
	ghs.rb.SetVcsVendorName("github")
	ghs.rb.SetOrganizationName(ghs.cfg.GitHubOrg)
	ghs.rb.SetTeamName(ghs.cfg.GitHubTeam)

	res := ghs.rb.Emit()
	return ghs.mb.Emit(metadata.WithResource(res)), nil
}
