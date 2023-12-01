// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package MyAwesomeReceiver // import "github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver"

import (
	"errors"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/multierr"
)

// ConfigKey List
const (
	ConfigKeySample = "receivers::MyAwesomeReceiver::sample"
)

// WantErr List
var (
	ErrMustString    = errors.New("sample.configuration must be a string")
	ErrSampleConfig  = errors.New("sample config data is required")
	ErrMustLowercase = errors.New("sample config data must be lowercase")
)

// Config that is exposed to this receiver through the OTEL config.yaml
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	sample                                  Sample `mapstructure:",squash"`
}

type Sample struct {
	configuration string
}

// setConfigKey

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
	if componentParser.IsSet(ConfigKeySample) {
		// use the value provided in the otel config.yaml
		value, ok := componentParser.Get(ConfigKeySample).(string)
		if !ok {
			return ErrMustString
		}
		cfg.sample.configuration = value
	} else {
		// default value
		cfg.sample.configuration = "data"
	}

	return nil
}

// Validate the configuration passed through the OTEL config.yaml
func (cfg *Config) Validate() error {
	var err error

	if cfg.sample.configuration == "" {
		err = multierr.Append(err, ErrSampleConfig)
	} else {
		if cfg.sample.configuration != strings.ToLower(cfg.sample.configuration) {
			err = multierr.Append(err, ErrMustLowercase)
		}
	}

	return err
}
