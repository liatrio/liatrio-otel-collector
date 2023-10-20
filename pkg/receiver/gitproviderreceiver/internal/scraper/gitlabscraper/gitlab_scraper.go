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
	"sync"
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

type projectData struct {
	ProjectPath   string
	DefaultBranch string
	Branches      []string
}

// Returns a struct with the project path and an array of branch names via the given channel.
//
// The max number of branch names currently returned is 100,000 since the GitLab API will let us get
// as much as we want but it will take a long time to get all the branches. The biggest repo we
// found was a GitLab repo with just over 100,000 branches which is an extreme edge case which we
// believe is not worth supporting.
func (gls *gitlabScraper) getBranches(
	ctx context.Context,
	graphClient graphql.Client,
	projects []gitlabProject,
	ch chan projectData,
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	for _, project := range projects {
		branches, err := getBranchNames(ctx, graphClient, project.Path)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
			return
		}
		ch <- projectData{ProjectPath: project.Path, DefaultBranch: branches.Project.Repository.RootRef, Branches: branches.Project.Repository.BranchNames}
	}
}

func (gls *gitlabScraper) getMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projects []gitlabProject,
	state MergeRequestState,
	ch chan []MergeRequestNode,
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()
	for _, project := range projects {
		gls.logger.Sugar().Debugf("getting merge requests for project: %v", project.Path)
		var mergeRequestData []MergeRequestNode
		var mrCursor *string
		for hasNextPage := true; hasNextPage; {
			// Get the next page of data
			mr, err := getMergeRequests(ctx, graphClient, project.Path, mrCursor, state)
			if err != nil {
				gls.logger.Sugar().Errorf("error: %v", err)
			}
			if len(mr.Project.MergeRequests.Nodes) == 0 {
				break
			}
			// Check if there is a next page
			hasNextPage = mr.Project.MergeRequests.PageInfo.HasNextPage

			// Set the cursor to the end cursor of the current page
			mrCursor = &mr.Project.MergeRequests.PageInfo.EndCursor
			mergeRequestData = append(mergeRequestData, mr.Project.MergeRequests.Nodes...)
			gls.logger.Sugar().Debugf("project: %v has %v %s merge requests", project.Path, len(mr.Project.MergeRequests.Nodes), state)
		}
		if len(mergeRequestData) != 0 {
			ch <- mergeRequestData
		}
	}
}

func (gls *gitlabScraper) processMergeRequests(
	mergeCh chan []MergeRequestNode,
	now pcommon.Timestamp,
) {
	for mrs := range mergeCh {
		for _, mr := range mrs {
			gls.mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(now, int64(mr.DiffStatsSummary.Additions), mr.Project.FullPath, mr.SourceBranch)
			gls.mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(now, int64(mr.DiffStatsSummary.Deletions), mr.Project.FullPath, mr.SourceBranch)

			//IsZero() tells us if the time is or isnt  January 1, year 1, 00:00:00 UTC, which is what null graphql date values get returned as in Go
			if mr.MergedAt.IsZero() {
				mrAge := int64(time.Since(mr.CreatedAt).Hours())
				gls.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, mrAge, mr.Project.FullPath, mr.SourceBranch)
				gls.logger.Sugar().Debugf("%s merge request for branch %v, age: %v", mr.Project.FullPath, mr.SourceBranch, mrAge)
			} else {
				mergedAge := int64(mr.MergedAt.Sub(mr.CreatedAt).Hours())
				gls.mb.RecordGitRepositoryPullRequestMergeTimeDataPoint(now, mergedAge, mr.Project.FullPath, mr.SourceBranch)
				gls.logger.Sugar().Debugf("%s merge request for branch %v, merged age: %v", mr.Project.FullPath, mr.SourceBranch, mergedAge)
			}
		}
	}
}

func (gls *gitlabScraper) processBranches(
	restClient *gitlab.Client,
	branchCh chan projectData,
	now pcommon.Timestamp,
) {
	for proj := range branchCh {
		gls.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(proj.Branches)), proj.ProjectPath)
		gls.logger.Sugar().Debugf("%s branch count: %v", proj.ProjectPath, int64(len(proj.Branches)))

		for _, branch := range proj.Branches {
			if branch == proj.DefaultBranch {
				continue
			}

			diff, _, err := restClient.Repositories.Compare(proj.ProjectPath, &gitlab.CompareOptions{From: &proj.DefaultBranch, To: gitlab.String(branch)})
			if err != nil {
				gls.logger.Sugar().Errorf("error: %v", err)
			}

			if len(diff.Commits) != 0 {
				branchAge := time.Since(*diff.Commits[0].CreatedAt).Hours()
				gls.logger.Sugar().Debugf("%v age: %v hours, commit name: %s", branch, branchAge, diff.Commits[0].Title)
				gls.mb.RecordGitRepositoryBranchTimeDataPoint(now, int64(branchAge), proj.ProjectPath, branch)
			}
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

	var wg1 sync.WaitGroup

	var opBuf int = 5
	chunkSize := (len(projectList) + opBuf - 1) / opBuf
	var work [][]gitlabProject = chunkSlice(projectList, chunkSize)
	if opBuf > len(work) {
		opBuf = len(work)
	}

	branchCh := make(chan projectData, opBuf)
	mergeCh := make(chan []MergeRequestNode, opBuf)
	gls.logger.Sugar().Debugf("There are %v projects", len(projectList))
	gls.logger.Sugar().Debugf("There are %v workers", len(work))
	for i := 0; i < opBuf; i++ {
		gls.logger.Sugar().Debugf("worker %v has work of size %v", i, len(work[i]))
	}

	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for i := 0; i < opBuf; i++ {
		wg1.Add(3)
		i := i
		go gls.getBranches(ctx, graphClient, work[i], branchCh, &wg1)
		go gls.getMergeRequests(ctx, graphClient, work[i], MergeRequestStateOpened, mergeCh, &wg1)
		go gls.getMergeRequests(ctx, graphClient, work[i], MergeRequestStateMerged, mergeCh, &wg1)
	}

	for i := 0; i < opBuf; i++ {
		go gls.processBranches(restClient, branchCh, now)
		go gls.processMergeRequests(mergeCh, now)
	}

	wg1.Wait()
	close(branchCh)
	close(mergeCh)
	// record repository count metric
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	return gls.mb.Emit(), nil
}
