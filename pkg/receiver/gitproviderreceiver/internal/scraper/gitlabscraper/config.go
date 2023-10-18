// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
)

// Config relating to GitLab Metric Scraper.
type Config struct {
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	confighttp.HTTPClientSettings `mapstructure:",squash"`
	internal.ScraperConfig
	// GitLabOrg is the name of the GitLab organization to scrape (gitlab scraper only)
	GitLabOrg string `mapstructure:"gitlab_org"`

	ProjectTopic  string `mapstructure:"project_topic"`
	ProjectSearch string `mapstructure:"project_search"`
}
