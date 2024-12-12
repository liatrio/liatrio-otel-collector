package gitlabprocessor 

import (
	"errors"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/component"
)

// Starting at only adding clientconfig. For calls, auth is required.
type Config struct {
	confighttp.ClientConfig       `mapstructure:",squash"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Auth == nil {
		return errors.New("authentication config for GitLab is required")
	}
	return nil
}



