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

var (
	errClientNotInitErr = errors.New("http client not initialized")
)

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

	subgroups, err := getGroupDescendents(context.Background(), graphClient, gls.cfg.GitLabOrg)

	var projectList []string

	if len(subgroups.Group.DescendantGroups.Nodes) > 0 {
		for _, d := range subgroups.Group.DescendantGroups.Nodes {
			projectList = append(projectList, d.Name)
		}
	}

	// TODO: this only works for top level groups with 1 level of subgroups
	for _, group := range subgroups.Group.DescendantGroups.Nodes {
		// full path should be a function that takes in a list of strings and returns a string
		// concatenated together. This makes it testable & works with multiple subgroups
		projects, err := getGroupProjects(context.Background(), graphClient, gls.cfg.GitLabOrg+"/"+group.Name)
		if err != nil {
			gls.logger.Sugar().Errorf("error getting descendant groups: %v", err)
		}
		for _, project := range projects.Group.Projects.Nodes {
			projectList = append(projectList, project.Name)
		}
	}

	// TODO: Must paginate and update query to get all branches of projects in the list
	//for _, project := range projectList {

	//	gls.logger.Sugar().Debugf("project: %v", project)

	//	branches, err := getBranchNames(context.Background(), graphClient, project)

	//	if err != nil {
	//		gls.logger.Sugar().Errorf("error getting branches: %v", err)
	//	}

	//	var branchList []string
	//	branchList = append(branchList, branches.Project.Repository.BranchNames...)

	//	gls.logger.Sugar().Debugf("branch: %v", branches.Project.Repository.BranchNames)

	//	gls.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(branchList)), project)

	//}

	if err != nil {
		gls.logger.Sugar().Errorf("error: %v", err)
	}

	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))

	gls.logger.Sugar().Debugf("metrics: %v", gls.cfg.Metrics.GitRepositoryCount)

	return gls.mb.Emit(), nil
}
