package githuboappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"go.uber.org/zap"
)

// githubAppAuthenticator provides a simple struct to contain an http client
// with transport created by the ghinstallation library.
type githubAppAuthenticator struct {
	logger    *zap.Logger
	transport *ghinstallation.Transport
	client    *http.Client
}

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

// TODO update this comment with additional details for contextual accuracy.
// This is a wrapper function due to the requirements for the extension/auth
// package requiring a function be passed to NewClient() whereas ghinstallation
// auto creates the client but can't be returned as an extension component.
func (g *githubAppAuthenticator) roundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return g.client.Transport, nil
}
