package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"context"
	"github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"testing"
	"time"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	tc := []struct {
		desc string
		tf   func(*testing.T)
	}{
		{
			desc: "create new factory with the correct type",
			tf: func(t *testing.T) {
				f := NewFactory()
				assert.EqualValues(t, typeStr, f.Type())
			},
		},
		{
			desc: "create new factory with default config",
			tf: func(t *testing.T) {
				factory := NewFactory()

				var expectefCfg component.Config = &Config{
					ScraperControllerSettings: scraperhelper.ScraperControllerSettings{
						CollectionInterval: 10 * time.Second,
					},
					HTTPClientSettings: confighttp.HTTPClientSettings{
						Timeout: 15 * time.Second,
					},
					MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
				}
				assert.Equal(t, expectefCfg, factory.CreateDefaultConfig())
			},
		},
        {
            desc: "create new factory and metric receiver without returing an error",
            tf: func(t *testing.T) {
                factory := NewFactory()
                cfg := factory.CreateDefaultConfig()
                _, err := factory.CreateMetricsReceiver(
                    context.Background(), 
                    receivertest.NewNopCreateSettings(), 
                    cfg, 
                    consumertest.NewNop(),
                )
                assert.NoError(t, err)
            },
        },
        {
            desc: "create new factory and metric receiver returning errors and invalid config",
            tf: func(t *testing.T) {
                factory := NewFactory()
                _, err := factory.CreateMetricsReceiver(
                    context.Background(), 
                    receivertest.NewNopCreateSettings(), 
                    nil, 
                    consumertest.NewNop(),
                )
                assert.ErrorIs(t, err, ghConfigNotValid)


            },
        },
	}

	for _, tt := range tc {
		t.Run(tt.desc, tt.tf)
	}
}
