// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package gitproviderreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/scraper/githubscraper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol/otelcoltest"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

func TestLoadConfig(t *testing.T) {
	factories, err := otelcoltest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[metadata.Type] = factory
	cfg, err := otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 2)

	r0 := cfg.Receivers[component.NewID(metadata.Type)]
	defaultConfigGitHubScraper := factory.CreateDefaultConfig()
	defaultConfigGitHubScraper.(*Config).Scrapers = map[string]internal.Config{
		githubscraper.TypeStr: (&githubscraper.Factory{}).CreateDefaultConfig(),
	}

	assert.Equal(t, defaultConfigGitHubScraper, r0)

	r1 := cfg.Receivers[component.NewIDWithName(metadata.Type, "customname")].(*Config)
	expectedConfig := &Config{
		ScraperControllerSettings: scraperhelper.ScraperControllerSettings{
			CollectionInterval: 30 * time.Second,
			InitialDelay:       1 * time.Second,
		},
		Scrapers: map[string]internal.Config{
			githubscraper.TypeStr: (&githubscraper.Factory{}).CreateDefaultConfig(),
		},
	}

	assert.Equal(t, expectedConfig, r1)
}

// func TestLoadConfig_CompnentParcerNil(t *testing.T) {
// 	factories, err := otelcoltest.NopFactories()
// 	require.NoError(t, err)

// 	factory := NewFactory()
// 	factories.Receivers[metadata.Type] = factory

// 	// Is this mocking?
// 	p := (*confmap.Conf)(nil)
// 	cfg := (*Config)(nil)
// 	err = cfg.Unmarshal(p)

// 	require.NoError(t, err)

// 	// assert.Equal(t, len(cfg.Receivers), 2)
// }

// // Mock componentParser.Unmarshal()
// func TestLoadConfig_CompnentParcerError(t *testing.T) {
// 	factories, err := otelcoltest.NopFactories()
// 	require.NoError(t, err)

// 	factory := NewFactory()
// 	factories.Receivers[metadata.Type] = factory

// 	// cfg, err := otelcoltest.LoadConfig(filepath.Join("testdata", "config-nilconfig.yaml"), factories)
// 	componentParser := (*confmap.Conf)
// 	cfg := (*Config)(nil)

// 	err = cfg.Unmarshal(componentParser)

// 	// require.Error(t, err)
// 	// require.NotNil(t, cfg)

// 	// assert.Equal(t, len(cfg.Receivers), 2)
// }

// func TestLoadInvalidConfig_NoScrapers(t *testing.T) {
// 	factories, err := otelcoltest.NopFactories()
// 	require.NoError(t, err)

// 	factory := NewFactory()
// 	factories.Receivers[metadata.Type] = factory
// 	_, err = otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "config-noscrapers.yaml"), factories)

// 	require.Contains(t, err.Error(), "must specify at least one scraper")
// }

func TestConfig_Unmarshal(t *testing.T) {
	// factories, err := otelcoltest.NopFactories()
	// require.NoError(t, err)

	// factory := NewFactory()
	// factories.Receivers[metadata.Type] = factory
	// cfg, err := otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "config.yaml"), factories)

	type fields struct {
		ScraperControllerSettings scraperhelper.ScraperControllerSettings
		Scrapers                  map[string]internal.Config
		MetricsBuilderConfig      metadata.MetricsBuilderConfig
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
		// {
		// 	name: "valid Compenent Parser",
		// 	fields: fields{},
		// 	args: args{ componentParser: notnil },
		// 	wantErr: false,
		// },
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{
				ScraperControllerSettings: test.fields.ScraperControllerSettings,
				Scrapers:                  test.fields.Scrapers,
				MetricsBuilderConfig:      test.fields.MetricsBuilderConfig,
			}
			if err := cfg.Unmarshal(test.args.componentParser); (err != nil) != test.wantErr {
				t.Errorf("Config.Unmarshal() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
