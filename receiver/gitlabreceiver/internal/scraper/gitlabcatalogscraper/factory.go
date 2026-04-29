// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabcatalogscraper

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
)

const (
	// TypeStr is the value of "type" key in configuration.
	TypeStr            = "gitlab_cicd_catalog"
	defaultHTTPTimeout = 15 * time.Second
)

type Factory struct{}

func (f *Factory) CreateDefaultConfig() internal.Config {
	return &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		ClientConfig: confighttp.ClientConfig{
			Timeout: defaultHTTPTimeout,
		},
		ConcurrencyLimit: 5,
	}
}

func (f *Factory) CreateMetricsScraper(
	ctx context.Context,
	params receiver.Settings,
	cfg internal.Config,
) (scraper.Metrics, error) {
	conf := cfg.(*Config)
	s := newGitLabCatalogScraper(ctx, params, conf)

	return scraper.NewMetrics(
		s.scrape,
		scraper.WithStart(s.start),
	)
}
