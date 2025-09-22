// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineRunStateChangedEventUnmarshal(t *testing.T) {
	// Read the actual webhook JSON data
	data, err := os.ReadFile("testdata/example-pipeline-event.json")
	require.NoError(t, err)

	var event PipelineRunStateChangedEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	// Verify key fields are populated correctly
	assert.Equal(t, "ms.vss-pipelines.run-state-changed-event", event.EventType)
	assert.Equal(t, "pipelines", event.PublisherID)
	assert.Equal(t, "5a9cc47d-2818-4a4a-b99d-db0a28f4e08f", event.Resource.ProjectID)
	assert.Equal(t, int64(5152), event.Resource.Run.ID)
	assert.Equal(t, "liatrio.azure-pipelines", event.Resource.Run.Pipeline.Name)
	assert.Equal(t, "inProgress", event.Resource.Run.State)
	assert.NotNil(t, event.Resource.Run.CreatedDate)
	assert.Nil(t, event.Resource.Run.FinishedDate) // Should be nil for in-progress run
}

func TestPipelineStageStateChangedEventUnmarshal(t *testing.T) {
	// Read the actual webhook JSON data
	data, err := os.ReadFile("testdata/example-stage-event.json")
	require.NoError(t, err)

	var event PipelineStageStateChangedEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	// Verify key fields are populated correctly
	assert.Equal(t, "ms.vss-pipelines.stage-state-changed-event", event.EventType)
	assert.Equal(t, "pipelines", event.PublisherID)
	assert.Equal(t, "Deploy", event.Resource.Stage.Name)
	assert.Equal(t, "Deployment Stage", event.Resource.Stage.DisplayName)
	assert.Equal(t, "completed", event.Resource.Stage.State)
	assert.Equal(t, "succeeded", event.Resource.Stage.Result)
	assert.NotNil(t, event.Resource.Stage.StartTime)
	assert.NotNil(t, event.Resource.Stage.FinishTime)
}

func TestPipelineJobStateChangedEventUnmarshal(t *testing.T) {
	// Read the actual webhook JSON data
	data, err := os.ReadFile("testdata/example-job-event.json")
	require.NoError(t, err)

	var event PipelineJobStateChangedEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	// Verify key fields are populated correctly
	assert.Equal(t, "ms.vss-pipelines.job-state-changed-event", event.EventType)
	assert.Equal(t, "pipelines", event.PublisherID)
	assert.Equal(t, "5a9cc47d-2818-4a4a-b99d-db0a28f4e08f", event.Resource.ProjectID)
	assert.Equal(t, "Deploy to Production", event.Resource.Job.Name)
	assert.Equal(t, "completed", event.Resource.Job.State)
	assert.Equal(t, "succeeded", event.Resource.Job.Result)
	assert.NotNil(t, event.Resource.Job.StartTime)
	assert.NotNil(t, event.Resource.Job.FinishTime)

	// Verify repository information
	assert.Len(t, event.Resource.Repositories, 1)
	repo := event.Resource.Repositories[0]
	assert.Equal(t, "self", repo.Alias)
	assert.Equal(t, "487e28f0-8046-41cf-8eee-a566eeca25e3", repo.ID)
	assert.Equal(t, "Git", repo.Type)
	assert.Equal(t, "Example User", repo.Change.Author.Name)
	assert.Equal(t, "user@example.com", repo.Change.Author.Email)
}
