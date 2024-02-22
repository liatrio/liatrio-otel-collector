// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpjsonreceiver // import "github.com/liatrio/liatrio-otel-collector/receiver/httpjson"

import (
	"errors"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const configKeyEndpoint = "endpoint"

var configKeyFields = "fields"

var (
	ErrMustNotNil = errors.New("sample interface must not be nil")
)

// Config that is exposed to this receiver through the OTEL config.yaml
type Config struct {
	confighttp.HTTPClientSettings           `mapstructure:",squash"`
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	Method                                  string
	Fields                                  map[string]interface{} `mapstructure:"fields"`
	Endpoint                                string                 `mapstructure:"endpoint"`
}

// Unmarshal a config.Parser into the config struct.
func (cfg *Config) Unmarshal(componentParser *confmap.Conf) error {
	if componentParser == nil {
		return nil
	}

	err := componentParser.Unmarshal(cfg)
	if err != nil {
		return err
	}

	// dynamically load the individual collector configs based on the key name
	if componentParser.IsSet(configKeyFields) {
		// use the value provided in the otel config.yaml
		value := componentParser.Get(configKeyFields).(map[string]interface{})
		if value == nil {
			if componentParser.Get(configKeyFields) == nil {
				return ErrMustNotNil
			}
		}
		cfg.Fields = value
	} else {
		// default value
		cfg.Fields = make(map[string]interface{})
	}

	if componentParser.IsSet(configKeyEndpoint) {
		value := componentParser.Get(configKeyEndpoint)
		if value.(string) == "" {
			return errors.New("URL Endpoint cannot be blank. value: " + value.(string))
		}
		cfg.Endpoint = value.(string)
	}

	return nil
}

// Validate the configuration passed through the OTEL config.yaml
func (cfg *Config) Validate() error {
	var err error = nil

	if cfg.Fields == nil {
		// err = multierr.Append(err, errors.New("sample config data is required"))
		err = ErrMustNotNil
	}

	return err
}
