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

func (gls *gitlabScraper) getMergeRequestInfo(projectName string) {

	graphClient := graphql.NewClient("https://gitlab.com/api/graphql", gls.client)

	mergeRequests, err := getOpenedMergeRequests(context.Background(), graphClient, gls.cfg.GitLabOrg+"/"+projectName)

	requestMap := make(map[string]string)
	requestMap["projectName"] = projectName

	for _, m := range mergeRequests.Project.MergeRequests.Nodes {
		requestMap["mergeRequestTitle"] = m.Title
		requestMap["mergeRequestCreated"] = m.CreatedAt.String()
	}

	if err != nil {
		gls.logger.Sugar().Errorf("error getting merge request info: %v", err)
	}

	gls.logger.Sugar().Debugf("found merge request: %v", requestMap["mergeRequestTitle"]+", created: "+requestMap["mergeRequestCreated"])
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

	projects, err := getProjects(context.Background(), graphClient, gls.cfg.GitLabOrg)

	if err != nil {
		gls.logger.Sugar().Errorf("error getting projects: %v", err)
	}

	var projectList []string
	var branchList []string

	for _, project := range projects.Group.Projects.Nodes {

		gls.logger.Sugar().Debugf("project: %v", project.Name)

		projectList = append(projectList, project.Name)

		branches, err := getBranchNames(context.Background(), graphClient, gls.cfg.GitLabOrg+"/"+project.Name)

		if err != nil {
			gls.logger.Sugar().Errorf("error getting branches: %v", err)
		}

		branchList = append(branchList, branches.Project.Repository.BranchNames...)

		gls.logger.Sugar().Debugf("branch: %v", branches.Project.Repository.BranchNames)

		// set the metric with metric builder
		gls.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(len(branches.Project.Repository.BranchNames)), project.Name)
	}

	gls.logger.Sugar().Debugf("metrics: %v", gls.cfg.Metrics.GitRepositoryCount)

	return gls.mb.Emit(), nil
}
