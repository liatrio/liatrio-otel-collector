package githubappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"
)

func TestCreateDefaultConfig(t *testing.T) {
	expected := &Config{}
	cfg := createDefaultConfig()

	assert.Equal(t, expected, cfg)
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateExtension(t *testing.T) {
	cfg := createDefaultConfig().(*Config)

	tests := []struct {
		name        string
		settings    *Config
		shouldError bool
		expectedErr error
	}{
		{
			name: "valid_settings",
			settings: &Config{
				GitHubAppID:             1234,
				GitHubAppInstId:         1234,
				GitHubAppPrivateKeyFile: "./testdata/test-key.pem",
			},
			shouldError: false,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			cfg.GitHubAppID = testcase.settings.GitHubAppID
			cfg.GitHubAppInstId = testcase.settings.GitHubAppInstId
			cfg.GitHubAppPrivateKeyFile = testcase.settings.GitHubAppPrivateKeyFile

			ext, err := createExtension(context.Background(), extensiontest.NewNopCreateSettings(), cfg)
			if testcase.shouldError {
				assert.Error(t, err)
				assert.Nil(t, ext)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ext)
			}
		})
	}
}

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	assert.NotNil(t, f)
}
