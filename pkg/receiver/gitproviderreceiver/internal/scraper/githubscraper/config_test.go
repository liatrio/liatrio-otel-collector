package githubscraper

import (
	// "path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
	// "github.com/stretchr/testify/require"
	// "go.opentelemetry.io/collector/otelcol/otelcoltest"
)

// TestConfig ensures a config created with the factory is the same as one created manually with
// the exported Config struct.
func TestConfig(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	expectedConfig := &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Timeout: 15 * time.Second,
		},
	}

	assert.Equal(t, expectedConfig, defaultConfig)
}
