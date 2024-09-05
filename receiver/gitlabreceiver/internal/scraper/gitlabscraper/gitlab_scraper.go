//go:generate ../../../../../.tools/genqlient

package gitlabscraper

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/xanzy/go-gitlab"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

type gitlabScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
	rb       *metadata.ResourceBuilder
}

func (gls *gitlabScraper) start(ctx context.Context, host component.Host) (err error) {
	gls.logger.Sugar().Info("Starting the scraper inside scraper.go")
	// TODO: Fix the ToClient configuration
	gls.client, err = gls.cfg.ToClient(ctx, host, gls.settings)
	return
}

func newGitLabScraper(
	_ context.Context,
	settings receiver.Settings,
	cfg *Config,
) *gitlabScraper {
	return &gitlabScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		rb:       metadata.NewResourceBuilder(cfg.ResourceAttributes),
	}
}

// Scrape the GitLab GraphQL API for the various metrics.
func (gls *gitlabScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	gls.logger.Sugar().Debug("checking if client is initialized")
	if gls.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	gls.logger.Sugar().Debugf("current time: %v", now)

	currentDate := time.Now().Day()

	gls.logger.Sugar().Debugf("current date: %v", currentDate)

	gls.logger.Sugar().Debug("creating a new gitlab client")

	// Enable the ability to override the endpoint for self-hosted gitlab instances
	graphCURL := "https://gitlab.com/api/graphql"
	restCURL := "https://gitlab.com/"

	if gls.cfg.ClientConfig.Endpoint != "" {
		var err error

		graphCURL, err = url.JoinPath(gls.cfg.ClientConfig.Endpoint, "api/graphql")
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		restCURL, err = url.JoinPath(gls.cfg.ClientConfig.Endpoint, "/")
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}
	}

	graphClient := graphql.NewClient(graphCURL, gls.client)
	restClient, err := gitlab.NewClient("", gitlab.WithHTTPClient(gls.client), gitlab.WithBaseURL(restCURL))
	if err != nil {
		gls.logger.Sugar().Errorf("error: %v", err)
	}

	projectList, err := gls.getProjects(restClient)
	if err != nil {
		gls.logger.Sugar().Errorf("error: %v", err)
		return gls.mb.Emit(), err
	}
	// record repository count metric
	gls.mb.RecordVcsRepositoryCountDataPoint(now, int64(len(projectList)))

	var wg sync.WaitGroup
	wg.Add(len(projectList))
	var mux sync.Mutex

	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for _, project := range projectList {
		project := project
		path := project.Path
		now := now
		go func() {
			defer wg.Done()

			branches, err := gls.getBranchNames(ctx, graphClient, path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting branches: %v", zap.Error(err))
				return
			}
			// Create a mutual exclusion lock to prevent the recordDataPoint
			// from having a nil pointer error passing in the SetStartTimestamp
			mux.Lock()
			gls.mb.RecordVcsRepositoryRefCountDataPoint(now, int64(len(branches.BranchNames)), path)

			for _, branch := range branches.BranchNames {
				if branch == branches.RootRef {
					continue
				}

				commit, err := gls.getInitialCommit(restClient, path, branches.RootRef, branch)
				if err != nil {
					gls.logger.Sugar().Errorf("error: %v", err)
				}

				if commit != nil {
					branchAge := time.Since(*commit.CreatedAt).Seconds()
					gls.mb.RecordVcsRepositoryRefTimeDataPoint(now, int64(branchAge), path, branch)
				}
			}

			// Get both the merged and open merge requests for the repository
			mrs, err := gls.getCombinedMergeRequests(ctx, graphClient, path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting merge requests: %v", zap.Error(err))
				return
			}

			// Get the number of contributors for the repository
			contributorCount, err := gls.getContributorCount(restClient, path)
			if err != nil {
				gls.logger.Sugar().Errorf("error: %v", err)
				return
			}
			gls.mb.RecordVcsRepositoryContributorCountDataPoint(now, int64(contributorCount), path)

			for _, mr := range mrs {
				gls.mb.RecordVcsRepositoryRefLineAdditionCountDataPoint(now, int64(mr.DiffStatsSummary.Additions), path, mr.SourceBranch)
				gls.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(mr.DiffStatsSummary.Deletions), path, mr.SourceBranch)

				// Checks if the merge request has been merged. This is done with IsZero() which tells us if the
				// time is or isn't  January 1, year 1, 00:00:00 UTC, which is what null in graphql date values
				// get returned as in Go.
				if mr.MergedAt.IsZero() {
					mrAge := int64(time.Since(mr.CreatedAt).Seconds())
					gls.mb.RecordGitRepositoryPullRequestTimeOpenDataPoint(now, mrAge, path, mr.SourceBranch)
				} else {
					mergedAge := int64(mr.MergedAt.Sub(mr.CreatedAt).Seconds())
					gls.mb.RecordGitRepositoryPullRequestTimeToMergeDataPoint(now, mergedAge, path, mr.SourceBranch)
				}
			}

			mux.Unlock()
		}()
	}

	wg.Wait()

	gls.rb.SetGitVendorName("gitlab")
	gls.rb.SetOrganizationName(gls.cfg.GitLabOrg)

	res := gls.rb.Emit()
	return gls.mb.Emit(metadata.WithResource(res)), nil
}
