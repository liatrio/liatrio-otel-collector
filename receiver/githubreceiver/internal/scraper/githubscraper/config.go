// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper // import "github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/scraper/githubscraper"

import (
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"

	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"
)

// Config relating to GitHub Metric Scraper.
type Config struct {
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	confighttp.ClientConfig       `mapstructure:",squash"`
	internal.ScraperConfig
	// GitHubOrg is the name of the GitHub organization to scrape (github scraper only)
	GitHubOrg string `mapstructure:"github_org"`
	// SearchQuery is the query to use when defining a custom search for repository data
	SearchQuery      string `mapstructure:"search_query"`
	GitHubTeam       string `mapstructure:"github_team"`
	ConcurrencyLimit int    `mapstructure:"concurrency_limit"`
}

// Unmarshal decodes scraper settings, including fields beside the embedded client
// config. confighttp.ClientConfig implements confmap.Unmarshaler, and embedding
// promotes that method onto Config. confmap would then decode only the client
// config and drop github_org, search_query, and the other scraper keys.
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if conf == nil {
		return nil
	}
	return conf.Unmarshal(cfg, confmap.WithIgnoreUnused())
}
