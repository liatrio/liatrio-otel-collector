package githubappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"go.uber.org/zap"
)

// githubAppAuthenticator provides a simple struct to contain a zap logger and
// a http client which will return a transport with a roundTripper when the
// extension is created. This transport is created by the ghinstallation
// package and provided to the OTEL auth extension package.
type githubAppAuthenticator struct {
	logger *zap.Logger
	client *http.Client
}

// newGitHubAppAuthenticator calls the ghinstallation package to create the
// roundTripper transport for use by the OTEL auth package.
func newGitHubAppAuthenticator(cfg *Config, logger *zap.Logger) (*githubAppAuthenticator, error) {
	trans := http.DefaultTransport

	a, err := ghinstallation.NewKeyFromFile(trans, cfg.GitHubAppID, cfg.GitHubAppInstId, cfg.GitHubAppPrivateKeyFile)
	if err != nil {
		logger.Sugar().Errorf("unable to create transport using private key: %v", zap.Error(err))
		return nil, err
	}

	return &githubAppAuthenticator{
		logger: logger,
		client: &http.Client{
			Transport: a,
		},
	}, nil

}

// The roundTripper function has to be defined in this way so that the createExtension
// function can return auth.NewClient() as a component. Thus we take the
// roundTripper created by the ghinstallation package, send the transport up,
// and pass through this function so that the extension can handle
// authentication.
func (g *githubAppAuthenticator) roundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return g.client.Transport, nil
}
