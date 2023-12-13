// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver/internal/metadata"
)

// Config relating to GitLab Metric Scraper.
type Config struct {
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	confighttp.HTTPClientSettings `mapstructure:",squash"`
	internal.ScraperConfig
	// GitLabOrg is the name of the GitLab organization to scrape (gitlab scraper only)
	GitLabOrg string `mapstructure:"gitlab_org"`

	SearchTopic string `mapstructure:"search_topic"`
	SearchQuery string `mapstructure:"search_query"`
}
