// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// handlePipelineRunStateChanged handles the creation of spans for Azure DevOps Pipeline Run events
func (atr *azuredevopsTracesReceiver) handlePipelineRunStateChanged(e *PipelineRunStateChangedEvent) (ptrace.Traces, error) {
	t := ptrace.NewTraces()
	r := t.ResourceSpans().AppendEmpty()

	resource := r.Resource()

	err := atr.getPipelineRunStateChangedAttrs(resource, e)
	if err != nil {
		atr.logger.Sugar().Error("failed to get pipeline run attributes", zap.Error(err))
		return ptrace.Traces{}, fmt.Errorf("failed to get pipeline run attributes: %w", err)
	}

	traceID, err := newTraceID(int64(e.Resource.Run.Pipeline.ID), 1)
	if err != nil {
		atr.logger.Sugar().Error("failed to generate trace ID", zap.Error(err))
		return ptrace.Traces{}, fmt.Errorf("failed to generate trace ID: %w", err)
	}

	err = atr.createPipelineRunRootSpan(r, e, traceID)
	if err != nil {
		atr.logger.Sugar().Error("failed to create pipeline run root span", zap.Error(err))
		return ptrace.Traces{}, errors.New("failed to create pipeline run root span")
	}
	return t, nil
}

// handlePipelineStageStateChanged handles the creation of spans for Azure DevOps Pipeline Stage events
func (atr *azuredevopsTracesReceiver) handlePipelineStageStateChanged(e *PipelineStageStateChangedEvent) (ptrace.Traces, error) {
	t := ptrace.NewTraces()
	r := t.ResourceSpans().AppendEmpty()

	resource := r.Resource()

	err := atr.getPipelineStageStateChangedAttrs(resource, e)
	if err != nil {
		atr.logger.Sugar().Error("failed to get pipeline stage attributes", zap.Error(err))
		return ptrace.Traces{}, fmt.Errorf("failed to get pipeline stage attributes: %w", err)
	}

	traceID, err := newTraceID(int64(e.Resource.Run.Pipeline.ID), 1)
	if err != nil {
		atr.logger.Sugar().Error("failed to generate trace ID", zap.Error(err))
		return ptrace.Traces{}, fmt.Errorf("failed to generate trace ID: %w", err)
	}

	err = atr.createPipelineStageRootSpan(r, e, traceID)
	if err != nil {
		atr.logger.Sugar().Error("failed to create pipeline stage root span", zap.Error(err))
		return ptrace.Traces{}, errors.New("failed to create pipeline stage root span")
	}
	return t, nil
}

// handlePipelineJobStateChanged handles the creation of spans for Azure DevOps Pipeline Job events
func (atr *azuredevopsTracesReceiver) handlePipelineJobStateChanged(e *PipelineJobStateChangedEvent) (ptrace.Traces, error) {
	t := ptrace.NewTraces()
	r := t.ResourceSpans().AppendEmpty()

	resource := r.Resource()

	err := atr.getPipelineJobStateChangedAttrs(resource, e)
	if err != nil {
		atr.logger.Sugar().Error("failed to get pipeline job attributes", zap.Error(err))
		return ptrace.Traces{}, fmt.Errorf("failed to get pipeline job attributes: %w", err)
	}

	traceID, err := newTraceID(int64(e.Resource.Run.Pipeline.ID), 1)
	if err != nil {
		atr.logger.Sugar().Error("failed to generate trace ID", zap.Error(err))
		return ptrace.Traces{}, fmt.Errorf("failed to generate trace ID: %w", err)
	}

	err = atr.createPipelineJobRootSpan(r, e, traceID)
	if err != nil {
		atr.logger.Sugar().Error("failed to create pipeline job root span", zap.Error(err))
		return ptrace.Traces{}, errors.New("failed to create pipeline job root span")
	}
	return t, nil
}

