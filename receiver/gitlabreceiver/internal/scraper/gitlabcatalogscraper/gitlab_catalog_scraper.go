//go:generate ../../../../../.tools/genqlient

package gitlabcatalogscraper

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Khan/genqlient/graphql"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

type gitlabCatalogScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
	rb       *metadata.ResourceBuilder
}

func (gcs *gitlabCatalogScraper) start(ctx context.Context, host component.Host) (err error) {
	gcs.logger.Sugar().Info("Starting the GitLab CI/CD Catalog scraper")

	var extensions map[component.ID]component.Component
	if host != nil {
		extensions = host.GetExtensions()
	}

	gcs.client, err = gcs.cfg.ToClient(ctx, extensions, gcs.settings)
	return
}

func newGitLabCatalogScraper(
	_ context.Context,
	settings receiver.Settings,
	cfg *Config,
) *gitlabCatalogScraper {
	return &gitlabCatalogScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		rb:       metadata.NewResourceBuilder(cfg.ResourceAttributes),
	}
}

func (gcs *gitlabCatalogScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	if gcs.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	graphCURL := "https://gitlab.com/api/graphql"
	restCURL := "https://gitlab.com/"

	if gcs.cfg.ClientConfig.Endpoint != "" {
		var err error

		graphCURL, err = url.JoinPath(gcs.cfg.ClientConfig.Endpoint, "api/graphql")
		if err != nil {
			gcs.logger.Sugar().Errorf("error: %v", err)
		}

		restCURL, err = url.JoinPath(gcs.cfg.ClientConfig.Endpoint, "/")
		if err != nil {
			gcs.logger.Sugar().Errorf("error: %v", err)
		}
	}

	graphClient := graphql.NewClient(graphCURL, gcs.client)
	restClient, err := gitlab.NewClient("", gitlab.WithHTTPClient(gcs.client), gitlab.WithBaseURL(restCURL))
	if err != nil {
		gcs.logger.Sugar().Errorf("error creating REST client: %v", err)
	}

	var mux sync.Mutex

	limiter := make(chan struct{}, gcs.cfg.ConcurrencyLimit)

	// Internal adoption: fetch org projects and their component usages
	if gcs.cfg.GitLabOrg != "" {
		projectList, err := gcs.getProjects(ctx, restClient)
		if err != nil {
			gcs.logger.Sugar().Errorf("error fetching projects: %v", err)
		} else {
			// Step 1: Query componentUsages per project to record per-project counts
			// and identify which projects use components
			projectsWithComponents := []gitlabProject{}

			var wg sync.WaitGroup

			for _, project := range projectList {
				project := project
				wg.Add(1)
				limiter <- struct{}{}

				go func() {
					defer func() {
						<-limiter
						wg.Done()
					}()

					usages, err := gcs.getProjectComponentUsages(ctx, graphClient, project.Path)
					if err != nil {
						gcs.logger.Sugar().Errorf("error fetching component usages for project '%s': %v", project.Path, err)
						return
					}

					mux.Lock()
					gcs.mb.RecordGitlabCatalogProjectUsageCountDataPoint(now, int64(len(usages)), project.URL)
					if len(usages) > 0 {
						projectsWithComponents = append(projectsWithComponents, project)
					}
					mux.Unlock()
				}()
			}

			wg.Wait()

			// Step 2: Fetch CI configs for projects that use components
			// Parse full component paths to avoid bare name collisions
			// (e.g., components/go/test vs components/ruby/test)
			componentProjectCount := make(map[string]int)
			resourcePaths := make(map[string]bool)

			var wg2 sync.WaitGroup
			for _, project := range projectsWithComponents {
				project := project
				wg2.Add(1)
				limiter <- struct{}{}

				go func() {
					defer func() {
						<-limiter
						wg2.Done()
					}()

					paths, err := gcs.getComponentResourcePaths(restClient, project.Path)
					if err != nil {
						gcs.logger.Sugar().Warnf("could not fetch CI config for project '%s': %v", project.Path, err)
						return
					}

					mux.Lock()
					for compName, resourcePath := range paths {
						fullName := resourcePath + "/" + compName
						componentProjectCount[fullName]++
						resourcePaths[resourcePath] = true
					}
					mux.Unlock()
				}()
			}
			wg2.Wait()

			// Step 3: Record component adoption counts using full paths
			for fullName, count := range componentProjectCount {
				gcs.mb.RecordGitlabCatalogComponentProjectCountDataPoint(now, int64(count), fullName)
			}

			// Step 4: Look up catalog resource details by exact path
			var wg3 sync.WaitGroup
			for resourcePath := range resourcePaths {
				resourcePath := resourcePath
				wg3.Add(1)
				limiter <- struct{}{}

				go func() {
					defer func() {
						<-limiter
						wg3.Done()
					}()

					resource, err := gcs.getCatalogResourceByPath(ctx, graphClient, resourcePath)
					if err != nil {
						gcs.logger.Sugar().Warnf("could not fetch catalog resource '%s': %v", resourcePath, err)
						return
					}

					mux.Lock()
					gcs.mb.RecordGitlabCatalogResourceStarCountDataPoint(now, int64(resource.StarCount), resource.Name, resource.FullPath)
					gcs.mb.RecordGitlabCatalogResourceUsageCountDataPoint(now, int64(resource.Last30DayUsageCount), resource.Name, resource.FullPath)
					mux.Unlock()
				}()
			}
			wg3.Wait()
		}
	}

	gcs.rb.SetVcsVendorName("gitlab")
	gcs.rb.SetOrganizationName(gcs.cfg.GitLabOrg)

	res := gcs.rb.Emit()
	return gcs.mb.Emit(metadata.WithResource(res)), nil
}
