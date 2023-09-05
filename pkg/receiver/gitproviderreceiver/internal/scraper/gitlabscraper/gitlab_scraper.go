// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"net/http"
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

// scrape and return metrics
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

	// A map that maps a project name to an array of branch names.
	var branchNames map[string][]string = make(map[string][]string, len(projectList))

	// TODO: Must account for when there are more than 100,000 branch names in a project.
	for _, project := range projectList {
		branches, err := getBranchNames(context.Background(), graphClient, project.Path)
		if err != nil {
			gls.logger.Sugar().Errorf("error: %v", err)
		}

		branchCount := int64(len(branches.Project.Repository.BranchNames))

		branchNames[project.Name] = branches.Project.Repository.BranchNames
		gls.mb.RecordGitRepositoryBranchCountDataPoint(now, branchCount, project.Name)
		gls.logger.Sugar().Debug("branch count: ", branchCount)
	}

	// record metrics
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))
	gls.logger.Sugar().Debugf("metrics: %v", gls.cfg.Metrics.GitRepositoryCount)

	return gls.mb.Emit(), nil
}
