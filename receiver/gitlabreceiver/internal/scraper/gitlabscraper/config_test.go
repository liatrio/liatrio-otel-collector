// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfigUnmarshalKeepsSiblingFields(t *testing.T) {
	cfg := &Config{}
	conf := confmap.NewFromStringMap(map[string]any{
		"gitlab_org":   "liatrio",
		"search_query": "topic:otel",
		"timeout":      "15s",
	})
	require.NoError(t, conf.Unmarshal(cfg))
	assert.Equal(t, "liatrio", cfg.GitLabOrg)
	assert.Equal(t, "topic:otel", cfg.SearchQuery)
	assert.Equal(t, 15*time.Second, cfg.Timeout)
}
