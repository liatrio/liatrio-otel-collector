// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
)

// Config defines configuration for Azure DevOps scraper
type Config struct {
	confighttp.ClientConfig           `mapstructure:",squash"`
	metadata.MetricsBuilderConfig     `mapstructure:",squash"`
	metadata.ResourceAttributesConfig `mapstructure:"resource_attributes"`

	// Organization is the Azure DevOps organization name
	Organization string `mapstructure:"organization"`

	// Project is the specific Azure DevOps project to scrape
	Project string `mapstructure:"project"`

	// // PersonalAccessToken for authentication with Azure DevOps API
	// PersonalAccessToken string `mapstructure:"personal_access_token"`

	// BaseURL for Azure DevOps API (defaults to https://dev.azure.com)
	BaseURL string `mapstructure:"base_url"`

	// LimitPullRequests specifies the number of days in the past to search for pull requests
	LimitPullRequests int `mapstructure:"limit_pull_requests"`

	// ConcurrencyLimit limits the number of concurrent API requests
	ConcurrencyLimit int `mapstructure:"concurrency_limit"`
}

var _ internal.Config = (*Config)(nil)
