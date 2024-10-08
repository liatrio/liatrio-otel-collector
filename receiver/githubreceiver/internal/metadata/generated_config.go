// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/filter"
)

// MetricConfig provides common config for a particular metric.
type MetricConfig struct {
	Enabled bool `mapstructure:"enabled"`

	enabledSetByUser bool
}

func (ms *MetricConfig) Unmarshal(parser *confmap.Conf) error {
	if parser == nil {
		return nil
	}
	err := parser.Unmarshal(ms)
	if err != nil {
		return err
	}
	ms.enabledSetByUser = parser.IsSet("enabled")
	return nil
}

// MetricsConfig provides config for github metrics.
type MetricsConfig struct {
	VcsRepositoryChangeCount          MetricConfig `mapstructure:"vcs.repository.change.count"`
	VcsRepositoryChangeTimeOpen       MetricConfig `mapstructure:"vcs.repository.change.time_open"`
	VcsRepositoryChangeTimeToApproval MetricConfig `mapstructure:"vcs.repository.change.time_to_approval"`
	VcsRepositoryChangeTimeToMerge    MetricConfig `mapstructure:"vcs.repository.change.time_to_merge"`
	VcsRepositoryContributorCount     MetricConfig `mapstructure:"vcs.repository.contributor.count"`
	VcsRepositoryCount                MetricConfig `mapstructure:"vcs.repository.count"`
	VcsRepositoryCveCount             MetricConfig `mapstructure:"vcs.repository.cve.count"`
	VcsRepositoryRefCount             MetricConfig `mapstructure:"vcs.repository.ref.count"`
	VcsRepositoryRefLinesAdded        MetricConfig `mapstructure:"vcs.repository.ref.lines_added"`
	VcsRepositoryRefLinesDeleted      MetricConfig `mapstructure:"vcs.repository.ref.lines_deleted"`
	VcsRepositoryRefRevisionsAhead    MetricConfig `mapstructure:"vcs.repository.ref.revisions_ahead"`
	VcsRepositoryRefRevisionsBehind   MetricConfig `mapstructure:"vcs.repository.ref.revisions_behind"`
	VcsRepositoryRefTime              MetricConfig `mapstructure:"vcs.repository.ref.time"`
}

func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		VcsRepositoryChangeCount: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryChangeTimeOpen: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryChangeTimeToApproval: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryChangeTimeToMerge: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryContributorCount: MetricConfig{
			Enabled: false,
		},
		VcsRepositoryCount: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryCveCount: MetricConfig{
			Enabled: false,
		},
		VcsRepositoryRefCount: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryRefLinesAdded: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryRefLinesDeleted: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryRefRevisionsAhead: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryRefRevisionsBehind: MetricConfig{
			Enabled: true,
		},
		VcsRepositoryRefTime: MetricConfig{
			Enabled: true,
		},
	}
}

// ResourceAttributeConfig provides common config for a particular resource attribute.
type ResourceAttributeConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Experimental: MetricsInclude defines a list of filters for attribute values.
	// If the list is not empty, only metrics with matching resource attribute values will be emitted.
	MetricsInclude []filter.Config `mapstructure:"metrics_include"`
	// Experimental: MetricsExclude defines a list of filters for attribute values.
	// If the list is not empty, metrics with matching resource attribute values will not be emitted.
	// MetricsInclude has higher priority than MetricsExclude.
	MetricsExclude []filter.Config `mapstructure:"metrics_exclude"`

	enabledSetByUser bool
}

func (rac *ResourceAttributeConfig) Unmarshal(parser *confmap.Conf) error {
	if parser == nil {
		return nil
	}
	err := parser.Unmarshal(rac)
	if err != nil {
		return err
	}
	rac.enabledSetByUser = parser.IsSet("enabled")
	return nil
}

// ResourceAttributesConfig provides config for github resource attributes.
type ResourceAttributesConfig struct {
	OrganizationName ResourceAttributeConfig `mapstructure:"organization.name"`
	TeamName         ResourceAttributeConfig `mapstructure:"team.name"`
	VcsVendorName    ResourceAttributeConfig `mapstructure:"vcs.vendor.name"`
}

func DefaultResourceAttributesConfig() ResourceAttributesConfig {
	return ResourceAttributesConfig{
		OrganizationName: ResourceAttributeConfig{
			Enabled: true,
		},
		TeamName: ResourceAttributeConfig{
			Enabled: false,
		},
		VcsVendorName: ResourceAttributeConfig{
			Enabled: true,
		},
	}
}

// MetricsBuilderConfig is a configuration for github metrics builder.
type MetricsBuilderConfig struct {
	Metrics            MetricsConfig            `mapstructure:"metrics"`
	ResourceAttributes ResourceAttributesConfig `mapstructure:"resource_attributes"`
}

func DefaultMetricsBuilderConfig() MetricsBuilderConfig {
	return MetricsBuilderConfig{
		Metrics:            DefaultMetricsConfig(),
		ResourceAttributes: DefaultResourceAttributesConfig(),
	}
}
