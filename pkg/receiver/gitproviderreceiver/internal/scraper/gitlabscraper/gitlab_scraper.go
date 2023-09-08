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
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

type gitlabProject struct {
	Name           string
	Path           string
	CreatedAt      time.Time
	LastActivityAt time.Time
}

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

type projectData struct {
	ProjectPath string
	BranchCount int64
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

	ch <- projectData{ProjectPath: projectPath, BranchCount: int64(len(branches.Project.Repository.BranchNames))}
}

// Scrape the GitLab GraphQL API for the various metrics. took 9m56s to complete.
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

	ch := make(chan projectData)

	var wg sync.WaitGroup

	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for _, project := range projectList {
		wg.Add(1)

		// created shadowed declaration due to loop variable because Loop variables
		// captured by 'func' literals in 'go' statements might have unexpected values
		project := project
		go func() {
			gls.getBranches(ctx, graphClient, project.Path, ch, &wg)
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for proj := range ch {
		gls.mb.RecordGitRepositoryBranchCountDataPoint(now, proj.BranchCount, proj.ProjectPath)
		gls.logger.Sugar().Debugf("%s branch count: %v", proj.ProjectPath, proj.BranchCount)
	}

	// record repository count metric
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	return gls.mb.Emit(), nil
}
