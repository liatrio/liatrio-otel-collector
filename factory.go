package ldapreceiver // import "github.com/liatrio/ldapreceiver"

import (
	"context"
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
	return nil, nil
}

// NewFactory creates a factory for the ldapreceiver according to OTEL's conventions
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, stability))
}
