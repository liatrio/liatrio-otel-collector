// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// loadPipelineRunEvent loads and parses the example pipeline event JSON
func loadPipelineRunEvent(t *testing.T) *PipelineRunStateChangedEvent {
	data, err := os.ReadFile("testdata/example-pipeline-event.json")
	require.NoError(t, err)

	var event PipelineRunStateChangedEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	return &event
}

// loadStageEvent loads and parses the example stage event JSON
func loadStageEvent(t *testing.T) *PipelineStageStateChangedEvent {
	data, err := os.ReadFile("testdata/example-stage-event.json")
	require.NoError(t, err)

	var event PipelineStageStateChangedEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	return &event
}

// loadJobEvent loads and parses the example job event JSON
func loadJobEvent(t *testing.T) *PipelineJobStateChangedEvent {
	data, err := os.ReadFile("testdata/example-job-event.json")
	require.NoError(t, err)

	var event PipelineJobStateChangedEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	return &event
}
