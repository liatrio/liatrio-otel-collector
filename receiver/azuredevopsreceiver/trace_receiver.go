// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
)

var errMissingEndpoint = errors.New("missing a receiver endpoint")

const healthyResponse = `{"text": "Azure DevOps receiver webhook is healthy"}`

type azuredevopsTracesReceiver struct {
	traceConsumer consumer.Traces
	cfg           *Config
	server        *http.Server
	shutdownWG    sync.WaitGroup
	settings      receiver.Settings
	logger        *zap.Logger
	obsrecv       *receiverhelper.ObsReport
}

func newTracesReceiver(
	params receiver.Settings,
	config *Config,
	traceConsumer consumer.Traces,
) (*azuredevopsTracesReceiver, error) {
	if config.WebHook.Endpoint == "" {
		return nil, errMissingEndpoint
	}

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             params.ID,
		Transport:              "http",
		ReceiverCreateSettings: params,
	})

	if err != nil {
		return nil, err
	}

	atr := &azuredevopsTracesReceiver{
		traceConsumer: traceConsumer,
		cfg:           config,
		settings:      params,
		logger:        params.Logger,
		obsrecv:       obsrecv,
	}

	return atr, nil
}

func (atr *azuredevopsTracesReceiver) Start(ctx context.Context, host component.Host) error {
	endpoint := fmt.Sprintf("%s%s", atr.cfg.WebHook.ServerConfig.Endpoint, atr.cfg.WebHook.Path)
	atr.logger.Info("Starting Azure DevOps WebHook receiving server", zap.String("endpoint", endpoint))

	// noop if not nil. if start has not been called before these values should be nil.
	if atr.server != nil && atr.server.Handler != nil {
		return nil
	}

	ln, err := atr.cfg.WebHook.ServerConfig.ToListener(ctx)
	if err != nil {
		return err
	}

	// use gorilla mux to set up a router
	router := mux.NewRouter()

	// setup health route
	router.HandleFunc(atr.cfg.WebHook.HealthPath, atr.handleHealthCheck)

	// setup webhook route for traces
	router.HandleFunc(atr.cfg.WebHook.Path, atr.handleReq)

	atr.server, err = atr.cfg.WebHook.ServerConfig.ToServer(ctx, host, atr.settings.TelemetrySettings, router)
	if err != nil {
		return err
	}

	atr.logger.Info("Health check now listening at", zap.String("health_path", atr.cfg.WebHook.HealthPath))

	atr.shutdownWG.Add(1)
	go func() {
		defer atr.shutdownWG.Done()
		if errHTTP := atr.server.Serve(ln); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errHTTP))
		}
	}()

	return nil
}

func (atr *azuredevopsTracesReceiver) Shutdown(_ context.Context) error {
	if atr.server == nil {
		return nil
	}

	err := atr.server.Close()
	atr.shutdownWG.Wait()
	return err
}

func (atr *azuredevopsTracesReceiver) handleReq(w http.ResponseWriter, req *http.Request) {
	ctx := atr.obsrecv.StartTracesOp(req.Context())

	// Validate request path
	if req.URL.Path != atr.cfg.WebHook.Path {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		atr.logger.Sugar().Debugf("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Validate the payload using the configured secret (Azure DevOps style)
	if atr.cfg.WebHook.Secret != "" {
		if !atr.validateAzureDevOpsSignature(body, req.Header.Get("X-Hub-Signature-256")) {
			atr.logger.Debug("Payload validation failed")
			http.Error(w, "Invalid payload or signature", http.StatusBadRequest)
			return
		}
	}

	// Parse the webhook payload based on event type
	event, err := atr.parseAzureDevOpsWebhook(body)
	if err != nil {
		atr.logger.Sugar().Debugf("Webhook parsing failed", zap.Error(err))
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	// Convert event to traces based on type
	var td ptrace.Traces
	switch e := event.(type) {
	case *PipelineRunStateChangedEvent:
		if e.Resource.Run.State != "completed" {
			atr.logger.Debug("pipeline run not complete, skipping...", zap.String("state", e.Resource.Run.State))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		td, err = atr.handlePipelineEvent(e)
	case *PipelineStageStateChangedEvent:
		if e.Resource.Stage.State != "completed" {
			atr.logger.Debug("pipeline stage not complete, skipping...", zap.String("state", e.Resource.Stage.State))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		td, err = atr.handleStageEvent(e)
	case *PipelineJobStateChangedEvent:
		if e.Resource.Job.State != "completed" {
			atr.logger.Debug("pipeline job not complete, skipping...", zap.String("state", e.Resource.Job.State))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		td, err = atr.handleJobEvent(e)
	default:
		atr.logger.Sugar().Debug("event type not supported")
		http.Error(w, "event type not supported", http.StatusBadRequest)
		return
	}

	if td.SpanCount() > 0 {
		err = atr.traceConsumer.ConsumeTraces(ctx, td)
		if err != nil {
			http.Error(w, "failed to consume traces", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)

	atr.obsrecv.EndTracesOp(ctx, "protobuf", td.SpanCount(), err)
}

// validateAzureDevOpsSignature validates the webhook signature from Azure DevOps
func (atr *azuredevopsTracesReceiver) validateAzureDevOpsSignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")

	// Create HMAC hash
	mac := hmac.New(sha256.New, []byte(atr.cfg.WebHook.Secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// parseAzureDevOpsWebhook parses Azure DevOps webhook payloads based on event type
func (atr *azuredevopsTracesReceiver) parseAzureDevOpsWebhook(payload []byte) (interface{}, error) {
	// First, extract the event type from the payload
	var eventTypeExtractor struct {
		EventType string `json:"eventType"`
	}

	if err := json.Unmarshal(payload, &eventTypeExtractor); err != nil {
		return nil, fmt.Errorf("failed to extract event type from payload: %w", err)
	}

	eventType := eventTypeExtractor.EventType
	if eventType == "" {
		return nil, fmt.Errorf("event type not found in payload")
	}

	switch eventType {
	case "ms.vss-pipelines.run-state-changed-event":
		var event PipelineRunStateChangedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("failed to parse pipeline run state changed event: %w", err)
		}
		return &event, nil
	case "ms.vss-pipelines.stage-state-changed-event":
		var event PipelineStageStateChangedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("failed to parse pipeline stage state changed event: %w", err)
		}
		return &event, nil
	case "ms.vss-pipelines.job-state-changed-event":
		var event PipelineJobStateChangedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("failed to parse pipeline job state changed event: %w", err)
		}
		return &event, nil
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// Simple healthcheck endpoint.
func (atr *azuredevopsTracesReceiver) handleHealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(healthyResponse)) //nolint:all
}
