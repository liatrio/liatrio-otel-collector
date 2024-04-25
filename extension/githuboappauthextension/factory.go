package githuboappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/auth"

	"github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension/internal/metadata"
)

// NewFactory creates a factory for the GitHub App Auth extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createExtension(_ context.Context, set extension.CreateSettings, cfg component.Config) (extension.Extension, error) {
	ga, err := newGitHubAppAuthenticator(cfg.(*Config), set.Logger)
	if err != nil {
		return nil, err
	}

    // TODO change new client with round trip and creds based on https://github.com/bradleyfalzon/ghinstallation
    return auth.NewClient()

    // return auth.Client(ga.client)
	 //return auth.NewClient(
	// 	auth.WithClientRoundTripper(ga.roundTripper),
	// 	auth.WithClientPerRPCCredentials(ga.perRPCCredentials),
	// ), nil
} 
