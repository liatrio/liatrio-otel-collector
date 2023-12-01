// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package MyAwesomeReceiver // import "github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver/internal/metadata"
)

var (
	errConfigNotValid = errors.New("configuration is not valid")
)

// NewFactory creates a factory for the new receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
	)
}

// Create the default config based on the const(s) defined above.
func createDefaultConfig() component.Config {
	return &Config{ScraperControllerSettings: scraperhelper.NewDefaultScraperControllerSettings(metadata.Type)}
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

	scraperConfig := internal.ScraperConfig{ScraperControllerSettings: conf.ScraperControllerSettings}

	addScraperOpts, err := createAddScraperOpts(ctx, params, scraperConfig)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&conf.ScraperControllerSettings,
		params,
		consumer,
		addScraperOpts,
	)
}

func createAddScraperOpts(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg internal.ScraperConfig,
) (scraperhelper.ScraperControllerOption, error) {

	factory := internal.ScraperFactory{}
	scraper, err := factory.CreateMetricsScraper(ctx, params, cfg)
	if err != nil {
		return nil, err
	}

	return scraperhelper.AddScraper(scraper), nil
}
