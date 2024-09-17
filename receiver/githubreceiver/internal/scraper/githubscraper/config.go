// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package githubscraper // import "github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/scraper/githubscraper"

import (
	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"
)

// Config relating to Github Metric Scraper.
type Config struct {
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	confighttp.ClientConfig       `mapstructure:",squash"`
	internal.ScraperConfig
	// GitHubOrg is the name of the GitHub organization to srape (github scraper only)
	GitHubOrg string `mapstructure:"github_org"`
	// SearchQuery is the query to use when defining a custom search for repository data
	SearchQuery string `mapstructure:"search_query"`
}