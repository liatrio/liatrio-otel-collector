// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
)

// This file implements factory for the GitLab Scraper as part of the  Git Provider Receiver

const (
	// TypeStr is the value of "type" key in configuration.
	TypeStr            = "gitlab"
	defaultHTTPTimeout = 15 * time.Second
)

type Factory struct{}

func (f *Factory) CreateDefaultConfig() internal.Config {
	return &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Timeout: defaultHTTPTimeout,
		},
	}
}

func (f *Factory) CreateMetricsScraper(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg internal.Config,
) (scraperhelper.Scraper, error) {
	config := cfg.(*Config)
	scraper := newGitLabScraper(ctx, params, config)

	return scraperhelper.NewScraper(
		TypeStr,
		scraper.scrape,
		scraperhelper.WithStart(scraper.start),
	)
}
