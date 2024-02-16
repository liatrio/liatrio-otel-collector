// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpjsonreceiver // import "github.com/liatrio/liatrio-otel-collector/receiver/httpjson"

import (
	"errors"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const configKey = "sample"

var (
	ErrMustNotNil = errors.New("sample interface must not be nil")
)

// Config that is exposed to this receiver through the OTEL config.yaml
type Config struct {
	confighttp.HTTPClientSettings           `mapstructure:",squash"`
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	fields                                  []string `mapstructure:"fields"`
}

// Unmarshal a config.Parser into the config struct.
func (cfg *Config) Unmarshal(componentParser *confmap.Conf) error {
	if componentParser == nil {
		return nil
	}

	// load the non-dynamic config normally
	err := componentParser.Unmarshal(cfg)
	if err != nil {
		return err
	}

	// dynamically load the individual collector configs based on the key name
	if componentParser.IsSet(configKey) {
		// use the value provided in the otel config.yaml
		value, ok := componentParser.Get(configKey).([]string)
		if !ok {
			if componentParser.Get(configKey) == nil {
				return ErrMustNotNil
			}
		}
		cfg.fields = value
	} else {
		// default value
		cfg.fields = []string{}
	}

	return nil
}

// Validate the configuration passed through the OTEL config.yaml
func (cfg *Config) Validate() error {
	var err error = nil

	if cfg.fields == nil {
		// err = multierr.Append(err, errors.New("sample config data is required"))
		err = ErrMustNotNil
	}

	return err
}
