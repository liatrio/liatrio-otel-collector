// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package backstagereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/backstagereceiver"

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/liatrio/liatrio-otel-collector/receiver/backstagereceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"go.einride.tech/backstage/catalog"
)

type backstagereceiver struct {
	setting         receiver.CreateSettings
	config          *Config
	stopperChanList []chan struct{}
	client          catalog.Client
	consumer        consumer.Logs
	obsrecv         *receiverhelper.ObsReport
	mu              sync.Mutex
	cancel          context.CancelFunc
}

func newReceiver(params receiver.CreateSettings, config *Config, consumer consumer.Logs) (receiver.Logs, error) {
	transport := "http"

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             params.ID,
		Transport:              transport,
		ReceiverCreateSettings: params,
	})
	if err != nil {
		return nil, err
	}

	return &backstagereceiver{
		setting:  params,
		consumer: consumer,
		config:   config,
		obsrecv:  obsrecv,
		mu:       sync.Mutex{},
	}, nil
}

func (kr *backstagereceiver) Start(ctx context.Context, _ component.Host) error {
	client, err := kr.config.getClient()
	if err != nil {
		return err
	}
	kr.client = client
	kr.setting.Logger.Info("Object Receiver started")

	cctx, cancel := context.WithCancel(ctx)
	kr.cancel = cancel

	for _, object := range kr.config.Objects {
		kr.start(cctx, object)
	}
	return nil
}

func (kr *backstagereceiver) Shutdown(context.Context) error {
	kr.setting.Logger.Info("Object Receiver stopped")
	if kr.cancel != nil {
		kr.cancel()
	}

	kr.mu.Lock()
	for _, stopperChan := range kr.stopperChanList {
		close(stopperChan)
	}
	kr.mu.Unlock()
	return nil
}

// TODO
func (kr *backstagereceiver) start(ctx context.Context, object *BackstageConfig) {

	filters := object.Filters

	if len(object.Namespaces) > 0 {
		filters = append(filters,
			fmt.Sprintf("namespace=%s", strings.Join(object.Namespaces, ",")),
		)
	}

	filters = append(filters,
		fmt.Sprintf("kind=%s", object.Kind),
		fmt.Sprintf("group=%s", object.Group),
	)

	obj := schema.GroupResource{
		Group:    object.Group,
		Resource: object.Kind,
	}

	kr.setting.Logger.Info("Started collecting", zap.Any("gvr", obj.String()), zap.Any("namespaces", object.Namespaces))

	go kr.startPull(ctx, object, filters, object.Fields)
}

// TODO pagination
func (kr *backstagereceiver) startPull(ctx context.Context, config *BackstageConfig, filters, fields []string) {
	stopperChan := make(chan struct{})
	kr.mu.Lock()
	kr.stopperChanList = append(kr.stopperChanList, stopperChan)
	kr.mu.Unlock()
	ticker := newTicker(ctx, config.Interval)

	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			objects, err := kr.client.ListEntities(ctx, &catalog.ListEntitiesRequest{
				Filters: filters,
				Fields:  fields,
			})

			if err != nil {
				kr.setting.Logger.Error("error in pulling object", zap.String("filters", strings.Join(filters, "&")), zap.Error(err))
			} else if len(objects.Entities) > 0 {
				logs := pullObjectsToLogData(objects.Entities, time.Now(), config)
				obsCtx := kr.obsrecv.StartLogsOp(ctx)
				logRecordCount := logs.LogRecordCount()
				err = kr.consumer.ConsumeLogs(obsCtx, logs)
				kr.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), logRecordCount, err)
			}
		case <-stopperChan:
			return
		}

	}

}

// Start ticking immediately.
// Ref: https://stackoverflow.com/questions/32705582/how-to-get-time-tick-to-tick-immediately
func newTicker(ctx context.Context, repeat time.Duration) *time.Ticker {
	ticker := time.NewTicker(repeat)
	oc := ticker.C
	nc := make(chan time.Time, 1)
	go func() {
		nc <- time.Now()
		for {
			select {
			case tm := <-oc:
				nc <- tm
			case <-ctx.Done():
				return
			}
		}
	}()

	ticker.C = nc
	return ticker
}
