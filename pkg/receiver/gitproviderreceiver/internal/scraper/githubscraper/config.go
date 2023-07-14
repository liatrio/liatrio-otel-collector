// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
)

// Config relating to Github Metric Scraper.
type Config struct {
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	confighttp.HTTPClientSettings `mapstructure:",squash"`
	internal.ScraperConfig
	// GitHubOrg is the name of the GitHub organization to srape (github scraper only)
	GitHubOrg string `mapstructure:"github_org"`
}
