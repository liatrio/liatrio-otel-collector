package ldapreceiver

import (
	"fmt"
	"time"
)

// Configuration that is exposed to this ldapreceiver through the OTEL config.yaml
type Config struct {
	Interval string `mapstructure:"interval"`

	// TODO: either implement ASM authenticator, or leverage the basic auth extension (preferrable)
	//ASMEnabled bool   `mapstructure:"asm_enabled"`
	//UserSecretName string `mapstructure:"user_secret_name"`
	//PassSecretName string `mapstructure:"pass_secret_name"`
}

// Validate the configuration passed through the OTEL config.yaml
func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Seconds() < 10 {
		return fmt.Errorf("interval must be at least 10 seconds")
	}
	return nil
}