// createPipelineRunRootSpan creates a root span based on the provided pipeline run event, associated
// with the deterministic traceID.
func (atr *azuredevopsTracesReceiver) createPipelineRunRootSpan(
	resourceSpans ptrace.ResourceSpans,
	event *PipelineRunStateChangedEvent,
	traceID pcommon.TraceID,
) error {
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetTraceID(traceID)
	spanID, err := generatePipelineSpanID(int64(event.Resource.Run.Pipeline.ID))
	if err != nil {
		atr.logger.Sugar().Error("failed to generate pipeline span ID", zap.Error(err))
		return fmt.Errorf("failed to generate pipeline span ID: %w", err)
	}
	span.SetSpanID(spanID)
	span.SetName(fmt.Sprintf("Pipeline Run: %s", event.Resource.Run.Pipeline.Name))
	span.SetKind(ptrace.SpanKindInternal)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(event.Resource.Run.CreatedDate))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(event.Resource.Run.FinishedDate))

	attrs := span.Attributes()
	attrs.PutStr("azuredevops.pipeline.name", event.Resource.Run.Pipeline.Name)
	attrs.PutInt("azuredevops.pipeline.id", int64(event.Resource.Run.Pipeline.ID))
	attrs.PutStr("azuredevops.pipeline.run.state", event.Resource.Run.State)
	attrs.PutStr("azuredevops.pipeline.run.result", event.Resource.Run.Result)
	attrs.PutStr("azuredevops.pipeline.run.url", transformAzureDevOpsURL(event.Resource.Run.URL))

	return nil
}

// createPipelineStageRootSpan creates a root span based on the provided pipeline stage event, associated
// with the deterministic traceID.
func (atr *azuredevopsTracesReceiver) createPipelineStageRootSpan(
	resourceSpans ptrace.ResourceSpans,
	event *PipelineStageStateChangedEvent,
	traceID pcommon.TraceID,
) error {
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetTraceID(traceID)
	parentSpanId, err := generatePipelineSpanID(int64(event.Resource.Run.Pipeline.ID))
	if err != nil {
		atr.logger.Sugar().Error("failed to generate parent span ID", zap.Error(err))
		return fmt.Errorf("failed to generate parent span ID: %w", err)
	}
	spanID, err := generateStageSpanID(event.Resource.Stage.ID)
	if err != nil {
		atr.logger.Sugar().Error("failed to generate stage span ID", zap.Error(err))
		return fmt.Errorf("failed to generate stage span ID: %w", err)
	}
	span.SetSpanID(spanID)
	span.SetParentSpanID(parentSpanId)
	span.SetName(fmt.Sprintf("Pipeline Stage: %s", event.Resource.Stage.Name))
	span.SetKind(ptrace.SpanKindInternal)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(event.Resource.Run.CreatedDate))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(event.Resource.Run.FinishedDate))

	attrs := span.Attributes()
	attrs.PutStr("azuredevops.pipeline.name", event.Resource.Pipeline.Name)
	attrs.PutInt("azuredevops.pipeline.id", int64(event.Resource.Pipeline.ID))
	attrs.PutStr("azuredevops.pipeline.stage.name", event.Resource.Stage.Name)
	attrs.PutStr("azuredevops.pipeline.stage.state", event.Resource.Stage.State)
	attrs.PutStr("azuredevops.pipeline.stage.result", event.Resource.Stage.Result)

	return nil
}

