package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"fmt"
	"time"
)

// Configuration that is exposed to this github receiver through the OTEL config.yaml
type Config struct {
	Interval  string `mapstructure:"interval"`
	GitHubOrg string `mapstructure:"github_org"`
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
