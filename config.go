package ldapreceiver

import (
	"fmt"
	"time"
)

//import (
//	"go.opentelemetry.io/collector/receiver/scraperhelper"
//)

type Config struct {
	Interval       string `mapstructure:"interval"`
	ASMEnabled     bool   `mapstructure:"asm_enabled"`
	UserSecretName string `mapstructure:"user_secret_name"`
	PassSecretName string `mapstructure:"pass_secret_name"`
}

func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Seconds() < 10 {
		//if cfg.Interval < 10 {
		return fmt.Errorf("interval must be at least 10 seconds")
	}
	return nil
}
