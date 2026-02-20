// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// simpleLogsReceiver is a simple logs receiver that wraps a logs scraper
type simpleLogsReceiver struct {
	scraper interface {
		ScrapeLogs(ctx context.Context) (plog.Logs, error)
	}
	consumer     consumer.Logs
	settings     receiver.Settings
	initialDelay time.Duration
	interval     time.Duration
	cancel       context.CancelFunc
}

func (r *simpleLogsReceiver) Start(ctx context.Context, host component.Host) error {
	ctx, r.cancel = context.WithCancel(ctx)

	// Start the scraper if it has a Start method
	if starter, ok := r.scraper.(interface {
		Start(ctx context.Context, host component.Host) error
	}); ok {
		if err := starter.Start(ctx, host); err != nil {
			return err
		}
	}

	// Start the scraping loop
	go r.scrapeLoop(ctx)

	return nil
}

func (r *simpleLogsReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *simpleLogsReceiver) scrapeLoop(ctx context.Context) {
	// Wait for initial delay
	if r.initialDelay > 0 {
		select {
		case <-time.After(r.initialDelay):
		case <-ctx.Done():
			return
		}
	}

	// Create ticker for collection interval
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Scrape immediately, then on interval
	for {
		logs, err := r.scraper.ScrapeLogs(ctx)
		if err != nil {
			r.settings.Logger.Error("failed to scrape logs", zap.Error(err))
		} else if logs.LogRecordCount() > 0 {
			if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
				r.settings.Logger.Error("failed to consume logs", zap.Error(err))
			}
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}
