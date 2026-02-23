// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BuildRun represents a build/pipeline run from Azure DevOps Builds API
// API Reference: https://learn.microsoft.com/en-us/rest/api/azure/devops/build/builds/list
type BuildRun struct {
	ID                int          `json:"id"`
	BuildNumber       string       `json:"buildNumber"`
	Status            string       `json:"status"`            // completed, inProgress, cancelling, postponed, notStarted, none
	Result            string       `json:"result"`            // succeeded, partiallySucceeded, failed, canceled, none
	QueueTime         NullableTime `json:"queueTime"`         // When the build was queued
	StartTime         NullableTime `json:"startTime"`         // When the build started
	FinishTime        NullableTime `json:"finishTime"`        // When the build finished
	SourceBranch      string       `json:"sourceBranch"`      // e.g., "refs/heads/main"
	SourceVersion     string       `json:"sourceVersion"`     // Commit SHA
	RequestedFor      User         `json:"requestedFor"`      // User who triggered the build
	RequestedBy       User         `json:"requestedBy"`       // User who requested the build
	Reason            string       `json:"reason"`            // manual, individualCI, batchedCI, schedule, etc.
	Repository        Repository   `json:"repository"`        // Repository information
	Definition        Definition   `json:"definition"`        // Pipeline definition
	Project           Project      `json:"project"`           // Project information
	URI               string       `json:"uri"`               // API URI for this build
	URL               string       `json:"url"`               // Web URL for this build
	Links             Links        `json:"_links"`            // Related links
	RetainedByRelease bool         `json:"retainedByRelease"` // Whether retained by a release
	TriggeredByBuild  *BuildRef    `json:"triggeredByBuild"`  // If triggered by another build
}

// BuildRunsResponse represents the API response for listing builds
type BuildRunsResponse struct {
	Count int        `json:"count"`
	Value []BuildRun `json:"value"`
}

// User represents a user in Azure DevOps
type User struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
	UniqueName  string `json:"uniqueName"` // Usually email
	ImageURL    string `json:"imageUrl"`
}

// Repository represents repository information
type Repository struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Type          string `json:"type"` // TfsGit, GitHub, etc.
	DefaultBranch string `json:"defaultBranch"`
}

// Definition represents a pipeline definition
type Definition struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"` // Folder path in Azure DevOps
	Type string `json:"type"` // build, xaml
	URI  string `json:"uri"`
	URL  string `json:"url"`
}

// Project represents an Azure DevOps project
type Project struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	URL            string       `json:"url"`
	State          string       `json:"state"`
	Revision       int          `json:"revision"`
	Visibility     string       `json:"visibility"` // private, public
	LastUpdateTime NullableTime `json:"lastUpdateTime"`
}

// Links represents related links
type Links struct {
	Self     Link `json:"self"`
	Web      Link `json:"web"`
	Timeline Link `json:"timeline"`
}

// Link represents a single link
type Link struct {
	Href string `json:"href"`
}

// BuildRef represents a reference to another build
type BuildRef struct {
	ID          int    `json:"id"`
	BuildNumber string `json:"buildNumber"`
}

// BuildTimeline represents the timeline of a build with all jobs and tasks
// API Reference: https://learn.microsoft.com/en-us/rest/api/azure/devops/build/timeline/get
type BuildTimeline struct {
	ID            string                `json:"id"`
	ChangeID      int                   `json:"changeId"`
	URL           string                `json:"url"`
	Records       []BuildTimelineRecord `json:"records"`
	LastChangedBy string                `json:"lastChangedBy"`
	LastChangedOn NullableTime          `json:"lastChangedOn"`
}

// BuildTimelineRecord represents a single record in the build timeline (job, stage, or task)
type BuildTimelineRecord struct {
	ID               string         `json:"id"`
	ParentID         string         `json:"parentId"`         // Parent record ID (for hierarchy)
	Type             string         `json:"type"`             // Stage, Job, Task, Checkpoint
	Name             string         `json:"name"`             // Display name
	StartTime        NullableTime   `json:"startTime"`        // When this record started
	FinishTime       NullableTime   `json:"finishTime"`       // When this record finished
	CurrentOperation string         `json:"currentOperation"` // Current operation description
	PercentComplete  int            `json:"percentComplete"`  // 0-100
	State            string         `json:"state"`            // pending, inProgress, completed
	Result           string         `json:"result"`           // succeeded, failed, canceled, skipped, succeededWithIssues, abandoned
	ResultCode       string         `json:"resultCode"`       // Additional result information
	ChangeID         int            `json:"changeId"`         // Change identifier
	LastModified     NullableTime   `json:"lastModified"`     // Last modification time
	WorkerName       string         `json:"workerName"`       // Agent/worker name
	Order            int            `json:"order"`            // Execution order
	Details          *Link          `json:"details"`          // Link to details
	ErrorCount       int            `json:"errorCount"`       // Number of errors
	WarningCount     int            `json:"warningCount"`     // Number of warnings
	URL              string         `json:"url"`              // API URL
	Log              *Link          `json:"log"`              // Link to logs
	Task             *TaskReference `json:"task"`             // Task reference (for task records)
	Attempt          int            `json:"attempt"`          // Attempt number (for retries)
	Identifier       string         `json:"identifier"`       // Unique identifier
}

