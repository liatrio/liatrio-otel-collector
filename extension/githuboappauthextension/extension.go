package githuboappauthextension // import "github.com/liatrio/liatrio-otel-collector/extension/githubappauthextension"

import (
	"net/http"

	"go.uber.org/zap"
    "github.com/bradleyfalzon/ghinstallation/v2"
)

// githubAppAuthenticator provides a simple struct to contain an http client 
// with transport created by the ghinstallation library.
type githubAppAuthenticator struct {
	logger *zap.Logger
    transport *ghinstallation.Transport
    // roundTripper *http.RoundTripper
	client *http.Client
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

	// return &githubAppAuthenticator{
	// 	logger: logger,
	// 	client: &http.Client{
	// 		Transport: a,
	// 	},
	// }, nil
}
