package githuboappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

var (
	errNoGitHubAppIDProvided      = errors.New("no GitHub App ID provided in the GitHub App Auth extension configuration")
	errNoGitHubPrivateKeyProvided = errors.New("no GitHub App Private Key provided in the GitHub App Auth extension configuration")
)

// Config store the configuration for the GitHub App Installation flow. See:
// https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation
type Config struct {

    // GitHubAppID is the ID of the GitHub App.
	GitHubAppID string `mapstructure:"client_id"`

    // GitHubAppPrivateKeyFile is the file path to the private key generated
    // for the GitHub App.
	GitHubAppPrivateKeyFile string `mapstructure:"client_secret_file"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.GitHubAppID == "" {
		return errNoGitHubAppIDProvided
	}
	if cfg.GitHubAppPrivateKeyFile == "" {
		return errNoGitHubPrivateKeyProvided
	}
	return nil
}
