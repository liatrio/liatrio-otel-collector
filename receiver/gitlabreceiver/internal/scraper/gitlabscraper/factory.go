// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
)

// This file implements factory for the GitLab Scraper

const (
	// TypeStr is the value of "type" key in configuration.
	TypeStr            = "gitlab"
	defaultHTTPTimeout = 15 * time.Second
)

type Factory struct{}

func (f *Factory) CreateDefaultConfig() internal.Config {
	return &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		ClientConfig: confighttp.ClientConfig{
			Timeout: defaultHTTPTimeout,
		},
	}
}

func (f *Factory) CreateMetricsScraper(
	ctx context.Context,
	params receiver.Settings,
	cfg internal.Config,
) (scraperhelper.Scraper, error) {
	conf := cfg.(*Config)
	s := newGitLabScraper(ctx, params, conf)

	return scraperhelper.NewScraperWithComponentType(
		metadata.Type,
		s.scrape,
		scraperhelper.WithStart(s.start),
	)
}
