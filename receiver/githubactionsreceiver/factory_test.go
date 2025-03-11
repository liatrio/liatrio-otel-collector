// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/liatrio/liatrio-otel-collector/receiver/githubactionsreceiver/internal/metadata"
)

func TestFactoryCreate(t *testing.T) {
	factory := NewFactory()
	require.EqualValues(t, "githubactions", factory.Type().String())
}

func TestDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	require.NotNil(t, cfg, "Failed to create default configuration")
}

func TestCreateTracesReceiver(t *testing.T) {
	tests := []struct {
		desc string
		run  func(t *testing.T)
	}{
		{
			desc: "Defaults with valid inputs",
			run: func(t *testing.T) {
				t.Parallel()

				cfg := createDefaultConfig().(*Config)
				cfg.Endpoint = "localhost:8080"
				require.NoError(t, cfg.Validate(), "error validating default config")

				_, err := createTracesReceiver(
					context.Background(),
					receivertest.NewNopSettings(metadata.Type),
					cfg,
					consumertest.NewNop(),
				)
				require.NoError(t, err, "failed to create trace receiver")
			},
		},
		{
			desc: "Missing consumer",
			run: func(t *testing.T) {
				t.Parallel()

				cfg := createDefaultConfig().(*Config)
				cfg.Endpoint = "localhost:8080"
				require.NoError(t, cfg.Validate(), "error validating default config")

				_, err := createTracesReceiver(
					context.Background(),
					receivertest.NewNopSettings(metadata.Type),
					cfg,
					nil,
				)
				require.Error(t, err, "Succeeded in creating a receiver without a consumer")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, test.run)
	}
}