// createPipelineJobRootSpan creates a root span based on the provided pipeline job event, associated
// with the deterministic traceID.
func (atr *azuredevopsTracesReceiver) createPipelineJobRootSpan(
	resourceSpans ptrace.ResourceSpans,
	event *PipelineJobStateChangedEvent,
	traceID pcommon.TraceID,
) error {
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	span := scopeSpans.Spans().AppendEmpty()

	span.SetTraceID(traceID)
	parentSpanId, err := generateStageSpanID(event.Resource.Stage.ID)
	if err != nil {
		atr.logger.Sugar().Error("failed to generate parent span ID", zap.Error(err))
		return fmt.Errorf("failed to generate parent span ID: %w", err)
	}
	spanID, err := generateJobSpanID(int64(event.Resource.Run.ID), event.Resource.Job.Attempt, event.Resource.Job.Name)
	if err != nil {
		atr.logger.Sugar().Error("failed to generate job span ID", zap.Error(err))
		return fmt.Errorf("failed to generate job span ID: %w", err)
	}
	span.SetSpanID(spanID)
	span.SetParentSpanID(parentSpanId)
	span.SetName(fmt.Sprintf("Pipeline Job: %s", event.Resource.Job.Name))
	span.SetKind(ptrace.SpanKindInternal)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(event.Resource.Job.StartTime))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(event.Resource.Job.FinishTime))

	attrs := span.Attributes()
	attrs.PutStr("azuredevops.pipeline.name", event.Resource.Pipeline.Name)
	attrs.PutInt("azuredevops.pipeline.id", int64(event.Resource.Pipeline.ID))
	attrs.PutStr("azuredevops.pipeline.job.name", event.Resource.Job.Name)
	attrs.PutStr("azuredevops.pipeline.job.state", event.Resource.Job.State)
	attrs.PutStr("azuredevops.pipeline.job.result", event.Resource.Job.Result)
	attrs.PutStr("azuredevops.pipeline.stage.name", event.Resource.Stage.Name)
	attrs.PutInt("azuredevops.pipeline.job.attempt", int64(event.Resource.Job.Attempt))

	return nil
}

func transformAzureDevOpsURL(apiURL string) string {
	// Transform Azure DevOps API URLs to web URLs
	htmlURL := strings.Replace(apiURL, "/_apis/", "/", 1)
	return htmlURL
}

// newTraceID creates a deterministic Trace ID based on the provided inputs of
// runID and runAttempt. `t` is appended to the end of the input to
// differentiate between a deterministic traceID and the parentSpanID.
func newTraceID(runID int64, runAttempt int) (pcommon.TraceID, error) {
	input := fmt.Sprintf("%d%dt", runID, runAttempt)
	// TODO: Determine if this is the best hashing algorithm to use. This is
	// more likely to generate a unique hash compared to MD5 or SHA1. Could
	// alternatively use UUID library to generate a unique ID by also using a
	// hash.
	hash := sha256.Sum256([]byte(input))
	idHex := hex.EncodeToString(hash[:])

	var id pcommon.TraceID
	_, err := hex.Decode(id[:], []byte(idHex[:32]))
	if err != nil {
		return pcommon.TraceID{}, err
	}

	return id, nil
}

func generatePipelineSpanID(pipelineID int64) (pcommon.SpanID, error) {
	input := fmt.Sprintf("pipeline_%d", pipelineID)
	hash := sha256.Sum256([]byte(input))
	spanIDHex := hex.EncodeToString(hash[:])

	var spanID pcommon.SpanID
	_, err := hex.Decode(spanID[:], []byte(spanIDHex[16:32]))
	if err != nil {
		return pcommon.SpanID{}, err
	}

	return spanID, nil
}

func generateStageSpanID(stageID string) (pcommon.SpanID, error) {
	input := fmt.Sprintf("stage_%s", stageID)
	hash := sha256.Sum256([]byte(input))
	spanIDHex := hex.EncodeToString(hash[:])

	var spanID pcommon.SpanID
	_, err := hex.Decode(spanID[:], []byte(spanIDHex[16:32]))
	if err != nil {
		return pcommon.SpanID{}, err
	}

	return spanID, nil
}

func generateJobSpanID(runID int64, runAttempt int, job string) (pcommon.SpanID, error) {
	input := fmt.Sprintf("%d%d%s", runID, runAttempt, job)
	hash := sha256.Sum256([]byte(input))
	spanIDHex := hex.EncodeToString(hash[:])

	var spanID pcommon.SpanID
	_, err := hex.Decode(spanID[:], []byte(spanIDHex[16:32]))
	if err != nil {
		return pcommon.SpanID{}, err
	}

	return spanID, nil
}
