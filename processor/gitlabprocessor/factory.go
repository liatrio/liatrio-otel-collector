package pipelineprocessor

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabprocessor/internal/metadata"
)

var (
	processorCapabilities = consumer.Capabilities{MutatesData: true}
	errConfigNotValid     = errors.New("configuration is not valid for the pipeline processor")
)

const (
	defaultHTTPTimeout = 15 * time.Second
)

// NewFactory returns a new factory for the GitLab Pipelines processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability))
}

// Note: This isn't a valid configuration because the processor would do no
// work.
func createDefaultConfig() component.Config {
	return &Config{
		ClientConfig: confighttp.ClientConfig{
			Timeout: defaultHTTPTimeout,
		},
	}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	conf, ok := cfg.(*Config)
	if !ok {
		return nil, errConfigNotValid
	}

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		newLogProcessor(ctx, set.Logger, conf).processLogs,
		processorhelper.WithCapabilities(processorCapabilities))
}
