// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFetchBuildRuns(t *testing.T) {
	// Load test data
	testData, err := os.ReadFile("testdata/build-run-response.json")
	require.NoError(t, err, "Failed to load test data")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		assert.Equal(t, "7.1", r.URL.Query().Get("api-version"))
		assert.Equal(t, "completed", r.URL.Query().Get("statusFilter"))
		assert.Equal(t, "100", r.URL.Query().Get("$top"))
		assert.NotEmpty(t, r.URL.Query().Get("minTime"))

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	// Create scraper with mock server
	cfg := &Config{
		Organization: "test-org",
		Project:      "test-project",
		BaseURL:      server.URL,
	}

	scraper := &azuredevopsScraper{
		cfg:    cfg,
		client: server.Client(),
		logger: zap.NewNop(),
	}

	// Execute test
	minTime := time.Now().Add(-24 * time.Hour)
	builds, err := scraper.fetchBuildRuns(context.Background(), minTime)

	// Assertions
	require.NoError(t, err)
	assert.Len(t, builds, 2, "Expected 2 builds")

	// Verify first build
	build1 := builds[0]
	assert.Equal(t, 12345, build1.ID)
	assert.Equal(t, "20240219.1", build1.BuildNumber)
	assert.Equal(t, "completed", build1.Status)
	assert.Equal(t, "succeeded", build1.Result)
	assert.Equal(t, "refs/heads/main", build1.SourceBranch)
	assert.Equal(t, "abc123def456", build1.SourceVersion)
	assert.Equal(t, "John Doe", build1.RequestedFor.DisplayName)
	assert.Equal(t, "john.doe@example.com", build1.RequestedFor.UniqueName)
	assert.Equal(t, "individualCI", build1.Reason)
	assert.Equal(t, "my-repo", build1.Repository.Name)
	assert.Equal(t, "CI Pipeline", build1.Definition.Name)
	assert.Equal(t, "MyProject", build1.Project.Name)
	assert.False(t, build1.QueueTime.IsZero())
	assert.False(t, build1.StartTime.IsZero())
	assert.False(t, build1.FinishTime.IsZero())

	// Verify second build
	build2 := builds[1]
	assert.Equal(t, 12346, build2.ID)
	assert.Equal(t, "20240219.2", build2.BuildNumber)
	assert.Equal(t, "completed", build2.Status)
	assert.Equal(t, "failed", build2.Result)
	assert.Equal(t, "refs/heads/feature/new-feature", build2.SourceBranch)
	assert.Equal(t, "Jane Smith", build2.RequestedFor.DisplayName)
	assert.Equal(t, "manual", build2.Reason)
}

func TestFetchBuildTimeline(t *testing.T) {
	// Load test data
	testData, err := os.ReadFile("testdata/build-timeline-response.json")
	require.NoError(t, err, "Failed to load test data")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		assert.Equal(t, "7.1", r.URL.Query().Get("api-version"))
		assert.Contains(t, r.URL.Path, "/12345/timeline")

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	// Create scraper with mock server
	cfg := &Config{
		Organization: "test-org",
		Project:      "test-project",
		BaseURL:      server.URL,
	}

	scraper := &azuredevopsScraper{
		cfg:    cfg,
		client: server.Client(),
		logger: zap.NewNop(),
	}

	// Execute test
	timeline, err := scraper.fetchBuildTimeline(context.Background(), 12345)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, timeline)
	assert.Equal(t, "timeline-12345", timeline.ID)
	assert.Equal(t, 5, timeline.ChangeID)
	assert.Len(t, timeline.Records, 6, "Expected 6 timeline records")

	// Verify record types
	recordTypes := make(map[string]int)
	for _, record := range timeline.Records {
		recordTypes[record.Type]++
	}
	assert.Equal(t, 2, recordTypes["Stage"], "Expected 2 Stage records")
	assert.Equal(t, 3, recordTypes["Job"], "Expected 3 Job records")
	assert.Equal(t, 1, recordTypes["Task"], "Expected 1 Task record")

	// Verify a job record
	var buildJob *BuildTimelineRecord
	for i := range timeline.Records {
		if timeline.Records[i].Name == "Build Job" {
			buildJob = &timeline.Records[i]
			break
		}
	}
	require.NotNil(t, buildJob, "Build Job not found")
	assert.Equal(t, "job-1", buildJob.ID)
	assert.Equal(t, "stage-1", buildJob.ParentID)
	assert.Equal(t, "Job", buildJob.Type)
	assert.Equal(t, "completed", buildJob.State)
	assert.Equal(t, "succeeded", buildJob.Result)
	assert.Equal(t, "Hosted Agent", buildJob.WorkerName)
	assert.Equal(t, 1, buildJob.Attempt)
	assert.Equal(t, 0, buildJob.ErrorCount)
	assert.Equal(t, 2, buildJob.WarningCount)
	assert.False(t, buildJob.StartTime.IsZero())
	assert.False(t, buildJob.FinishTime.IsZero())
}

func TestMapBuildStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		result   string
		expected string
	}{
		{
			name:     "completed succeeded",
			status:   "completed",
			result:   "succeeded",
			expected: "succeeded",
		},
		{
			name:     "completed failed",
			status:   "completed",
			result:   "failed",
			expected: "failed",
		},
		{
			name:     "completed partiallySucceeded",
			status:   "completed",
			result:   "partiallySucceeded",
			expected: "failed",
		},
		{
			name:     "completed canceled",
			status:   "completed",
			result:   "canceled",
			expected: "canceled",
		},
		{
			name:     "completed unknown result",
			status:   "completed",
			result:   "someUnknownResult",
			expected: "unknown",
		},
		{
			name:     "inProgress",
			status:   "inProgress",
			result:   "",
			expected: "inProgress",
		},
		{
			name:     "notStarted",
			status:   "notStarted",
			result:   "",
			expected: "notStarted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapBuildStatus(tt.status, tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJobsFromTimeline(t *testing.T) {
	// Load test data
	testData, err := os.ReadFile("testdata/build-timeline-response.json")
	require.NoError(t, err, "Failed to load test data")

	var timeline BuildTimeline
	err = json.Unmarshal(testData, &timeline)
	require.NoError(t, err, "Failed to unmarshal timeline")

	// Execute test
	jobs := extractJobsFromTimeline(&timeline)

	// Assertions
	assert.Len(t, jobs, 3, "Expected 3 Job records")

	// Verify all extracted records are jobs
	for _, job := range jobs {
		assert.Equal(t, "Job", job.Type)
	}

	// Verify job names
	jobNames := make([]string, len(jobs))
	for i, job := range jobs {
		jobNames[i] = job.Name
	}
	assert.Contains(t, jobNames, "Build Job")
	assert.Contains(t, jobNames, "Unit Tests")
	assert.Contains(t, jobNames, "Integration Tests")

	// Verify attempt numbers
	var integrationTestJob *BuildTimelineRecord
	for i := range jobs {
		if jobs[i].Name == "Integration Tests" {
			integrationTestJob = &jobs[i]
			break
		}
	}
	require.NotNil(t, integrationTestJob)
	assert.Equal(t, 2, integrationTestJob.Attempt, "Integration Tests should have attempt=2")
}

func TestNullableTimeUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expectZero bool
	}{
		{
			name:       "RFC3339 with timezone",
			input:      `"2024-02-19T10:00:00.0000000Z"`,
			expectErr:  false,
			expectZero: false,
		},
		{
			name:       "null value",
			input:      `"null"`,
			expectErr:  false,
			expectZero: true,
		},
		{
			name:       "empty string",
			input:      `""`,
			expectErr:  false,
			expectZero: true,
		},
		{
			name:       "zero time",
			input:      `"0001-01-01T00:00:00"`,
			expectErr:  false,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nt NullableTime
			err := json.Unmarshal([]byte(tt.input), &nt)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectZero {
					assert.True(t, nt.IsZero(), "Expected zero time")
				} else {
					assert.False(t, nt.IsZero(), "Expected non-zero time")
				}
			}
		})
	}
}
