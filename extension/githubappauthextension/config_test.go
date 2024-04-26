package githuboappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension/internal/metadata"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id          component.ID
		expected    component.Config
		expectedErr error
	}{
		{
			id: component.NewID(metadata.Type),
			expected: &Config{
				GitHubAppID:             1234,
				GitHubAppInstId:         1234,
				GitHubAppPrivateKeyFile: "path/to/something",
			},
		},
		{
			id:          component.NewIDWithName(metadata.Type, "missingurl"),
			expectedErr: errNoGitHubAppIDProvided,
		},
		{
			id:          component.NewIDWithName(metadata.Type, "missingid"),
			expectedErr: errNoGitHubPrivateKeyProvided,
		},
	}
	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
			require.NoError(t, err)
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()
			sub, err := cm.Sub(tt.id.String())
			require.NoError(t, err)
			require.NoError(t, component.UnmarshalConfig(sub, cfg))
			if tt.expectedErr != nil {
				assert.ErrorIs(t, component.ValidateConfig(cfg), tt.expectedErr)
				return
			}
			assert.NoError(t, component.ValidateConfig(cfg))
			assert.Equal(t, tt.expected, cfg)
		})
	}
}
