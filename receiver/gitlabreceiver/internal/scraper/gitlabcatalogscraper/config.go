// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabcatalogscraper

import (
	"errors"

	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
)

// Config relating to GitLab CI/CD Catalog Metric Scraper.
type Config struct {
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	confighttp.ClientConfig       `mapstructure:",squash"`
	internal.ScraperConfig
	// GitLabOrg is the name of the GitLab group to scan for internal adoption.
	GitLabOrg string `mapstructure:"gitlab_org"`
	// ConcurrencyLimit controls the maximum number of concurrent API requests.
	ConcurrencyLimit int `mapstructure:"concurrency_limit"`
}

func (cfg *Config) Validate() error {
	if cfg.GitLabOrg == "" {
		return errors.New("gitlab_org is required")
	}
	if cfg.ConcurrencyLimit < 1 {
		return errors.New("concurrency_limit must be at least 1")
	}
	return nil
}
