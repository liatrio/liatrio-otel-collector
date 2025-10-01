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
	"strconv"
	"strings"
	"time"
)

const (
	// Azure DevOps Endpoints
	repositoriesEndpoint  = "git/repositories?api-version=7.1"
	branchesEndpoint      = "git/repositories/%s/refs?api-version=7.1&filter=heads/"
	pullRequestsEndpoint  = "git/repositories/%s/pullrequests?api-version=7.1"
	initialCommitEndpoint = "git/repositories/%s/commits?api-version=7.1"
	latestBuildEndpoint   = "build/builds?api-version=7.1"
	testCoverageEndpoint  = "test/codecoverage?buildId=%s&api-version=7.1"
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
	IsDisabled    bool   `json:"isDisabled"`
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
	Reviewers  []struct {
		DisplayName string `json:"displayName"`
		ID          string `json:"id"`
		Vote        *int   `json:"vote"`
	} `json:"reviewers"`
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

// AzureDevOpsCodeCoverage represents code coverage data from Azure DevOps
type AzureDevOpsCodeCoverage struct {
	CoverageData []struct {
		CoverageStats []struct {
			Label            string  `json:"label"`
			Position         int     `json:"position"`
			Total            int     `json:"total"`
			Covered          int     `json:"covered"`
			IsDeltaAvailable bool    `json:"isDeltaAvailable"`
			Delta            float64 `json:"delta"`
		} `json:"coverageStats"`
		BuildPlatform string `json:"buildPlatform"`
		BuildFlavor   string `json:"buildFlavor"`
	} `json:"coverageData"`
	Build struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"build"`
	Status                        string `json:"status"`
	CoverageDetailedSummaryStatus string `json:"coverageDetailedSummaryStatus"`
}

// makeRequest makes an authenticated request to the Azure DevOps REST API
func (ados *azuredevopsScraper) makeRequest(ctx context.Context, endpoint string, apiResource string) (*http.Response, error) {
	baseURLModifier := ""
	if apiResource != "" {
		baseURLModifier = fmt.Sprintf("/%s", apiResource)
	}
	fullURL := fmt.Sprintf("%s/%s%s/_apis/%s", ados.cfg.BaseURL, ados.cfg.Organization, baseURLModifier, endpoint)

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
		Count int                     `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var allRepos []AzureDevOpsRepository
	allRepos = append(allRepos, result.Value...)

	// Filter out disabled repositories or repositories that don't match the search query
	var enabledRepos []AzureDevOpsRepository
	for _, repo := range allRepos {
		if ados.cfg.SearchQuery != "" {
			if strings.Contains(repo.Name, ados.cfg.SearchQuery) {
				enabledRepos = append(enabledRepos, repo)
			}
		} else {
			enabledRepos = append(enabledRepos, repo)
		}
	}

	return enabledRepos, nil
}

// getBranches retrieves all branches for a given repository
func (ados *azuredevopsScraper) getBranches(ctx context.Context, projectID, repoID string) ([]AzureDevOpsBranch, error) {
	var allRefs []struct {
		Name     string `json:"name"`
		ObjectID string `json:"objectId"`
	}

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
		Count int `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	allRefs = append(allRefs, result.Value...)

	var branches []AzureDevOpsBranch
	for _, ref := range allRefs {
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
func (ados *azuredevopsScraper) getPullRequests(ctx context.Context, projectID string, repoID string, status string, minTime time.Time) ([]AzureDevOpsPullRequest, error) {
	var allPRs []AzureDevOpsPullRequest
	skip := 0
	top := 100

	for {
		// Build the endpoint with searchCriteria.minTime if LimitPullRequests is configured
		endpoint := fmt.Sprintf(pullRequestsEndpoint, url.QueryEscape(repoID))
		endpoint += fmt.Sprintf("&searchCriteria.status=%s", url.QueryEscape(status))
		endpoint += fmt.Sprintf("&$top=%d", top)
		endpoint += fmt.Sprintf("&$skip=%d", skip)

		if !minTime.IsZero() {
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
			Count int                      `json:"count"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		allPRs = append(allPRs, result.Value...)

		// Break if we received fewer than the requested number of items
		if result.Count < top {
			break
		}

		skip += top
	}

	return allPRs, nil
}

