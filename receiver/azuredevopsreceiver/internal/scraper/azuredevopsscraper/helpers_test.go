// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureDevOpsProjectUnmarshal(t *testing.T) {
	jsonData := `{
		"id": "12345",
		"name": "Test Project",
		"description": "A test project",
		"url": "https://dev.azure.com/org/_apis/projects/12345",
		"state": "wellFormed",
		"revision": 1,
		"visibility": "private",
		"lastUpdateTime": "2023-01-01T00:00:00Z"
	}`

	var project AzureDevOpsProject
	err := json.Unmarshal([]byte(jsonData), &project)
	require.NoError(t, err)

	assert.Equal(t, "12345", project.ID)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, "A test project", project.Description)
	assert.Equal(t, "wellFormed", project.State)
	assert.Equal(t, "private", project.Visibility)
}

func TestAzureDevOpsRepositoryUnmarshal(t *testing.T) {
	jsonData := `{
		"id": "repo123",
		"name": "test-repo",
		"url": "https://dev.azure.com/org/_apis/git/repositories/repo123",
		"webUrl": "https://dev.azure.com/org/project/_git/test-repo",
		"defaultBranch": "refs/heads/main",
		"size": 1024,
		"remoteUrl": "https://org@dev.azure.com/org/project/_git/test-repo",
		"sshUrl": "git@ssh.dev.azure.com:v3/org/project/test-repo",
		"project": {
			"id": "proj123",
			"name": "Test Project"
		}
	}`

	var repo AzureDevOpsRepository
	err := json.Unmarshal([]byte(jsonData), &repo)
	require.NoError(t, err)

	assert.Equal(t, "repo123", repo.ID)
	assert.Equal(t, "test-repo", repo.Name)
	assert.Equal(t, "refs/heads/main", repo.DefaultBranch)
	assert.Equal(t, int64(1024), repo.Size)
	assert.Equal(t, "proj123", repo.Project.ID)
	assert.Equal(t, "Test Project", repo.Project.Name)
}

func TestAzureDevOpsPullRequestUnmarshal(t *testing.T) {
	jsonData := `{
		"pullRequestId": 123,
		"codeReviewId": 456,
		"status": "active",
		"createdBy": {
			"displayName": "John Doe",
			"id": "user123"
		},
		"creationDate": "2023-01-01T00:00:00Z",
		"closedDate": "0001-01-01T00:00:00Z",
		"title": "Test PR",
		"description": "A test pull request",
		"sourceRefName": "refs/heads/feature-branch",
		"targetRefName": "refs/heads/main",
		"mergeStatus": "succeeded",
		"isDraft": false,
		"mergeId": "merge123"
	}`

	var pr AzureDevOpsPullRequest
	err := json.Unmarshal([]byte(jsonData), &pr)
	require.NoError(t, err)

	assert.Equal(t, 123, pr.PullRequestID)
	assert.Equal(t, "active", pr.Status)
	assert.Equal(t, "John Doe", pr.CreatedBy.DisplayName)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "refs/heads/feature-branch", pr.SourceRefName)
	assert.Equal(t, "refs/heads/main", pr.TargetRefName)
	assert.False(t, pr.IsDraft)
}
