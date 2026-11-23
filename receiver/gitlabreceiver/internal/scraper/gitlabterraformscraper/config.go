// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabterraformscraper

import (
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
)

// Config relating to GitLab Terraform Module Adoption Scraper.
type Config struct {
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	confighttp.ClientConfig       `mapstructure:",squash"`
	internal.ScraperConfig
	// GitLabOrg is the name of the GitLab group to scrape for Terraform module adoption
	GitLabOrg        string `mapstructure:"gitlab_org"`
	ConcurrencyLimit int    `mapstructure:"concurrency_limit"`
}

// Unmarshal decodes scraper settings, including fields beside the embedded client
// config. confighttp.ClientConfig implements confmap.Unmarshaler, and embedding
// promotes that method onto Config. confmap would then decode only the client
// config and drop gitlab_org and concurrency_limit.
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if conf == nil {
		return nil
	}
	return conf.Unmarshal(cfg, confmap.WithIgnoreUnused())
}
