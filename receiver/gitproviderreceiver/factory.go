// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitproviderreceiver // import "github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal/metadata"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal/scraper/githubscraper"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal/scraper/gitlabscraper"
)

// This file implements a factory for the git provider receiver

//const (
//	defaultInterval = 30 * time.Second
//	defaultTimeout  = 15 * time.Second
//)

var (
	scraperFactories = map[string]internal.ScraperFactory{
		githubscraper.TypeStr: &githubscraper.Factory{},
		gitlabscraper.TypeStr: &gitlabscraper.Factory{},
	}

	errConfigNotValid = errors.New("configuration is not valid for the git provider receiver")
)

// NewFactory creates a factory for the git provider receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
	)
}

// Gets a factory for defined scraper.
func getScraperFactory(key string) (internal.ScraperFactory, bool) {
	if factory, ok := scraperFactories[key]; ok {
		return factory, true
	}

	return nil, false
}

// Create the default config based on the const(s) defined above.
func createDefaultConfig() component.Config {
	return &Config{
		ScraperControllerSettings: scraperhelper.NewDefaultScraperControllerSettings(metadata.Type),
		// TODO: metrics builder configuration may need to be in each sub scraper,
		// TODO: for right now setting here because the metrics in this receiver will apply to all
		// TODO: scrapers defined as a common set of gitprovider
		// TODO: aqp completely remove these comments if the metrics build config
		// needs to be defined in each scraper
		// MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
}

// Create the metrics receiver according to the OTEL conventions taking in the
// context, receiver params, configuration from the component, and consumer (process or exporter)
func createMetricsReceiver(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {

	// check that the configuration is valid
	conf, ok := cfg.(*Config)
	if !ok {
		return nil, errConfigNotValid
	}

	addScraperOpts, err := createAddScraperOpts(ctx, params, conf, scraperFactories)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&conf.ScraperControllerSettings,
		params,
		consumer,
		addScraperOpts...,
	)
}

func createAddScraperOpts(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg *Config,
	factories map[string]internal.ScraperFactory,
) ([]scraperhelper.ScraperControllerOption, error) {
	scraperControllerOptions := make([]scraperhelper.ScraperControllerOption, 0, len(cfg.Scrapers))

	for key, cfg := range cfg.Scrapers {
		gitProviderScraper, err := createGitProviderScraper(ctx, params, key, cfg, factories)
		if err != nil {
			return nil, fmt.Errorf("failed to create scraper %q: %w", key, err)
		}

		scraperControllerOptions = append(scraperControllerOptions, scraperhelper.AddScraper(gitProviderScraper))
	}

	return scraperControllerOptions, nil
}

func createGitProviderScraper(
	ctx context.Context,
	params receiver.CreateSettings,
	key string,
	cfg internal.Config,
	factories map[string]internal.ScraperFactory,
) (scraper scraperhelper.Scraper, err error) {
	factory := factories[key]
	if factory == nil {
		return nil, fmt.Errorf("factory not found for scraper %q", key)
	}

	scraper, err = factory.CreateMetricsScraper(ctx, params, cfg)
	if err != nil {
		return nil, err
	}

	return
}
