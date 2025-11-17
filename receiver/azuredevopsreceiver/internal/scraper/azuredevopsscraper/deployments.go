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
	"strings"
	"time"
)

const (
	vsrmBaseURL = "https://vsrm.dev.azure.com"
	apiVersion  = "7.1"
)

// ReleaseDefinition represents a release pipeline definition
type ReleaseDefinition struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ReleaseDefinitionsResponse represents the API response for release definitions
type ReleaseDefinitionsResponse struct {
	Value []ReleaseDefinition `json:"value"`
}

// ReleaseEnvironment represents an environment/stage in a release definition
type ReleaseEnvironment struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ReleaseDefinitionDetail represents detailed release definition with environments
type ReleaseDefinitionDetail struct {
	ID           int                  `json:"id"`
	Name         string               `json:"name"`
	Environments []ReleaseEnvironment `json:"environments"`
}

// Deployment represents a deployment from Azure DevOps Release Management API
type Deployment struct {
	ID               int       `json:"id"`
	DeploymentStatus string    `json:"deploymentStatus"`
	StartedOn        time.Time `json:"startedOn"`
	CompletedOn      time.Time `json:"completedOn"`
	Release          struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"release"`
}

// DeploymentsResponse represents the API response for deployments
type DeploymentsResponse struct {
	Value []Deployment `json:"value"`
}

// getReleaseDefinitionID finds the release definition ID by pipeline name
func (ados *azuredevopsScraper) getReleaseDefinitionID(ctx context.Context, org, project, pipelineName string) (int, error) {
	urlPath := fmt.Sprintf("%s/%s/%s/_apis/release/definitions", vsrmBaseURL, org, project)

	params := url.Values{}
	params.Set("searchText", pipelineName)
	params.Set("$top", "100")
	params.Set("api-version", apiVersion)

	req, err := http.NewRequestWithContext(ctx, "GET", urlPath+"?"+params.Encode(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ados.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get release definitions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
		return 0, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result ReleaseDefinitionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Try exact match first
	for _, def := range result.Value {
		if strings.EqualFold(strings.TrimSpace(def.Name), strings.TrimSpace(pipelineName)) {
			return def.ID, nil
		}
	}

	// If only one result, use it
	if len(result.Value) == 1 {
		return result.Value[0].ID, nil
	}

	return 0, fmt.Errorf("pipeline '%s' not found (found %d results)", pipelineName, len(result.Value))
}

// getDefinitionEnvironmentID finds the environment ID by stage name within a release definition
func (ados *azuredevopsScraper) getDefinitionEnvironmentID(ctx context.Context, org, project string, definitionID int, stageName string) (int, error) {
	urlPath := fmt.Sprintf("%s/%s/%s/_apis/release/definitions/%d", vsrmBaseURL, org, project, definitionID)

	params := url.Values{}
	params.Set("api-version", apiVersion)

	req, err := http.NewRequestWithContext(ctx, "GET", urlPath+"?"+params.Encode(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ados.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get release definition: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
		return 0, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result ReleaseDefinitionDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	for _, env := range result.Environments {
		if strings.EqualFold(strings.TrimSpace(env.Name), strings.TrimSpace(stageName)) {
			return env.ID, nil
		}
	}

	return 0, fmt.Errorf("stage '%s' not found in pipeline (found %d environments)", stageName, len(result.Environments))
}

// fetchDeployments retrieves deployments for a specific release definition and environment
func (ados *azuredevopsScraper) fetchDeployments(ctx context.Context, org, project string, definitionID, environmentID, lookbackDays int) ([]Deployment, error) {
	urlPath := fmt.Sprintf("%s/%s/%s/_apis/release/deployments", vsrmBaseURL, org, project)

	minTime := time.Now().UTC().AddDate(0, 0, -lookbackDays)
	minTimeISO := minTime.Format("2006-01-02T15:04:05Z")

	var allDeployments []Deployment
	continuationToken := ""

	for {
		params := url.Values{}
		params.Set("definitionId", fmt.Sprintf("%d", definitionID))
		params.Set("definitionEnvironmentId", fmt.Sprintf("%d", environmentID))
		params.Set("minModifiedTime", minTimeISO)
		params.Set("$top", "100")
		params.Set("queryOrder", "descending")
		params.Set("api-version", apiVersion)

		if continuationToken != "" {
			params.Set("continuationToken", continuationToken)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", urlPath+"?"+params.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := ados.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployments: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var result DeploymentsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		allDeployments = append(allDeployments, result.Value...)

		// Check for continuation token in headers
		continuationToken = resp.Header.Get("x-ms-continuationtoken")
		if continuationToken == "" {
			break
		}

		// Small delay to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	return allDeployments, nil
}

// extractServiceName extracts the service name from a deployment's release name
// Assumes release name format: "ServiceName <version/build>"
func extractServiceName(deployment Deployment) string {
	releaseName := deployment.Release.Name
	if releaseName == "" {
		return "unknown"
	}

	// Split on first space and take the first part
	parts := strings.SplitN(releaseName, " ", 2)
	if len(parts) > 0 && parts[0] != "" {
		return strings.TrimSpace(parts[0])
	}

	return "unknown"
}
