// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
)

// This file implements factory for the Azure DevOps Scraper

const (
	// TypeStr is the value of "type" key in configuration.
	TypeStr            = "azuredevops"
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
) (scraper.Metrics, error) {
	conf := cfg.(*Config)
	s := newAzureDevOpsScraper(ctx, params, conf)

	return scraper.NewMetrics(
		s.scrape,
		scraper.WithStart(s.start),
	)
}
