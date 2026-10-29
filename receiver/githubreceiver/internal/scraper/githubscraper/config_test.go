// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"

	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"
)

// TestConfig ensures a config created with the factory is the same as one created manually with
// the exported Config struct.
func TestConfig(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	clientConfig := confighttp.NewDefaultClientConfig()
	clientConfig.Timeout = 15 * time.Second

	expectedConfig := &Config{
		MetricsBuilderConfig: metadata.NewDefaultMetricsBuilderConfig(),
		ClientConfig:         clientConfig,
	}

	assert.Equal(t, expectedConfig, defaultConfig)
}

// TestConfigUnmarshalSiblingFields locks the collector v0.164 confighttp.ClientConfig
// behavior: sibling keys must survive unmarshaling next to the squashed HTTP client.
func TestConfigUnmarshalSiblingFields(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":     "https://example.github.test",
		"github_org":   "liatrio",
		"search_query": "org:liatrio",
	})
	cfg := (&Factory{}).CreateDefaultConfig().(*Config)
	require.NoError(t, conf.Unmarshal(cfg))
	assert.Equal(t, "https://example.github.test", cfg.ClientConfig.Endpoint)
	assert.Equal(t, "liatrio", cfg.GitHubOrg)
	assert.Equal(t, "org:liatrio", cfg.SearchQuery)
}
