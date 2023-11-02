// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

// A struct representing the GitLab Scraper.
type gitlabScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
}

// A struct representing a GitLab project.
type gitlabProject struct {
	// The name of the project.
	Name string

	// The full path to the project.
	Path string

	// When it was created.
	CreatedAt time.Time

	// When it was last active.
	LastActivityAt time.Time
}

func (gls *gitlabScraper) start(_ context.Context, host component.Host) (err error) {
	gls.logger.Sugar().Info("Starting the scraper inside scraper.go")
	// TODO: Fix the ToClient configuration
	gls.client, err = gls.cfg.ToClient(host, gls.settings)
	return
}

// Create a new GitLab Scraper.
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
	}
}

// Iterate through the list of branches and record the relevant metrics.
func (gls *gitlabScraper) processBranches(
	client *gitlab.Client,
	branches *getBranchNamesProjectRepository,
	projectPath string,
	now pcommon.Timestamp,
) {
	gls.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(branches.BranchNames)), projectPath)
	gls.logger.Sugar().Debugf("%s branch count: %v", projectPath, int64(len(branches.BranchNames)))

	for _, branch := range branches.BranchNames {
		if branch == branches.RootRef {
			continue
		}

		initialCommit, err := gls.getInitialCommit(client, projectPath, branches.RootRef, branch)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		if initialCommit != nil {
			branchAge := time.Since(*initialCommit.CreatedAt).Hours()
			gls.logger.Sugar().Debugf("%v age: %v hours, commit name: %s", branch, branchAge, initialCommit.Title)
			gls.mb.RecordGitRepositoryBranchTimeDataPoint(now, int64(branchAge), projectPath, branch)
		}
	}
}

// Get merge request data for a given project that are in a given state.
func (gls *gitlabScraper) getMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
	state MergeRequestState,
) ([]MergeRequestNode, error) {
	var mergeRequestData []MergeRequestNode
	var mergeRequestCursor *string

	for hasNextPage := true; hasNextPage; {
		// Get the next page of data
		mergeRequest, err := getMergeRequests(ctx, graphClient, projectPath, mergeRequestCursor, state)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
			return nil, err
		}

		if len(mergeRequest.Project.MergeRequests.Nodes) == 0 {
			break
		}

		hasNextPage = mergeRequest.Project.MergeRequests.PageInfo.HasNextPage
		mergeRequestCursor = &mergeRequest.Project.MergeRequests.PageInfo.EndCursor
		mergeRequestData = append(mergeRequestData, mergeRequest.Project.MergeRequests.Nodes...)
	}

	return mergeRequestData, nil
}

// Get merge request data for a given project in the opened or merged state.
func (gls *gitlabScraper) getCombinedMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
) ([]MergeRequestNode, error) {
	openMergeRequests, err := gls.getMergeRequests(ctx, graphClient, projectPath, MergeRequestStateOpened)
	if err != nil {
		gls.logger.Sugar().Errorf("error getting open merge requests", zap.Error(err))
		return nil, err
	}

	mergedMergeRequests, err := gls.getMergeRequests(ctx, graphClient, projectPath, MergeRequestStateMerged)
	if err != nil {
		gls.logger.Sugar().Errorf("error getting merged merge requests", zap.Error(err))
		return nil, err
	}

	// Combine the open and merged merge requests and return the result.
	return append(openMergeRequests, mergedMergeRequests...), nil
}

func (gls *gitlabScraper) processMergeRequests(
	client *gitlab.Client,
	mergeRequests []MergeRequestNode,
	projectPath string,
	now pcommon.Timestamp,
) {
	for _, mergeRequest := range mergeRequests {
		gls.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(mergeRequest.DiffStatsSummary.Additions), projectPath, mergeRequest.SourceBranch)
		gls.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(mergeRequest.DiffStatsSummary.Deletions), projectPath, mergeRequest.SourceBranch)

		// IsZero() tells us if the time is or isn't "January 1, year 1, 00:00:00 UTC", which is what
		// null graphql date values get returned as in Go.
		if mergeRequest.MergedAt.IsZero() {
			mergeRequestAge := int64(time.Since(mergeRequest.CreatedAt).Hours())
			gls.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, mergeRequestAge, projectPath, mergeRequest.SourceBranch)
			gls.logger.Sugar().Debugf("%s merge request for branch %v, age: %v", projectPath, mergeRequest.SourceBranch, mergeRequestAge)
		} else {
			mergeRequestAge := int64(mergeRequest.MergedAt.Sub(mergeRequest.CreatedAt).Hours())
			gls.mb.RecordGitRepositoryPullRequestMergeTimeDataPoint(now, mergeRequestAge, projectPath, mergeRequest.SourceBranch)
			gls.logger.Sugar().Debugf("%s merge request for branch %v, merged age: %v", projectPath, mergeRequest.SourceBranch, mergeRequestAge)
		}
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

	gls.logger.Sugar().Debug("creating a new GitLab client")

	// Enable the ability to override the endpoint for self-hosted GitLab instances.
	graphClientURL := "https://gitlab.com/api/graphql"
	restClientURL := "https://gitlab.com/"

	if gls.cfg.HTTPClientSettings.Endpoint != "" {
		var err error

		graphClientURL, err = url.JoinPath(gls.cfg.HTTPClientSettings.Endpoint, "api/graphql")
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		restClientURL, err = url.JoinPath(gls.cfg.HTTPClientSettings.Endpoint, "/")
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}
	}

	graphClient := graphql.NewClient(graphClientURL, gls.client)
	restClient, err := gitlab.NewClient("", gitlab.WithHTTPClient(gls.client), gitlab.WithBaseURL(restClientURL))
	if err != nil {
		gls.logger.Sugar().Errorf("error: %v", err)
	}

	var projectList []gitlabProject

	for nextPage := 1; nextPage > 0; {
		projects, res, err := restClient.Groups.ListGroupProjects(gls.cfg.GitLabOrg, &gitlab.ListGroupProjectsOptions{
			IncludeSubGroups: gitlab.Bool(true),
			Topic:            gitlab.String(gls.cfg.SearchTopic),
			Search:           gitlab.String(gls.cfg.SearchQuery),
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

		for _, project := range projects {
			projectList = append(projectList, gitlabProject{
				Name:           project.Name,
				Path:           project.PathWithNamespace,
				CreatedAt:      *project.CreatedAt,
				LastActivityAt: *project.LastActivityAt,
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

	var maxProcesses int = 3
	sem := make(chan int, maxProcesses)

	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for _, project := range projectList {
		sem <- 1
		go func(project gitlabProject) {
			branches, err := gls.getBranchNames(ctx, graphClient, project.Path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting branches", zap.Error(err))
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
			mergeRequests, err := gls.getCombinedMergeRequests(ctx, graphClient, project.Path)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting merge requests", zap.Error(err))
				<-sem
				return
			}
			gls.processMergeRequests(restClient, mergeRequests, project.Path, now)
			<-sem
		}(project)
	}

	// wait until all goroutines are finished
	for i := 0; i < maxProcesses; i++ {
		sem <- 1
	}

	// record repository count metric
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	return gls.mb.Emit(), nil
}