// TaskReference represents a reference to a task definition
type TaskReference struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// fetchBuildRuns fetches build runs from the Azure DevOps Builds API with pagination support
func (ados *azuredevopsScraper) fetchBuildRuns(ctx context.Context, minTime time.Time) ([]BuildRun, error) {
	var allBuilds []BuildRun
	continuationToken := ""
	pageSize := 100 // Azure DevOps max page size
	maxRuns := ados.cfg.MaxPipelineRuns

	// If maxRuns is 0, fetch all available runs with pagination
	// Otherwise, limit to maxRuns
	if maxRuns == 0 {
		ados.logger.Sugar().Info("MaxPipelineRuns is 0, will fetch all available pipeline runs with pagination")
	} else {
		ados.logger.Sugar().Infof("MaxPipelineRuns is %d, will fetch up to %d pipeline runs", maxRuns, maxRuns)
	}

	for {
		// Build the API URL
		apiURL := fmt.Sprintf("%s/%s/%s/_apis/build/builds",
			ados.cfg.BaseURL,
			url.PathEscape(ados.cfg.Organization),
			url.PathEscape(ados.cfg.Project))

		// Add query parameters
		params := url.Values{}
		params.Add("api-version", apiVersion)
		params.Add("statusFilter", "completed") // Only fetch completed builds
		params.Add("minTime", minTime.Format(time.RFC3339))

		// Determine page size for this request
		currentPageSize := pageSize
		if maxRuns > 0 {
			remaining := maxRuns - len(allBuilds)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				currentPageSize = remaining
			}
		}
		params.Add("$top", fmt.Sprintf("%d", currentPageSize))

		// Add continuation token if we have one
		if continuationToken != "" {
			params.Add("continuationToken", continuationToken)
		}

		fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
		ados.logger.Sugar().Debugf("Fetching build runs (page size: %d, total so far: %d)", currentPageSize, len(allBuilds))

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Execute request
		resp, err := ados.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch build runs: %w", err)
		}

		// Check response status
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		// Parse response
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var buildsResp BuildRunsResponse
		if err := json.Unmarshal(body, &buildsResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal builds response: %w", err)
		}

		// Append builds from this page
		allBuilds = append(allBuilds, buildsResp.Value...)
		ados.logger.Sugar().Infof("Fetched %d build runs in this page (total: %d)", buildsResp.Count, len(allBuilds))

		// Check for continuation token in response headers
		continuationToken = resp.Header.Get("x-ms-continuationtoken")

		// Stop if no more pages or we've hit our limit
		if continuationToken == "" || buildsResp.Count < currentPageSize {
			break
		}

		if maxRuns > 0 && len(allBuilds) >= maxRuns {
			break
		}
	}

	ados.logger.Sugar().Infof("Fetched total of %d build runs from Builds API", len(allBuilds))

	return allBuilds, nil
}

// fetchBuildTimeline fetches the timeline for a specific build, containing job and task details
func (ados *azuredevopsScraper) fetchBuildTimeline(ctx context.Context, buildID int) (*BuildTimeline, error) {
	// Build the API URL
	apiURL := fmt.Sprintf("%s/%s/%s/_apis/build/builds/%d/timeline",
		ados.cfg.BaseURL,
		url.PathEscape(ados.cfg.Organization),
		url.PathEscape(ados.cfg.Project),
		buildID)

	// Add query parameters
	params := url.Values{}
	params.Add("api-version", apiVersion)

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	ados.logger.Sugar().Debugf("Fetching build timeline for build %d from: %s", buildID, fullURL)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := ados.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch build timeline: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var timeline BuildTimeline
	if err := json.Unmarshal(body, &timeline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal timeline response: %w", err)
	}

	ados.logger.Sugar().Debugf("Fetched timeline with %d records for build %d", len(timeline.Records), buildID)

	return &timeline, nil
}

// mapBuildStatus normalizes Azure DevOps build status/result to a standard status
func mapBuildStatus(status, result string) string {
	// If build is not completed, return the status
	if status != "completed" {
		return status
	}

	// For completed builds, map the result
	switch result {
	case "succeeded":
		return "succeeded"
	case "partiallySucceeded":
		return "failed" // Treat partially succeeded as failed for alerting purposes
	case "failed":
		return "failed"
	case "canceled":
		return "canceled"
	default:
		return "unknown"
	}
}

// extractJobsFromTimeline extracts job-level records from a build timeline
func extractJobsFromTimeline(timeline *BuildTimeline) []BuildTimelineRecord {
	var jobs []BuildTimelineRecord

	for _, record := range timeline.Records {
		// Filter for Job type records only (exclude Stage and Task types)
		if record.Type == "Job" {
			jobs = append(jobs, record)
		}
	}

	return jobs
}
