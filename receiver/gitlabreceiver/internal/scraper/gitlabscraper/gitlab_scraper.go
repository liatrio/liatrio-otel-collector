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
	gitlab "gitlab.com/gitlab-org/api/client-go"
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

	projectList, err := gls.getProjects(ctx, restClient)
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
		url := project.URL

		go func() {
			defer wg.Done()

			branches, err := gls.getBranchNames(ctx, graphClient, path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting branches for project '%s': %v", path, zap.Error(err))
				return
			}
			// Create a mutual exclusion lock to prevent the recordDataPoint
			// from having a nil pointer error passing in the SetStartTimestamp
			mux.Lock()
			refType := metadata.AttributeVcsRefHeadTypeBranch
			gls.mb.RecordVcsRefCountDataPoint(now, int64(len(branches.BranchNames)), url, path, refType)
			mux.Unlock()
			for _, branch := range branches.BranchNames {
				if branch == branches.RootRef {
					continue
				}

				commit, err := gls.getInitialCommit(ctx, restClient, path, branches.RootRef, branch)
				if err != nil {
					gls.logger.Sugar().Errorf("error getting initial commit for project '%s' and branch '%s': %v", path, branch, err)
				}

				if commit != nil {
					branchAge := time.Since(*commit.CreatedAt).Seconds()
					mux.Lock()
					gls.mb.RecordVcsRefTimeDataPoint(now, int64(branchAge), url, path, branch, refType)
					mux.Unlock()
				}
			}

			// Get both the merged and open merge requests for the repository
			mrs, err := gls.getCombinedMergeRequests(ctx, graphClient, path, gls.cfg.LimitMergeRequests)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting merge requests for project '%s': %v", path, zap.Error(err))
				return
			}

			// Get the number of contributors for the repository
			contributorCount, err := gls.getContributorCount(restClient, path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting contributor count for project '%s': %v", path, err)
				return
			}
			mux.Lock()
			gls.mb.RecordVcsContributorCountDataPoint(now, int64(contributorCount), url, path)
			// gls.mb.RecordVcsRepositoryContributorCountDataPoint(now, int64(contributorCount), path)

			for _, mr := range mrs {
				//nolint:lll
				gls.mb.RecordVcsRefLinesDeltaDataPoint(now, int64(mr.DiffStatsSummary.Additions), url, path, mr.SourceBranch, refType, metadata.AttributeVcsLineChangeTypeAdded)
				//nolint:lll
				gls.mb.RecordVcsRefLinesDeltaDataPoint(now, int64(mr.DiffStatsSummary.Deletions), url, path, mr.SourceBranch, refType, metadata.AttributeVcsLineChangeTypeRemoved)

				// Checks if the merge request has been merged. This is done with IsZero() which tells us if the
				// time is or isn't  January 1, year 1, 00:00:00 UTC, which is what null in graphql date values
				// get returned as in Go.
				if mr.MergedAt.IsZero() {
					mrAge := int64(time.Since(mr.CreatedAt).Seconds())
					gls.mb.RecordVcsChangeDurationDataPoint(now, mrAge, url, path, mr.SourceBranch, metadata.AttributeVcsChangeStateOpen)
				} else {
					mergedAge := int64(mr.MergedAt.Sub(mr.CreatedAt).Seconds())
					gls.mb.RecordVcsChangeTimeToMergeDataPoint(now, mergedAge, url, path, mr.SourceBranch)
				}
			}
			mux.Unlock()
		}()
	}

	wg.Wait()

	gls.logger.Sugar().Infof("Finished processing Gitlab org %s", gls.cfg.GitLabOrg)

	gls.rb.SetVcsVendorName("gitlab")
	gls.rb.SetOrganizationName(gls.cfg.GitLabOrg)

	res := gls.rb.Emit()
	return gls.mb.Emit(metadata.WithResource(res)), nil
}
