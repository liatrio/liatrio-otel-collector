// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/githubactionsreceiver"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/multierr"
)

var errMissingEndpointFromConfig = errors.New("missing receiver server endpoint from config")

// Config defines configuration for GitHub Actions receiver
type Config struct {
	confighttp.ServerConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	Path                    string                   `mapstructure:"path"`                // path for data collection. Default is <host>:<port>/events
	Secret                  string                   `mapstructure:"secret"`              // github webhook hash signature. Default is empty
	CustomServiceName       string                   `mapstructure:"custom_service_name"` // custom service name. Default is empty
	ServiceNamePrefix       string                   `mapstructure:"service_name_prefix"` // service name prefix. Default is empty
	ServiceNameSuffix       string                   `mapstructure:"service_name_suffix"` // service name suffix. Default is empty
}

var _ component.Config = (*Config)(nil)

// Unmarshal decodes receiver settings, including fields beside the embedded server
// config. confighttp.ServerConfig implements confmap.Unmarshaler, and embedding
// promotes that method onto Config. confmap would then decode only the server
// config and drop path, secret, and the service-name fields.
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if conf == nil {
		return nil
	}
	return conf.Unmarshal(cfg, confmap.WithIgnoreUnused())
}

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	var errs error

	if cfg.NetAddr.Endpoint == "" {
		errs = multierr.Append(errs, errMissingEndpointFromConfig)
	}

	return errs
}