// getInitialCommit finds the initial commit that diverged from the default branch
func (ados *azuredevopsScraper) getInitialCommit(ctx context.Context, projectID, repoID, defaultBranch, branch string) (*AzureDevOpsCommit, error) {
	// Get commits that are in the branch but not in the default branch
	endpoint := fmt.Sprintf(initialCommitEndpoint, url.QueryEscape(repoID))
	endpoint += fmt.Sprintf("&searchCriteria.itemVersion.version=%s", url.QueryEscape(branch))
	endpoint += "&searchCriteria.excludeDeletes=true"
	endpoint += fmt.Sprintf("&searchCriteria.compareVersion.version=%s", url.QueryEscape(strings.TrimPrefix(defaultBranch, "refs/heads/")))
	endpoint += "&searchCriteria.compareVersion.versionType=branch"
	endpoint += "&searchCriteria.showOldestCommitsFirst=true"
	endpoint += "&$top=1"

	resp, err := ados.makeRequest(ctx, endpoint, projectID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []AzureDevOpsCommit `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Value) == 0 {
		return nil, nil
	}

	return &result.Value[0], nil
}

func (ados *azuredevopsScraper) getCombinedPullRequests(
	ctx context.Context,
	projectID string,
	repoID string,
) ([]AzureDevOpsPullRequest, error) {
	minTime := time.Time{}

	// Add time filter if LimitPullRequests is configured (days in the past)
	if ados.cfg.LimitPullRequests > 0 {
		minTime = time.Now().AddDate(0, 0, -ados.cfg.LimitPullRequests)
	}

	// always grab all open PRs
	activePrs, err := ados.getPullRequests(ctx, projectID, repoID, "active", time.Time{})
	if err != nil {
		return nil, err
	}
	completedPrs, err := ados.getPullRequests(ctx, projectID, repoID, "completed", minTime)
	if err != nil {
		return nil, err
	}
	prs := append(activePrs, completedPrs...)
	return prs, nil
}

func (ados *azuredevopsScraper) getLatestBuildId(ctx context.Context, projectID string, repoID string, branchName string) (string, error) {
	endpoint := latestBuildEndpoint
	endpoint += fmt.Sprintf("&repositoryId=%s", url.QueryEscape(repoID))
	endpoint += "&repositoryType=TfsGit"
	endpoint += fmt.Sprintf("&branchName=%s", url.QueryEscape(branchName))
	endpoint += "&statusFilter=completed"
	endpoint += "&$top=1"
	resp, err := ados.makeRequest(ctx, endpoint, projectID)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		//read body of response and log it with error
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			ados.logger.Sugar().Errorf("API request failed with status %d. Unable to read body due to error %s", resp.StatusCode, err)
			return "", fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
		ados.logger.Sugar().Errorf("API request failed with status %d and body %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			ID int64 `json:"id"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Value) == 0 {
		return "", nil
	}

	return strconv.FormatInt(result.Value[0].ID, 10), nil
}

func (ados *azuredevopsScraper) getCodeCoverageForBuild(ctx context.Context, projectID string, buildId string) (int64, error) {
	endpoint := fmt.Sprintf(testCoverageEndpoint, url.QueryEscape(buildId))
	resp, err := ados.makeRequest(ctx, endpoint, projectID)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result AzureDevOpsCodeCoverage

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	// Calculate overall coverage percentage across all modules
	totalLinesCovered := 0
	totalLinesValid := 0

	for _, buildCoverage := range result.CoverageData {
		for _, module := range buildCoverage.CoverageStats {
			totalLinesCovered += module.Covered
			totalLinesValid += module.Total
		}
	}

	if totalLinesValid == 0 {
		return 0, nil
	}

	coveragePercentage := int64((float64(totalLinesCovered) / float64(totalLinesValid)) * 100)
	return coveragePercentage, nil
}
