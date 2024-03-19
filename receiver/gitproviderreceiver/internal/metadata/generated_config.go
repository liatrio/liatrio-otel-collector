// Code generated by mdatagen. DO NOT EDIT.

package metadata

import "go.opentelemetry.io/collector/confmap"

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

// MetricsConfig provides config for gitprovider metrics.
type MetricsConfig struct {
	GitRepositoryBranchCommitAheadbyCount  MetricConfig `mapstructure:"git.repository.branch.commit.aheadby.count"`
	GitRepositoryBranchCommitBehindbyCount MetricConfig `mapstructure:"git.repository.branch.commit.behindby.count"`
	GitRepositoryBranchCount               MetricConfig `mapstructure:"git.repository.branch.count"`
	GitRepositoryBranchLineAdditionCount   MetricConfig `mapstructure:"git.repository.branch.line.addition.count"`
	GitRepositoryBranchLineDeletionCount   MetricConfig `mapstructure:"git.repository.branch.line.deletion.count"`
	GitRepositoryBranchTime                MetricConfig `mapstructure:"git.repository.branch.time"`
	GitRepositoryContributorCount          MetricConfig `mapstructure:"git.repository.contributor.count"`
	GitRepositoryCount                     MetricConfig `mapstructure:"git.repository.count"`
	GitRepositoryPullRequestMergedCount    MetricConfig `mapstructure:"git.repository.pull_request.merged.count"`
	GitRepositoryPullRequestOpenCount      MetricConfig `mapstructure:"git.repository.pull_request.open.count"`
	GitRepositoryPullRequestOpenTime       MetricConfig `mapstructure:"git.repository.pull_request.open.time"`
	GitRepositoryPullRequestTimeToApproval MetricConfig `mapstructure:"git.repository.pull_request.time_to_approval"`
	GitRepositoryPullRequestTimeToMerge    MetricConfig `mapstructure:"git.repository.pull_request.time_to_merge"`
}

func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		GitRepositoryBranchCommitAheadbyCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryBranchCommitBehindbyCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryBranchCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryBranchLineAdditionCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryBranchLineDeletionCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryBranchTime: MetricConfig{
			Enabled: true,
		},
		GitRepositoryContributorCount: MetricConfig{
			Enabled: false,
		},
		GitRepositoryCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryPullRequestMergedCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryPullRequestOpenCount: MetricConfig{
			Enabled: true,
		},
		GitRepositoryPullRequestOpenTime: MetricConfig{
			Enabled: true,
		},
		GitRepositoryPullRequestTimeToApproval: MetricConfig{
			Enabled: true,
		},
		GitRepositoryPullRequestTimeToMerge: MetricConfig{
			Enabled: true,
		},
	}
}

// ResourceAttributeConfig provides common config for a particular resource attribute.
type ResourceAttributeConfig struct {
	Enabled bool `mapstructure:"enabled"`

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

// ResourceAttributesConfig provides config for gitprovider resource attributes.
type ResourceAttributesConfig struct {
	GitVendorName    ResourceAttributeConfig `mapstructure:"git.vendor.name"`
	OrganizationName ResourceAttributeConfig `mapstructure:"organization.name"`
}

func DefaultResourceAttributesConfig() ResourceAttributesConfig {
	return ResourceAttributesConfig{
		GitVendorName: ResourceAttributeConfig{
			Enabled: true,
		},
		OrganizationName: ResourceAttributeConfig{
			Enabled: true,
		},
	}
}

// MetricsBuilderConfig is a configuration for gitprovider metrics builder.
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
