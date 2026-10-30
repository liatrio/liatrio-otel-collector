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

func TestConfigUnmarshalKeepsSiblingFields(t *testing.T) {
	cfg := &Config{}
	conf := confmap.NewFromStringMap(map[string]any{
		"github_org": "liatrio",
		"timeout":    "15s",
		"endpoint":   "https://api.github.com",
	})
	require.NoError(t, conf.Unmarshal(cfg))
	assert.Equal(t, "liatrio", cfg.GitHubOrg)
	assert.Equal(t, 15*time.Second, cfg.Timeout)
	assert.Equal(t, "https://api.github.com", cfg.Endpoint)
}
