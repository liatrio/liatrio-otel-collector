// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabcatalogscraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfigUnmarshalKeepsSiblingFields(t *testing.T) {
	cfg := &Config{}
	conf := confmap.NewFromStringMap(map[string]any{
		"gitlab_org":        "liatrio",
		"concurrency_limit": 3,
	})
	require.NoError(t, conf.Unmarshal(cfg))
	assert.Equal(t, "liatrio", cfg.GitLabOrg)
	assert.Equal(t, 3, cfg.ConcurrencyLimit)
}
