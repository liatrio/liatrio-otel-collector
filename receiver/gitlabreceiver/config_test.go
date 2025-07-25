// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol/otelcoltest"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/scraper/gitlabscraper"
)

func TestLoadConfig(t *testing.T) {
	factories, err := otelcoltest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[metadata.Type] = factory
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/33594
	// nolint:staticcheck
	cfg, err := otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Len(t, cfg.Receivers, 2)

	r0 := cfg.Receivers[component.NewID(metadata.Type)]
	defaultConfiggitlabscraper := factory.CreateDefaultConfig()
	defaultConfiggitlabscraper.(*Config).Scrapers = map[string]internal.Config{
		gitlabscraper.TypeStr: (&gitlabscraper.Factory{}).CreateDefaultConfig(),
	}

	assert.Equal(t, defaultConfiggitlabscraper, r0)

	r1 := cfg.Receivers[component.NewIDWithName(metadata.Type, "customname")].(*Config)
	expectedConfig := &Config{
		ControllerConfig: scraperhelper.ControllerConfig{
			CollectionInterval: 30 * time.Second,
			InitialDelay:       1 * time.Second,
		},
		Scrapers: map[string]internal.Config{
			gitlabscraper.TypeStr: (&gitlabscraper.Factory{}).CreateDefaultConfig(),
		},
	}

	assert.Equal(t, expectedConfig, r1)
}

func TestLoadInvalidConfig_NoScrapers(t *testing.T) {
	factories, err := otelcoltest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[metadata.Type] = factory
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/33594
	// nolint:staticcheck
	_, err = otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "config-noscrapers.yaml"), factories)

	require.Contains(t, err.Error(), "must specify at least one scraper")
}

func TestLoadInvalidConfig_InvalidScraperKey(t *testing.T) {
	factories, err := otelcoltest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[metadata.Type] = factory
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/33594
	// nolint:staticcheck
	_, err = otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "config-invalidscraperkey.yaml"), factories)

	require.Contains(t, err.Error(), "invalid scraper key: \"invalidscraperkey\"")
}

func TestConfig_Unmarshal(t *testing.T) {
	type fields struct {
		ControllerConfig     scraperhelper.ControllerConfig
		Scrapers             map[string]internal.Config
		MetricsBuilderConfig metadata.MetricsBuilderConfig
	}

	type args struct {
		componentParser *confmap.Conf
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Empty Component Parser",
			fields:  fields{},
			args:    args{componentParser: nil},
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{
				ControllerConfig:     test.fields.ControllerConfig,
				Scrapers:             test.fields.Scrapers,
				MetricsBuilderConfig: test.fields.MetricsBuilderConfig,
			}
			if err := cfg.Unmarshal(test.args.componentParser); (err != nil) != test.wantErr {
				t.Errorf("Config.Unmarshal() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
