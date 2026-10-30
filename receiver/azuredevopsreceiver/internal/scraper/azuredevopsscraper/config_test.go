// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

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
		"organization": "liatrio",
		"project":      "otel",
		"timeout":      "15s",
	})
	require.NoError(t, conf.Unmarshal(cfg))
	assert.Equal(t, "liatrio", cfg.Organization)
	assert.Equal(t, "otel", cfg.Project)
	assert.Equal(t, 15*time.Second, cfg.Timeout)
}
