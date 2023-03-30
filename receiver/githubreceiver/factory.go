package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

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
	typeStr         = "github"
	defaultInterval = 60 * time.Second
	//defaultIgnoreTLS = false
	defaultTimeout = 15 * time.Second
	stability      = component.StabilityLevelAlpha
)

var (
	ghConfigNotValid = errors.New("config is not a valid github receiver configuration")
)

// Create ethe default config based on the const(s) defined above.
func createDefaultConfig() component.Config {
	return &Config{
		Interval: fmt.Sprint(defaultInterval),
		//InsecureSkipVerify: defaultIgnoreTLS,
	}
}

// Create the metrics receiver according to the OTEL conventions taking in the
// context, receiver params, configuration from the component, and consumer (process or exporter)
func createMetricsReceiver(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Metrics) (receiver.Metrics, error) {

	// if the next consumer (processer or exporter) in the pipeline has an issue
	// or is passed as nil then through the next consumer error
	if consumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	ghCfg, ok := cfg.(*Config)
	if !ok {
		return nil, ghConfigNotValid
	}

	logger := params.Logger

	ghRcvr := &ghReceiver{
		logger:       logger,
		nextConsumer: consumer,
		config:       ghCfg,
	}

	return ghRcvr, nil
}

// NewFactory creates a factory for the githubreceiver according to OTEL's conventions
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, stability))
}
