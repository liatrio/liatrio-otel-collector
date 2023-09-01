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

type gitlabProject struct {
	Name           string
	Path           string
	CreatedAt      string
	LastActivityAt string
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

	projects, err := getAllGroupProjects(context.Background(), graphClient, gls.cfg.GitLabOrg)

	// get all projects for group/org
	var projectList []gitlabProject

	if len(projects.Group.Projects.Nodes) > 0 {
		for _, p := range projects.Group.Projects.Nodes {
			projectList = append(projectList, gitlabProject{Name: p.Name, Path: p.FullPath, CreatedAt: p.CreatedAt.String(), LastActivityAt: p.LastActivityAt.String()})
		}
	}

	// TODO: Must paginate and update query to get all branches of projects in the list

	// log error
	if err != nil {
		gls.logger.Sugar().Errorf("error: %v", err)
	}

	// record metrics
	gls.mb.RecordGitRepositoryCountDataPoint(now, int64(len(projectList)))
	gls.logger.Sugar().Debugf("metrics: %v", gls.cfg.Metrics.GitRepositoryCount)

	return gls.mb.Emit(), nil
}
