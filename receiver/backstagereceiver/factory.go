// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package backstagereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/backstagereceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/liatrio/liatrio-otel-collector/receiver/backstagereceiver/internal/metadata"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(
			createLogsReceiver,
			metadata.LogsStability,
		),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		BackstageAPIConfig: BackstageAPIConfig{
			URL: "http://localhost:7007",
		},
	}
}

func createLogsReceiver(
	_ context.Context,
	params receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	rcfg := cfg.(*Config)
	return newReceiver(params, rcfg, consumer)
}
