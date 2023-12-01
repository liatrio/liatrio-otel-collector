// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package MyAwesomeReceiver

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver/internal/metadata"
)

var creationSet = receivertest.NewNopCreateSettings()

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	tReceiver, err := factory.CreateTracesReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.Equal(t, err, component.ErrDataTypeIsNotSupported)
	assert.Nil(t, tReceiver)

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, mReceiver)

	tLogs, err := factory.CreateLogsReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.Equal(t, err, component.ErrDataTypeIsNotSupported)
	assert.Nil(t, tLogs)
}

func TestCreateReceiver_ScraperConfigError(t *testing.T) {
	const errorKey string = "error"

	factory := NewFactory()
	cfg := &Config{
		ScraperControllerSettings: scraperhelper.NewDefaultScraperControllerSettings(metadata.Type),
		sample:                    Sample{"data"},
	}

	_, err := factory.CreateMetricsReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.EqualError(t, err, fmt.Sprintf("failed to create scraper %q: factory not found for scraper %q", errorKey, errorKey))
}
