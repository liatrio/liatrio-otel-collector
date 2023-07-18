// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/xanzy/go-gitlab"
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

type Group struct {
	ID     int
	Name   string `json:"name"`
	WebURL string
}

type Project struct {
	ID     int
	Name   string
	WebURL string
}

type Commit struct {
	Date string
}

type Branch struct {
	Name            string
	CommitCount     int
	CreatedDate     time.Time
	LastUpdatedDate string
	EndCursor       string
}

type PullRequest struct {
	Title       string
	CreatedDate time.Time
	ClosedDate  time.Time
}

type Repo struct {
	Name          string
	Owner         string
	DefaultBranch string
	Branches      []Branch
	PullRequests  []PullRequest
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

func (gls *gitlabScraper) getGroups() []int {

	var ids []int
	client, _ := gitlab.NewClient(os.Getenv("GL_PAT"))

	opt := &gitlab.ListGroupsOptions{ListOptions: gitlab.ListOptions{
		PerPage: 10,
		Page:    1,
	}}

	groups, _, err := client.Groups.ListGroups(opt)

	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	for _, group := range groups {
		fmt.Println(group.Name)
		ids = append(ids, group.ID)
	}

	return ids
}

func (gls *gitlabScraper) getProjects() {

	client, _ := gitlab.NewClient(gls.cfg.GitLabPat)

	opt := &gitlab.ListProjectsOptions{ListOptions: gitlab.ListOptions{
		PerPage: 10,
		Page:    1,
	}}

	projects, _, err := client.Projects.ListProjects(opt)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	for _, project := range projects {
		fmt.Println(project.Name)

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

	gls.getGroups()
	gls.getProjects()

	gls.logger.Sugar().Debugf("metrics: %v", gls.cfg.Metrics.GitRepositoryCount)
	return gls.mb.Emit(), nil
}
