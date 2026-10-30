// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabprocessor

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
		"token":   "glpat-example",
		"timeout": "20s",
	})
	require.NoError(t, conf.Unmarshal(cfg))
	assert.Equal(t, "glpat-example", cfg.Token)
	assert.Equal(t, 20*time.Second, cfg.Timeout)
}
