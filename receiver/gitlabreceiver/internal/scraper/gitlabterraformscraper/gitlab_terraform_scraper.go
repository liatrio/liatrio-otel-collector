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
	if gts.cfg.Endpoint != "" {
		var err error
		restCURL, err = url.JoinPath(gts.cfg.Endpoint, "/")
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

	// Each metric corresponds to a tier of work; disable a metric in config to
	// skip the API calls that feed it. Tiers are cumulative because each builds
	// on the previous tier's data:
	//   - module.count          : Packages API only
	//   - module.consumer.count : + Search API per module
	//   - module.consumer       : + Projects API per consumer (name/URL lookup)
	metricsCfg := gts.cfg.Metrics
	moduleCountEnabled := metricsCfg.VcsTerraformModuleCount.Enabled
	consumerCountEnabled := metricsCfg.VcsTerraformModuleConsumerCount.Enabled
	consumerEnabled := metricsCfg.VcsTerraformModuleConsumer.Enabled

	// Tier 1: module discovery. Skip if no metric in this scraper is enabled.
	if !moduleCountEnabled && !consumerCountEnabled && !consumerEnabled {
		gts.rb.SetVcsVendorName("gitlab")
		gts.rb.SetOrganizationName(gts.cfg.GitLabOrg)
		return gts.mb.Emit(metadata.WithResource(gts.rb.Emit())), nil
	}

	modules, err := gts.getModules(ctx, restClient)
	if err != nil {
		gts.logger.Sugar().Errorf("error getting Terraform modules: %v", err)
		return gts.mb.Emit(), err
	}

	if moduleCountEnabled {
		gts.mb.RecordVcsTerraformModuleCountDataPoint(now, int64(len(modules)))
	}

	// Tiers 2 and 3: skip the consumer search work entirely when neither
	// consumer metric is enabled.
	if !consumerCountEnabled && !consumerEnabled {
		gts.rb.SetVcsVendorName("gitlab")
		gts.rb.SetOrganizationName(gts.cfg.GitLabOrg)
		return gts.mb.Emit(metadata.WithResource(gts.rb.Emit())), nil
	}

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

			consumers, err := gts.searchModuleConsumers(ctx, restClient, module, consumerEnabled)
			if err != nil {
				gts.logger.Sugar().Errorf("error searching consumers for module '%s/%s': %v", module.Name, module.System, err)
				return
			}

			mux.Lock()
			if consumerCountEnabled {
				gts.mb.RecordVcsTerraformModuleConsumerCountDataPoint(now, int64(len(consumers)), module.Name, module.System)
			}
			if consumerEnabled {
				for _, consumer := range consumers {
					gts.mb.RecordVcsTerraformModuleConsumerDataPoint(now, int64(1), module.Name, module.System, consumer.ProjectName, consumer.ProjectURL)
				}
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
