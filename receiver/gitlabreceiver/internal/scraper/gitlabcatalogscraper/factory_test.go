package gitlabcatalogscraper

import (
	"context"
	"testing"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

var creationSet = receivertest.NewNopSettings(metadata.Type)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	assert.NotNil(t, cfg, "failed to create default config")

	typedCfg := cfg.(*Config)
	assert.Equal(t, 5, typedCfg.ConcurrencyLimit)
}

func TestCreateMetricsScraper(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	mReceiver, err := factory.CreateMetricsScraper(context.Background(), creationSet, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, mReceiver)
}
