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

	// SearchQuery is the specific Azure DevOps project to scrape
	SearchQuery string `mapstructure:"search_query"`

	// BaseURL for Azure DevOps API (defaults to https://dev.azure.com)
	BaseURL string `mapstructure:"base_url"`

	// LimitPullRequests specifies the number of days in the past to search for pull requests
	LimitPullRequests int `mapstructure:"limit_pull_requests"`

	// ConcurrencyLimit limits the number of concurrent API requests
	ConcurrencyLimit int `mapstructure:"concurrency_limit"`

	// DeploymentPipelineName is the name of the Release Pipeline to scrape deployments from
	DeploymentPipelineName string `mapstructure:"deployment_pipeline_name"`

	// DeploymentStageName is the name of the Stage/Environment within the pipeline to track
	DeploymentStageName string `mapstructure:"deployment_stage_name"`

	// DeploymentLookbackDays specifies how many days back to fetch deployment history
	DeploymentLookbackDays int `mapstructure:"deployment_lookback_days"`

	// WorkItemTypes is a list of work item types to track (e.g., "User Story", "Bug", "Task")
	// If empty, defaults to ["User Story", "Bug"]
	WorkItemTypes []string `mapstructure:"work_item_types"`

	// WorkItemLookbackDays specifies how many days back to fetch work item history
	// Defaults to 30 days if not set
	WorkItemLookbackDays int `mapstructure:"work_item_lookback_days"`
}

var _ internal.Config = (*Config)(nil)
