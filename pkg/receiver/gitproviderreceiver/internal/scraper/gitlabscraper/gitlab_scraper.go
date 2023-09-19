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
	projectPath string,
	ch chan projectData,
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	branches, err := getBranchNames(ctx, graphClient, projectPath)
	if err != nil {
		gls.logger.Sugar().Errorf("error: %v", err)
		return
	}

	ch <- projectData{ProjectPath: projectPath, Branches: branches.Project.Repository.BranchNames}
}

func (gls *gitlabScraper) getOpenedMergeRequests(
	ctx context.Context,
	graphClient graphql.Client,
	projectPath string,
	ch chan []MergeRequestNode,
	waitGroup *sync.WaitGroup,
) {
	var mergeRequestData []MergeRequestNode
	var mrCursor *string

	defer waitGroup.Done()

	for hasNextPage := true; hasNextPage; {
		// Get the next page of data
		mr, err := getOpenedMergeRequests(ctx, graphClient, projectPath, mrCursor)
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
	}

	ch <- mergeRequestData
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
	var wg2 sync.WaitGroup

	branchCh := make(chan projectData)

	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for _, project := range projectList {
		wg1.Add(1)

		go func(project gitlabProject) {
			gls.getBranches(ctx, graphClient, project.Path, branchCh, &wg1)
		}(project)
	}

	go func() {
		wg1.Wait()
		close(branchCh)
	}()

	mergeCh := make(chan []MergeRequestNode)

	for _, project := range projectList {
		wg2.Add(1)

		go func(project gitlabProject) {
			gls.getOpenedMergeRequests(ctx, graphClient, project.Path, mergeCh, &wg2)
		}(project)
	}

	go func() {
		wg2.Wait()
		close(mergeCh)
	}()

	for proj := range ch {
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

	// Handling the merge request data
	for mrs := range mergeCh {
		if len(mrs) != 0 {
			for _, mr := range mrs {
				mrAge := int64(time.Since(mr.CreatedAt).Hours())
				gls.mb.RecordGitRepositoryPullRequestTimeDataPoint(now, mrAge, mr.Project.FullPath, mr.TargetBranch)
				gls.logger.Sugar().Debugf("%s merge request for branch %v, age: %v", mr.Project.FullPath, mr.TargetBranch, mrAge)
			}
		}
	}

	// record repository count metric
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	return gls.mb.Emit(), nil
}
