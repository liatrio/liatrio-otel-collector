package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"context"
	"time"

	"github.com/google/go-github/v50/github"
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

func ghClient(ghRcvr *ghReceiver) (client *github.Client){
	client = github.NewClient(nil)
	return
}

func (ghRcvr *ghReceiver) Start(ctx context.Context, host component.Host) error {
	ghRcvr.host = host
	ctx = context.Background()
	ctx, ghRcvr.cancel = context.WithCancel(ctx)

	interval, _ := time.ParseDuration(ghRcvr.config.Interval)

	go func() {
        ghClient := ghClient(ghRcvr)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ghRcvr.logger.Info("Proccessing GitHub metrics...")
                orgs, _, err := ghClient.Organizations.List(ctx, "adrielp", nil)
                if err != nil {
                    ghRcvr.logger.Error("Error getting organizations", zap.Error(err))
                }
                ghRcvr.logger.Info("Organizations", zap.Any("orgs", orgs))

                repos, _, err := ghClient.Repositories.List(ctx, "adrielp", nil)
                if err != nil {
                    ghRcvr.logger.Error("Error getting repositories", zap.Error(err))
                }
                ghRcvr.logger.Info("Repositories", zap.Any("repos", repos))
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
