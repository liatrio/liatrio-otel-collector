// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "httpjson/internal"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
)

type ScraperFactory struct{}

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
		"httpjsonScraper",
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
	s.logger.Sugar().Info("starting the httpjson scraper")
	return nil
}

func (s *scraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	s.logger.Sugar().Info("running the httpjson scrape function")
	return pmetric.NewMetrics(), nil
}
