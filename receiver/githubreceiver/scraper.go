package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type ghReceiver struct {
	host         component.Host
	cancel       context.CancelFunc
	logger       *zap.Logger
	nextConsumer consumer.Metrics
	config       *Config
}

func (ghRcvr *ghReceiver) Start(ctx context.Context, host component.Host) error {
	ghRcvr.host = host
	ctx = context.Background()
	ctx, ghRcvr.cancel = context.WithCancel(ctx)

	interval, _ := time.ParseDuration(ghRcvr.config.Interval)

	go func() {
		//ghConn := ghClient(ghRcvr)
		//defer ghConn.Close()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ghRcvr.logger.Info("Proccessing GitHub metrics...")
				//getResults(ghConn, ghRcvr)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (ghRcvr *ghReceiver) Shutdown(ctx context.Context) error {
	ghRcvr.cancel()
	return nil
}
