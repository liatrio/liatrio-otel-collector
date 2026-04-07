package gitlabterraformscraper

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

type gitlabTerraformScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
	rb       *metadata.ResourceBuilder
}

func (gts *gitlabTerraformScraper) start(ctx context.Context, host component.Host) (err error) {
	gts.logger.Sugar().Info("Starting the GitLab Terraform scraper")

	var extensions map[component.ID]component.Component
	if host != nil {
		extensions = host.GetExtensions()
	}

	gts.client, err = gts.cfg.ToClient(ctx, extensions, gts.settings)
	return
}

func newGitLabTerraformScraper(
	_ context.Context,
	settings receiver.Settings,
	cfg *Config,
) *gitlabTerraformScraper {
	return &gitlabTerraformScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		rb:       metadata.NewResourceBuilder(cfg.ResourceAttributes),
	}
}

func (gts *gitlabTerraformScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	if gts.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	// Build the REST client URL, supporting self-hosted GitLab instances
	restCURL := "https://gitlab.com/"
	if gts.cfg.ClientConfig.Endpoint != "" {
		var err error
		restCURL, err = url.JoinPath(gts.cfg.ClientConfig.Endpoint, "/")
		if err != nil {
			gts.logger.Sugar().Errorf("error building REST URL: %v", err)
			return gts.mb.Emit(), err
		}
	}

	restClient, err := gitlab.NewClient("", gitlab.WithHTTPClient(gts.client), gitlab.WithBaseURL(restCURL))
	if err != nil {
		gts.logger.Sugar().Errorf("error creating GitLab REST client: %v", err)
		return gts.mb.Emit(), err
	}

	// Discover all Terraform modules in the group registry
	modules, err := gts.getModules(ctx, restClient)
	if err != nil {
		gts.logger.Sugar().Errorf("error getting Terraform modules: %v", err)
		return gts.mb.Emit(), err
	}

	// Record total module count
	gts.mb.RecordVcsTerraformModuleCountDataPoint(now, int64(len(modules)))

	// Search for consumers of each module concurrently
	var wg sync.WaitGroup
	wg.Add(len(modules))
	var mux sync.Mutex

	var max int
	switch {
	case gts.cfg.ConcurrencyLimit > 0:
		max = gts.cfg.ConcurrencyLimit
	default:
		max = len(modules) + 1
	}

	limiter := make(chan struct{}, max)

	for _, module := range modules {
		module := module
		limiter <- struct{}{}

		go func() {
			defer func() {
				<-limiter
				wg.Done()
			}()

			consumers, err := gts.searchModuleConsumers(ctx, restClient, module)
			if err != nil {
				gts.logger.Sugar().Errorf("error searching consumers for module '%s/%s': %v", module.Name, module.System, err)
				return
			}

			mux.Lock()
			gts.mb.RecordVcsTerraformModuleConsumerCountDataPoint(now, int64(len(consumers)), module.Name, module.System)
			for _, consumer := range consumers {
				gts.mb.RecordVcsTerraformModuleConsumerDataPoint(now, int64(1), module.Name, module.System, consumer.ProjectName, consumer.ProjectURL)
			}
			mux.Unlock()
		}()
	}

	wg.Wait()

	gts.rb.SetVcsVendorName("gitlab")
	gts.rb.SetOrganizationName(gts.cfg.GitLabOrg)

	gts.logger.Sugar().Infof("Finished processing Terraform modules for GitLab group %s", gts.cfg.GitLabOrg)

	res := gts.rb.Emit()
	return gts.mb.Emit(metadata.WithResource(res)), nil
}
