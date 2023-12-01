// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver/internal"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
)

type ScraperFactory struct {
	// Create the default configuration for the sccraper.
	// CreateDefaultConfig() ScraperConfig
	// Create a scraper based on the configuration passed or return an error if not valid.
	// CreateMetricsScraper(ctx context.Context, params receiver.CreateSettings, cfg ScraperConfig) (scraperhelper.Scraper, error)
}

func (f *ScraperFactory) CreateMetricsScraper(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg ScraperConfig,
) (scraperhelper.Scraper, error) {
	s := &scraper{
		cfg:      &cfg,
		logger:   params.Logger,
		settings: params.TelemetrySettings,
	}

	return scraperhelper.NewScraper(
		"MyAwesomeReceiverScraper",
		s.scrape,
		scraperhelper.WithStart(s.start),
	)
}

type ScraperConfig struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
}

type scraper struct {
	cfg      *ScraperConfig
	logger   *zap.Logger
	settings component.TelemetrySettings
}

func (s *scraper) start(_ context.Context, host component.Host) error {
	s.logger.Sugar().Info("starting the MyAwesomeReceiver scraper")
	return nil
}

func (s *scraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	s.logger.Sugar().Info("running the MyAwesomeReceiver scrape function")
	return pmetric.NewMetrics(), nil
}
