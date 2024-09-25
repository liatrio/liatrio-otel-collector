// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/githubreceiver/internal/scraper/githubscraper"

import (
	"context"
	"time"

	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

// This file implements factory for the GitHub Scraper as part of the  GitHub Receiver

const (
	// TypeStr is the value of "type" key in configuration.
	TypeStr            = "github"
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
	s := newGitHubScraper(ctx, params, conf)

	scrapeType, err := component.NewType(TypeStr)
	if err != nil {
		return nil, err
	}
	return scraperhelper.NewScraper(
		scrapeType,
		s.scrape,
		scraperhelper.WithStart(s.start),
	)
}
