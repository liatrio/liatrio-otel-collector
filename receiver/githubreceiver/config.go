package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"fmt"
	"time"

	"github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver/internal/metadata"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

// Configuration that is exposed to this github receiver through the OTEL config.yaml
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	confighttp.HTTPClientSettings           `mapstructure:",squash"`
	//TODO: define this
	//MetricsBuilderConfig `mapstructure:",squash"`
	metadata.MetricsBuilderConfig `mapstructure:",squash"`
	Interval                      string `mapstructure:"interval"`
	GitHubOrg                     string `mapstructure:"github_org"`
	//Endpoint           string `mapstructure:"endpoint"`
}

// Validate the configuration passed through the OTEL config.yaml
func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Seconds() < 10 {
		return fmt.Errorf("interval must be at least 10 seconds")
	}

	//if cfg.GitHubOrg == "" {
	//	return fmt.Errorf("github_org must be set")
	//}
	return nil
}
