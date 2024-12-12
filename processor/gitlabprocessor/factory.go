package gitlabprocessor

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/attraction"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterlog"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermetric"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterspan"
	"github.com/liatrio/liatrio-otel-collector/processor/gitlabprocessor/internal/metadata"
)

var (
	processorCapabilities = consumer.Capabilities{MutatesData: true}
	errConfigNotValid     = errors.New("configuration is not valid for the gitlab receiver")
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
	return &Config{}
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

    // attrProc, err := attraction.NewAttrProc()
    attrProc, err := attraction.NewAttrProc()
	if err != nil {
		return nil, err
	}
    

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		newLogProcessor(set.Logger, attrProc, *conf).processLogs,
		processorhelper.WithCapabilities(processorCapabilities))
}
