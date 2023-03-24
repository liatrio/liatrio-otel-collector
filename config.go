package ldapreceiver

import (
	"fmt"
	"time"
)

// Configuration that is exposed to this ldapreceiver through the OTEL config.yaml
type Config struct {
	Interval           string `mapstructure:"interval"`
	SearchFilter       string `mapstructure:"search_filter"`
	Endpoint           string `mapstructure:"endpoint"`
	BaseDN             string `mapstructure:"base_dn"`
	InsecureSkipVerify bool   `mapstructure:"ignore_tls"`

	// TODO: replace with basic auth through OTel configuration
	User string `mapstructure:"user"`
	Pw   string `mapstructure:"pw"`
}

// Validate the configuration passed through the OTEL config.yaml
func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Seconds() < 10 {
		return fmt.Errorf("interval must be at least 10 seconds")
	}

	if cfg.SearchFilter == "" {
		return fmt.Errorf("search_filter must be set")
	}

	if cfg.BaseDN == "" {
		return fmt.Errorf("base_dn must be set")
	}

	if cfg.Endpoint == "" {
		return fmt.Errorf("endpoint must be set")
	}
	return nil
}
