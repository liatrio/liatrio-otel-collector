// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestMetricsBuilderConfig(t *testing.T) {
	tests := []struct {
		name string
		want MetricsBuilderConfig
	}{
		{
			name: "default",
			want: DefaultMetricsBuilderConfig(),
		},
		{
			name: "all_set",
			want: MetricsBuilderConfig{
				Metrics: MetricsConfig{
					GitRepositoryBranchAdditionCount:       MetricConfig{Enabled: true},
					GitRepositoryBranchCommitAheadbyCount:  MetricConfig{Enabled: true},
					GitRepositoryBranchCommitBehindbyCount: MetricConfig{Enabled: true},
					GitRepositoryBranchCount:               MetricConfig{Enabled: true},
					GitRepositoryBranchDeletionCount:       MetricConfig{Enabled: true},
					GitRepositoryBranchTime:                MetricConfig{Enabled: true},
					GitRepositoryContributorCount:          MetricConfig{Enabled: true},
					GitRepositoryCount:                     MetricConfig{Enabled: true},
					GitRepositoryPullRequestApprovalTime:   MetricConfig{Enabled: true},
					GitRepositoryPullRequestCount:          MetricConfig{Enabled: true},
					GitRepositoryPullRequestDeploymentTime: MetricConfig{Enabled: true},
					GitRepositoryPullRequestMergeTime:      MetricConfig{Enabled: true},
					GitRepositoryPullRequestTime:           MetricConfig{Enabled: true},
				},
				ResourceAttributes: ResourceAttributesConfig{
					GitVendorName:    ResourceAttributeConfig{Enabled: true},
					OrganizationName: ResourceAttributeConfig{Enabled: true},
				},
			},
		},
		{
			name: "none_set",
			want: MetricsBuilderConfig{
				Metrics: MetricsConfig{
					GitRepositoryBranchAdditionCount:       MetricConfig{Enabled: false},
					GitRepositoryBranchCommitAheadbyCount:  MetricConfig{Enabled: false},
					GitRepositoryBranchCommitBehindbyCount: MetricConfig{Enabled: false},
					GitRepositoryBranchCount:               MetricConfig{Enabled: false},
					GitRepositoryBranchDeletionCount:       MetricConfig{Enabled: false},
					GitRepositoryBranchTime:                MetricConfig{Enabled: false},
					GitRepositoryContributorCount:          MetricConfig{Enabled: false},
					GitRepositoryCount:                     MetricConfig{Enabled: false},
					GitRepositoryPullRequestApprovalTime:   MetricConfig{Enabled: false},
					GitRepositoryPullRequestCount:          MetricConfig{Enabled: false},
					GitRepositoryPullRequestDeploymentTime: MetricConfig{Enabled: false},
					GitRepositoryPullRequestMergeTime:      MetricConfig{Enabled: false},
					GitRepositoryPullRequestTime:           MetricConfig{Enabled: false},
				},
				ResourceAttributes: ResourceAttributesConfig{
					GitVendorName:    ResourceAttributeConfig{Enabled: false},
					OrganizationName: ResourceAttributeConfig{Enabled: false},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadMetricsBuilderConfig(t, tt.name)
			if diff := cmp.Diff(tt.want, cfg, cmpopts.IgnoreUnexported(MetricConfig{}, ResourceAttributeConfig{})); diff != "" {
				t.Errorf("Config mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func loadMetricsBuilderConfig(t *testing.T, name string) MetricsBuilderConfig {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	sub, err := cm.Sub(name)
	require.NoError(t, err)
	cfg := DefaultMetricsBuilderConfig()
	require.NoError(t, component.UnmarshalConfig(sub, &cfg))
	return cfg
}

func TestResourceAttributesConfig(t *testing.T) {
	tests := []struct {
		name string
		want ResourceAttributesConfig
	}{
		{
			name: "default",
			want: DefaultResourceAttributesConfig(),
		},
		{
			name: "all_set",
			want: ResourceAttributesConfig{
				GitVendorName:    ResourceAttributeConfig{Enabled: true},
				OrganizationName: ResourceAttributeConfig{Enabled: true},
			},
		},
		{
			name: "none_set",
			want: ResourceAttributesConfig{
				GitVendorName:    ResourceAttributeConfig{Enabled: false},
				OrganizationName: ResourceAttributeConfig{Enabled: false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadResourceAttributesConfig(t, tt.name)
			if diff := cmp.Diff(tt.want, cfg, cmpopts.IgnoreUnexported(ResourceAttributeConfig{})); diff != "" {
				t.Errorf("Config mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func loadResourceAttributesConfig(t *testing.T, name string) ResourceAttributesConfig {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	sub, err := cm.Sub(name)
	require.NoError(t, err)
	sub, err = sub.Sub("resource_attributes")
	require.NoError(t, err)
	cfg := DefaultResourceAttributesConfig()
	require.NoError(t, component.UnmarshalConfig(sub, &cfg))
	return cfg
}
