// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"net/http"
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
	ProjectPath string
	Branches    []string
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
		ch <- projectData{ProjectPath: project.Path, Branches: branches.Project.Repository.BranchNames}
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

func chunkSlice[T any](slice []T, chunkSize int) [][]T {
	var chunks [][]T

	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
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

	graphClient := graphql.NewClient("https://gitlab.com/api/graphql", gls.client)
	restClient, err := gitlab.NewClient("", gitlab.WithHTTPClient(gls.client))
	if err != nil {
		gls.logger.Sugar().Errorf("error: %v", err)
	}

	var projectList []gitlabProject
	var projectsCursor *string

	for hasNextPage := true; hasNextPage; {
		// TODO: since we pass in a context already, do we need to create a new background context?
		// Get the next page of data
		projects, err := getAllGroupProjects(context.Background(), graphClient, gls.cfg.GitLabOrg, projectsCursor)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		if len(projects.Group.Projects.Nodes) == 0 {
			err = errors.New("no GitLab projects found for the given group/org")
			gls.logger.Sugar().Error(err)

			return gls.mb.Emit(), err
		}

		// Check if there is a next page
		hasNextPage = projects.Group.Projects.PageInfo.HasNextPage

		// Set the cursor to the end cursor of the current page
		projectsCursor = &projects.Group.Projects.PageInfo.EndCursor

		for _, p := range projects.Group.Projects.Nodes {
			projectList = append(projectList, gitlabProject{
				Name:           p.Name,
				Path:           p.FullPath,
				CreatedAt:      p.CreatedAt,
				LastActivityAt: p.LastActivityAt,
			})
		}
	}

	var wg1 sync.WaitGroup

	var workers int = 1
	chunkSize := (len(projectList) + workers - 1) / workers
	var work [][]gitlabProject = chunkSlice(projectList, chunkSize)
	branchCh := make(chan projectData, workers)
	mergeCh := make(chan []MergeRequestNode, workers)
	gls.logger.Sugar().Debugf("There are %v projects", len(projectList))
	for i := 0; i < workers; i++ {
		gls.logger.Sugar().Debugf("worker %v has work of size %v", i, len(work[i]))
	}
	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for i := 0; i < workers; i++ {
		wg1.Add(3)
		i := i
		go gls.getBranches(ctx, graphClient, work[i], branchCh, &wg1)
		go gls.getMergeRequests(ctx, graphClient, work[i], MergeRequestStateOpened, mergeCh, &wg1)
		go gls.getMergeRequests(ctx, graphClient, work[i], MergeRequestStateMerged, mergeCh, &wg1)
	}

	//Handling the branch data
	go func() {
		for proj := range branchCh {
			gls.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(proj.Branches)), proj.ProjectPath)
			gls.logger.Sugar().Debugf("%s branch count: %v", proj.ProjectPath, int64(len(proj.Branches)))

			for _, branch := range proj.Branches {
				if branch == "main" {
					continue
				}

				diff, _, err := restClient.Repositories.Compare(proj.ProjectPath, &gitlab.CompareOptions{From: gitlab.String("main"), To: gitlab.String(branch)})
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
		close(branchCh)
	}()

	//Handling the merge request data
	go func() {
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
		close(mergeCh)
	}()
	wg1.Wait()
	// record repository count metric
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	return gls.mb.Emit(), nil
}
