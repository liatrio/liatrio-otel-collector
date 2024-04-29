package githubappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGitHubAppAuthSettings(t *testing.T) {
	tests := []struct {
		name          string
		settings      *Config
		shouldError   bool
		expectedError string
	}{
		{
			name: "all_valid_settings",
			settings: &Config{
				GitHubAppID:             1234,
				GitHubAppInstId:         1234,
				GitHubAppPrivateKeyFile: "testdata/test-key.pem",
			},
			shouldError:   false,
			expectedError: "",
		},
		{
			name: "invalid_settings",
			settings: &Config{
				GitHubAppID:             1234,
				GitHubAppInstId:         1234,
				GitHubAppPrivateKeyFile: "doesnotexist.pem",
			},
			shouldError:   true,
			expectedError: "could not read private key: open doesnotexist.pem: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newGitHubAppAuthenticator(test.settings, zap.NewNop())
			if test.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedError)
				return
			}
			assert.NoError(t, err)
		})
	}
}
