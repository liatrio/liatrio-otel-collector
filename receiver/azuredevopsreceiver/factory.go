// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver // import "github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal"
	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper"
)

// This file implements a factory for the azuredevops receiver
const (
	defaultReadTimeout  = 500 * time.Millisecond
	defaultWriteTimeout = 500 * time.Millisecond
	defaultPath         = "/events"
	defaultHealthPath   = "/health"
	defaultEndpoint     = "localhost:8080"
)

var (
	scraperFactories = map[string]internal.ScraperFactory{
		azuredevopsscraper.TypeStr: &azuredevopsscraper.Factory{},
	}

	errConfigNotValid = errors.New("configuration is not valid for the azuredevops receiver")
)

// NewFactory creates a factory for the azuredevops receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

// Gets a factory for defined scraper.
func getScraperFactory(key string) (internal.ScraperFactory, bool) {
	if factory, ok := scraperFactories[key]; ok {
		return factory, true
	}

	return nil, false
}

// Create the default config based on the const(s) defined above.
func createDefaultConfig() component.Config {
	return &Config{
		ControllerConfig: scraperhelper.NewDefaultControllerConfig(),
		WebHook: WebHook{
			ServerConfig: confighttp.ServerConfig{
				Endpoint:     defaultEndpoint,
				ReadTimeout:  defaultReadTimeout,
				WriteTimeout: defaultWriteTimeout,
			},
			Path:       defaultPath,
			HealthPath: defaultHealthPath,
		},
	}
}

// Create the metrics receiver according to the OTEL conventions taking in the
// context, receiver params, configuration from the component, and consumer (process or exporter)
func createMetricsReceiver(
	ctx context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {

	// check that the configuration is valid
	conf, ok := cfg.(*Config)
	if !ok {
		return nil, errConfigNotValid
	}

	addScraperOpts, err := createAddScraperOpts(ctx, params, conf, scraperFactories)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewMetricsController(
		&conf.ControllerConfig,
		params,
		consumer,
		addScraperOpts...,
	)
}

// createTracesReceiver creates a trace receiver based on provided config.
func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	rCfg := cfg.(*Config)
	if nextConsumer == nil {
		return nil, errors.New("no nextConsumer provided")
	}
	return newTracesReceiver(set, rCfg, nextConsumer)
}

// createLogsReceiver creates a logs receiver based on provided config.
func createLogsReceiver(
	ctx context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	// check that the configuration is valid
	conf, ok := cfg.(*Config)
	if !ok {
		return nil, errConfigNotValid
	}

	// Create logs scraper
	var logsScraper scraper.Logs
	for key, scraperCfg := range conf.Scrapers {
		factory := scraperFactories[key]
		if factory == nil {
			return nil, fmt.Errorf("factory not found for scraper %q", key)
		}

		var err error
		logsScraper, err = factory.CreateLogsScraper(ctx, params, scraperCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create logs scraper %q: %w", key, err)
		}
		break // Only use first scraper for now
	}

	if logsScraper == nil {
		return nil, errors.New("no logs scraper configured")
	}

	// Create a simple logs receiver wrapper
	return &simpleLogsReceiver{
		scraper:      logsScraper,
		consumer:     consumer,
		settings:     params,
		initialDelay: conf.ControllerConfig.InitialDelay,
		interval:     conf.ControllerConfig.CollectionInterval,
	}, nil
}

func createAddScraperOpts(
	ctx context.Context,
	params receiver.Settings,
	cfg *Config,
	factories map[string]internal.ScraperFactory,
) ([]scraperhelper.ControllerOption, error) {
	scraperControllerOptions := make([]scraperhelper.ControllerOption, 0, len(cfg.Scrapers))

	for key, cfg := range cfg.Scrapers {
		azuredevopsscraper, err := createAzureDevOpsScraper(ctx, params, key, cfg, factories)

		if err != nil {
			return nil, fmt.Errorf("failed to create scraper %q: %w", key, err)
		}

		scraperControllerOptions = append(scraperControllerOptions, scraperhelper.AddScraper(metadata.Type, azuredevopsscraper))
	}

	return scraperControllerOptions, nil
}

func createAzureDevOpsScraper(
	ctx context.Context,
	params receiver.Settings,
	key string,
	cfg internal.Config,
	factories map[string]internal.ScraperFactory,
) (scraper scraper.Metrics, err error) {
	factory := factories[key]
	if factory == nil {
		return nil, fmt.Errorf("factory not found for scraper %q", key)
	}

	scraper, err = factory.CreateMetricsScraper(ctx, params, cfg)
	if err != nil {
		return nil, err
	}

	return
}
