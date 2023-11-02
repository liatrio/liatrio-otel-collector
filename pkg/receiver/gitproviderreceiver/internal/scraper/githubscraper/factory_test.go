package githubscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

var creationSet = receivertest.NewNopCreateSettings()

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	assert.NotNil(t, defaultConfig, "failed to create default config")
}

func TestCreateMetricsScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	mReceiver, err := factory.CreateMetricsScraper(context.Background(), creationSet, defaultConfig)
	assert.NoError(t, err)
	assert.NotNil(t, mReceiver)
}
