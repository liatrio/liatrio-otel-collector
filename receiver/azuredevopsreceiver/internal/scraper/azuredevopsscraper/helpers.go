// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	// Azure DevOps Endpoints
	repositoriesEndpoint = "git/repositories?api-version=7.1"
	branchesEndpoint     = "git/repositories/%s/refs?api-version=7.1&filter=heads/"
	pullRequestsEndpoint = "git/repositories/%s/pullrequests?api-version=7.1"
)

// AzureDevOpsProject represents a project in Azure DevOps
type AzureDevOpsProject struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	Revision       int       `json:"revision"`
	Visibility     string    `json:"visibility"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

// AzureDevOpsRepository represents a Git repository in Azure DevOps
type AzureDevOpsRepository struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	WebURL        string `json:"webUrl"`
	DefaultBranch string `json:"defaultBranch"`
	Size          int64  `json:"size"`
	RemoteURL     string `json:"remoteUrl"`
	SSHURL        string `json:"sshUrl"`
	Project       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
}

// AzureDevOpsBranch represents a Git branch in Azure DevOps
type AzureDevOpsBranch struct {
	Name        string    `json:"name"`
	ObjectID    string    `json:"objectId"`
	CreatedDate time.Time `json:"createdDate"`
}

// AzureDevOpsPullRequest represents a pull request in Azure DevOps
type AzureDevOpsPullRequest struct {
	PullRequestID int    `json:"pullRequestId"`
	CodeReviewID  int    `json:"codeReviewId"`
	Status        string `json:"status"`
	CreatedBy     struct {
		DisplayName string `json:"displayName"`
		ID          string `json:"id"`
	} `json:"createdBy"`
	CreationDate    time.Time `json:"creationDate"`
	ClosedDate      time.Time `json:"closedDate"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	SourceRefName   string    `json:"sourceRefName"`
	TargetRefName   string    `json:"targetRefName"`
	MergeStatus     string    `json:"mergeStatus"`
	IsDraft         bool      `json:"isDraft"`
	MergeID         string    `json:"mergeId"`
	LastMergeCommit struct {
		CommitID string `json:"commitId"`
		URL      string `json:"url"`
	} `json:"lastMergeCommit"`
	Repository AzureDevOpsRepository `json:"repository"`
}

// AzureDevOpsCommit represents a Git commit in Azure DevOps
type AzureDevOpsCommit struct {
	CommitID string `json:"commitId"`
	Author   struct {
		Name  string    `json:"name"`
		Email string    `json:"email"`
		Date  time.Time `json:"date"`
	} `json:"author"`
	Committer struct {
		Name  string    `json:"name"`
		Email string    `json:"email"`
		Date  time.Time `json:"date"`
	} `json:"committer"`
	Comment   string `json:"comment"`
	URL       string `json:"url"`
	RemoteURL string `json:"remoteUrl"`
}

// makeRequest makes an authenticated request to the Azure DevOps REST API
func (ados *azuredevopsScraper) makeRequest(ctx context.Context, endpoint string, baseUrlModifier string) (*http.Response, error) {
	baseModifier := ""
	if baseUrlModifier != "" {
		baseModifier = fmt.Sprintf("/%s/", baseUrlModifier)
	}
	fullURL := fmt.Sprintf("%s/%s%s/_apis/%s", ados.cfg.BaseURL, ados.cfg.Organization, baseModifier, endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	return ados.client.Do(req)
}

// getRepositories retrieves all repositories for a given project
func (ados *azuredevopsScraper) getRepositories(ctx context.Context, projectID string) ([]AzureDevOpsRepository, error) {
	resp, err := ados.makeRequest(ctx, repositoriesEndpoint, projectID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []AzureDevOpsRepository `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}

// getBranches retrieves all branches for a given repository
func (ados *azuredevopsScraper) getBranches(ctx context.Context, projectID, repoID string) ([]AzureDevOpsBranch, error) {
	endpoint := fmt.Sprintf(branchesEndpoint, url.QueryEscape(repoID))
	resp, err := ados.makeRequest(ctx, endpoint, projectID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			Name     string `json:"name"`
			ObjectID string `json:"objectId"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var branches []AzureDevOpsBranch
	for _, ref := range result.Value {
		// Extract branch name from refs/heads/branch-name
		branchName := ref.Name
		if len(branchName) > 11 && branchName[:11] == "refs/heads/" {
			branchName = branchName[11:]
		}

		branches = append(branches, AzureDevOpsBranch{
			Name:        branchName,
			ObjectID:    ref.ObjectID,
			CreatedDate: time.Now(), // Azure DevOps doesn't provide branch creation date directly
		})
	}

	return branches, nil
}

// getPullRequests retrieves pull requests for a given repository
func (ados *azuredevopsScraper) getPullRequests(ctx context.Context, projectID, repoID string) ([]AzureDevOpsPullRequest, error) {
	// Build the endpoint with searchCriteria.minTime if LimitPullRequests is configured
	endpoint := fmt.Sprintf(pullRequestsEndpoint, url.QueryEscape(repoID))

	// Add time filter if LimitPullRequests is configured (days in the past)
	if ados.cfg.LimitPullRequests > 0 {
		minTime := time.Now().AddDate(0, 0, -ados.cfg.LimitPullRequests)
		minTimeStr := minTime.Format(time.RFC3339)
		endpoint += fmt.Sprintf("&searchCriteria.minTime=%s", url.QueryEscape(minTimeStr))
	}

	resp, err := ados.makeRequest(ctx, endpoint, projectID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []AzureDevOpsPullRequest `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}
