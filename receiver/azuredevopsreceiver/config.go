// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver // import "github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
)

const (
	scrapersKey = "scrapers"
)

// Config that is exposed to this github receiver through the OTEL config.yaml
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	Scrapers                       map[string]internal.Config `mapstructure:"scrapers"`
	metadata.MetricsBuilderConfig  `mapstructure:",squash"`
	WebHook                        WebHook `mapstructure:"webhook"`
}

type WebHook struct {
	confighttp.ServerConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	Path                    string                   `mapstructure:"path"`        // path for data collection. Default is /events
	HealthPath              string                   `mapstructure:"health_path"` // path for health check api. Default is /health_check
	// optional setting to set one or more required headers for all requests to have (except the health check)
	RequiredHeaders    map[string]configopaque.String `mapstructure:"required_headers"`
	AzureDevOpsHeaders AzureDevOpsHeaders             `mapstructure:",squash"` // GitLab headers set by default
	Secret             string                         `mapstructure:"secret"`  // secret for webhook
	ServiceName        string                         `mapstructure:"service_name"`
}

type AzureDevOpsHeaders struct {
	Customizable map[string]string `mapstructure:","` // can be overwritten via required_headers
	Fixed        map[string]string `mapstructure:","` // are not allowed to be overwritten
}

var _ component.Config = (*Config)(nil)
var _ confmap.Unmarshaler = (*Config)(nil)

// Validate the configuration passed through the OTEL config.yaml
func (cfg *Config) Validate() error {
	if len(cfg.Scrapers) == 0 {
		return errors.New("must specify at least one scraper")
	}
	return nil
}

// Unmarshal a config.Parser into the config struct.
func (cfg *Config) Unmarshal(componentParser *confmap.Conf) error {
	if componentParser == nil {
		return nil
	}

	// load the non-dynamic config normally
	err := componentParser.Unmarshal(cfg, confmap.WithIgnoreUnused())
	if err != nil {
		return err
	}

	// dynamically load the individual collector configs based on the key name

	cfg.Scrapers = map[string]internal.Config{}

	scrapersSection, err := componentParser.Sub(scrapersKey)
	if err != nil {
		return err
	}

	for key := range scrapersSection.ToStringMap() {
		factory, ok := getScraperFactory(key)
		if !ok {
			return fmt.Errorf("invalid scraper key: %q", key)
		}

		collectorCfg := factory.CreateDefaultConfig()
		collectorSection, err := scrapersSection.Sub(key)
		if err != nil {
			return err
		}

		err = collectorSection.Unmarshal(collectorCfg)
		if err != nil {
			return fmt.Errorf("error reading settings for scraper type %q: %w", key, err)
		}

		cfg.Scrapers[key] = collectorCfg
	}

	return nil
}
