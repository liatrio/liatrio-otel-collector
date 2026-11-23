package gitlabprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
)

// Starting at only adding clientconfig. For calls, auth is required.
type Config struct {
	confighttp.ClientConfig `mapstructure:",squash"`
	Token                   string `mapstructure:"token"`
}

var _ component.Config = (*Config)(nil)

// Unmarshal decodes processor settings, including fields beside the embedded client
// config. confighttp.ClientConfig implements confmap.Unmarshaler, and embedding
// promotes that method onto Config. confmap would then decode only the client
// config and drop token.
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if conf == nil {
		return nil
	}
	return conf.Unmarshal(cfg, confmap.WithIgnoreUnused())
}

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Token == "" {
		return errors.New("authentication config for GitLab is required")
	}
	// if cfg.Auth == nil {
	// 	return errors.New("authentication config for GitLab is required")
	// }
	return nil
}
