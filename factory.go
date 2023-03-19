package ldapreceiver // import "github.com/liatrio/ldapreceiver"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr         = "ldap"
	defaultInterval = 10 * time.Second
	defaultTimeout  = 10 * time.Second
	stability       = component.StabilityLevelAlpha
)

var (
	ldapConfigNotValid = errors.New("config is not a valid ldap receiver configuration")
)

func createDefaultConfig() component.Config {
	return &Config{
		Interval: fmt.Sprint(defaultInterval),
	}
}

func createMetricsReceiver(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	// if the next consumer (processer or exporter) in the pipeline has an issue
	// or is passed as nil then through the next consumer error
	if consumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	ldapCfg, ok := cfg.(*Config)
	if !ok {
		return nil, ldapConfigNotValid
	}

	logger := params.Logger

	ldapRcvr := &ldapReceiver{
		logger:       logger,
		nextConsumer: consumer,
		config:       ldapCfg,
	}

	//httpcheckScraper := newScraper(cfg, params)
	//scraper, err := scraperhelper.NewScraper(typeStr, httpcheckScraper.scrape, scraperhelper.WithStart(httpcheckScraper.start))
	//if err != nil {
	//	return nil, err
	//}

	//return scraperhelper.NewScraperControllerReceiver(&cfg.ScraperControllerSettings, params, consumer, scraperhelper.AddScraper(scraper))

	return ldapRcvr, nil
}

// NewFactory creates a factory for the ldapreceiver according to OTEL's conventions
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, stability))
}
