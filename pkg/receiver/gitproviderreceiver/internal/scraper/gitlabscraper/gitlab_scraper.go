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

type gitlabScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
}

func (gls *gitlabScraper) start(_ context.Context, host component.Host) (err error) {
	gls.logger.Sugar().Info("Starting the scraper inside scraper.go")
	// TODO: Fix the ToClient configuration
	gls.client, err = gls.cfg.ToClient(host, gls.settings)
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
	}
}

type gitlabProject struct {
	Name           string
	Path           string
	CreatedAt      time.Time
	LastActivityAt time.Time
}

func (gls *gitlabScraper) processBranches(client *gitlab.Client, branches *getBranchNamesProjectRepository, projectPath string, now pcommon.Timestamp) {
	gls.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(branches.BranchNames)), projectPath)
	gls.logger.Sugar().Debugf("%s branch count: %v", projectPath, int64(len(branches.BranchNames)))

	for _, branch := range branches.BranchNames {
		if branch == branches.RootRef {
			continue
		}

		commit, err := gls.getInitialCommit(client, projectPath, branches.RootRef, branch)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		if commit != nil {
			branchAge := time.Since(*commit.CreatedAt).Hours()
			gls.logger.Sugar().Debugf("%v age: %v hours, commit name: %s", branch, branchAge, commit.Title)
			gls.mb.RecordGitRepositoryBranchTimeDataPoint(now, int64(branchAge), projectPath, branch)
		}
	}
}

func (gls *gitlabScraper) getContributorCount(
	restClient *gitlab.Client,
	projectPath string,
) (int, error) {
	contribs, _, err := restClient.Repositories.Contributors(projectPath, nil)
	if err != nil {
		gls.logger.Sugar().Errorf("error getting contributors", zap.Error(err))
		return 0, err
	}
	contribCount := 0
	if len(contribs) > 0 {
		contribCount = len(contribs)
	}
	return contribCount, nil
}

func (gls *gitlabScraper) getMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
	state MergeRequestState,
) ([]MergeRequestNode, error) {
	var mergeRequestData []MergeRequestNode
	var mrCursor *string

	for hasNextPage := true; hasNextPage; {
		// Get the next page of data
		mr, err := getMergeRequests(ctx, graphClient, projectPath, mrCursor, state)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
			return nil, err
		}
		if len(mr.Project.MergeRequests.Nodes) == 0 {
			break
		}

		mrCursor = &mr.Project.MergeRequests.PageInfo.EndCursor
		hasNextPage = mr.Project.MergeRequests.PageInfo.HasNextPage
		mergeRequestData = append(mergeRequestData, mr.Project.MergeRequests.Nodes...)
	}

	return mergeRequestData, nil
}

func (gls *gitlabScraper) processMergeRequests(client *gitlab.Client, mrs []MergeRequestNode, projectPath string, now pcommon.Timestamp) {
	for _, mr := range mrs {
		gls.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(mr.DiffStatsSummary.Additions), projectPath, mr.SourceBranch)
		gls.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(mr.DiffStatsSummary.Deletions), projectPath, mr.SourceBranch)

		// IsZero() tells us if the time is or isnt  January 1, year 1, 00:00:00 UTC, which is what null graphql date values get returned as in Go
		if mr.MergedAt.IsZero() {
			mrAge := int64(time.Since(mr.CreatedAt).Hours())
			gls.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, mrAge, projectPath, mr.SourceBranch)
			gls.logger.Sugar().Debugf("%s merge request for branch %v, age: %v", projectPath, mr.SourceBranch, mrAge)
		} else {
			mergedAge := int64(mr.MergedAt.Sub(mr.CreatedAt).Hours())
			gls.mb.RecordGitRepositoryPullRequestMergeTimeDataPoint(now, mergedAge, projectPath, mr.SourceBranch)
			gls.logger.Sugar().Debugf("%s merge request for branch %v, merged age: %v", projectPath, mr.SourceBranch, mergedAge)
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

	gls.logger.Sugar().Debug("creating a new gitlab client")

	// Enable the ability to override the endpoint for self-hosted gitlab instances
	graphCURL := "https://gitlab.com/api/graphql"
	restCURL := "https://gitlab.com/"

	if gls.cfg.HTTPClientSettings.Endpoint != "" {
		var err error

		graphCURL, err = url.JoinPath(gls.cfg.HTTPClientSettings.Endpoint, "api/graphql")
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		restCURL, err = url.JoinPath(gls.cfg.HTTPClientSettings.Endpoint, "/")
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
			}
			if branches != nil {
				gls.processBranches(restClient, branches, project.Path, now)
			}
			<-sem
		}(project)
	}

	for _, project := range projectList {
		sem <- 1
		go func(project gitlabProject) {
			openMrs, err := gls.getMergeRequests(ctx, graphClient, project.Path, MergeRequestStateOpened)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting open merge requests", zap.Error(err))
				<-sem
			}
			mergedMrs, err := gls.getMergeRequests(ctx, graphClient, project.Path, MergeRequestStateMerged)
			if err != nil {
				gls.logger.Sugar().Errorf("error getting merged merge requests", zap.Error(err))
				<-sem
			}
			if openMrs != nil || mergedMrs != nil {
				mrs := append(openMrs, mergedMrs...)
				gls.processMergeRequests(restClient, mrs, project.Path, now)
			}
			<-sem
		}(project)
	}

	for _, project := range projectList {
		sem <- 1
		go func(project gitlabProject) {
			contribCount, err := gls.getContributorCount(restClient, project.Path)
			if err != nil {
				gls.logger.Sugar().Errorf("error: %v", err)
				<-sem
				return
			}
			gls.logger.Sugar().Debugf("contributor count: %v for repo %v", contribCount, project.Path)
			gls.mb.RecordGitRepositoryContributorCountDataPoint(now, int64(contribCount), project.Path)
			<-sem
		}(project)
	}
	//wait until all goroutines are finished
	for i := 0; i < maxProcesses; i++ {
		sem <- 1
	}

	// record repository count metric
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	return gls.mb.Emit(), nil
}
