//go:generate ../../../../../.tools/genqlient

package gitlabscraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/xanzy/go-gitlab"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal/metadata"
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
	settings receiver.CreateSettings,
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

	var projectList []gitlabProject

	for nextPage := 1; nextPage > 0; {
		// TODO: since we pass in a context already, do we need to create a new background context?
		projects, res, err := restClient.Groups.ListGroupProjects(gls.cfg.GitLabOrg, &gitlab.ListGroupProjectsOptions{
			IncludeSubGroups: gitlab.Ptr(true),
			Topic:            gitlab.Ptr(gls.cfg.SearchTopic),
			Search:           gitlab.Ptr(gls.cfg.SearchQuery),
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 100,
			},
		})
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)

			return gls.mb.Emit(), err
		}

		if len(projects) == 0 {
			errMsg := fmt.Sprintf("no GitLab projects found for the given group/org: %s", gls.cfg.GitLabOrg)
			err = errors.New(errMsg)
			gls.logger.Sugar().Error(err)

			return gls.mb.Emit(), err
		}

		for _, p := range projects {
			projectList = append(projectList, gitlabProject{
				Name:           p.Name,
				Path:           p.PathWithNamespace,
				CreatedAt:      *p.CreatedAt,
				LastActivityAt: *p.LastActivityAt,
			})
		}

		nextPageHeader := res.Header.Get("x-next-page")
		if len(nextPageHeader) > 0 {
			nextPage, err = strconv.Atoi(nextPageHeader)
			if err != nil {
				gls.logger.Sugar().Errorf("error: %v", err)

				return gls.mb.Emit(), err
			}
		} else {
			nextPage = 0
		}
	}

	var maxProcesses = 3
	sem := make(chan int, maxProcesses)
	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for _, project := range projectList {
		sem <- 1
		go func(project gitlabProject) {
			branches, err := gls.getBranchNames(ctx, graphClient, project.Path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting branches: %v", zap.Error(err))
				<-sem
				return
			}
			gls.processBranches(restClient, branches, project.Path, now)
			<-sem
		}(project)
	}

	for _, project := range projectList {
		sem <- 1
		go func(project gitlabProject) {
			mrs, err := gls.getCombinedMergeRequests(ctx, graphClient, project.Path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting merge requests: %v", zap.Error(err))
				<-sem
				return
			}
			gls.processMergeRequests(mrs, project.Path, now)
			<-sem
		}(project)
	}

	for _, project := range projectList {
		sem <- 1
		go func(project gitlabProject) {
			contributorCount, err := gls.getContributorCount(restClient, project.Path)
			if err != nil {
				gls.logger.Sugar().Errorf("error: %v", err)
				<-sem
				return
			}
			gls.logger.Sugar().Debugf("contributor count: %v for repo %v", contributorCount, project.Path)
			gls.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contributorCount), project.Path)
			<-sem
		}(project)
	}
	// wait until all goroutines are finished
	for i := 0; i < maxProcesses; i++ {
		sem <- 1
	}

	// record repository count metric
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	gls.rb.SetGitVendorName("gitlab")
	gls.rb.SetOrganizationName(gls.cfg.GitLabOrg)

	res := gls.rb.Emit()
	return gls.mb.Emit(metadata.WithResource(res)), nil
}
