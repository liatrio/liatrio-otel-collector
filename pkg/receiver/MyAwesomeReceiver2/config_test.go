// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package MyAwesomeReceiver

import (
	"path/filepath"
	"testing"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
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

	assert.Equal(t, len(cfg.Receivers), 1)

	r0 := cfg.Receivers[component.NewID(metadata.Type)]
	defaultConfig := factory.CreateDefaultConfig()

	assert.Equal(t, defaultConfig, r0)
}

////////////////// Testing Begins ///////////

func TestConfig_Unmarshal(t *testing.T) {
	type fields struct {
		ScraperControllerSettings scraperhelper.ScraperControllerSettings
		sample                    Sample
	}
	type args struct {
		componentParser *confmap.Conf
	}
	confMap, errCM := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	if errCM != nil {
		return
	}
	tests := []struct {
		name      string
		confMap   *confmap.Conf
		fields    fields
		args      args
		configKey string
		wantErr   error
	}{
		// TODO: Add test cases.
		{
			name:      "default configuration int",
			confMap:   confMap,
			fields:    fields{},
			args:      args{componentParser: confMap},
			configKey: ConfigKeySample,
			wantErr:   ErrMustString,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ScraperControllerSettings: tt.fields.ScraperControllerSettings,
				sample:                    tt.fields.sample,
			}
			err := cfg.Unmarshal(tt.args.componentParser)
			if err != tt.wantErr {
				t.Errorf("Config.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
