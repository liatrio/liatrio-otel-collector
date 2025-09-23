// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewAzureDevOpsScraper(t *testing.T) {
	cfg := &Config{
		Organization:      "test-org",
		BaseURL:           "https://dev.azure.com",
		LimitPullRequests: 100,
		ConcurrencyLimit:  5,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	assert.NotNil(t, scraper)
	assert.Equal(t, cfg, scraper.cfg)
	assert.NotNil(t, scraper.logger)
	assert.NotNil(t, scraper.mb)
	assert.NotNil(t, scraper.rb)
}

func TestAzureDevOpsScraperStart(t *testing.T) {
	cfg := &Config{
		Organization: "test-org",
		BaseURL:      "https://dev.azure.com",
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	// Test start without host - should not panic
	err := scraper.start(context.Background(), nil)
	require.NoError(t, err)
}

func TestMakeRequestURL(t *testing.T) {
	cfg := &Config{
		Organization: "test-org",
		BaseURL:      "https://dev.azure.com",
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	// We can't easily test the actual request without mocking, but we can verify the scraper was created
	assert.NotNil(t, scraper)
	assert.Equal(t, "test-org", scraper.cfg.Organization)
	assert.Equal(t, "https://dev.azure.com", scraper.cfg.BaseURL)
}

// Mock HTTP server helper function
func setupMockServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	mux := http.NewServeMux()

	for pattern, handler := range handlers {
		mux.HandleFunc(pattern, handler)
	}

	return httptest.NewServer(mux)
}

func TestScrapeSuccess(t *testing.T) {
	// Mock responses
	repositoriesResponse := `{
		"value": [
			{
				"id": "repo-1",
				"name": "test-repo",
				"url": "https://dev.azure.com/test-org/test-project/_apis/git/repositories/repo-1",
				"webUrl": "https://dev.azure.com/test-org/test-project/_git/test-repo",
				"defaultBranch": "refs/heads/main",
				"size": 1024,
				"project": {
					"id": "project-1",
					"name": "test-project"
				}
			}
		]
	}`

	branchesResponse := `{
		"value": [
			{
				"name": "refs/heads/main",
				"objectId": "abc123"
			},
			{
				"name": "refs/heads/feature-branch",
				"objectId": "def456"
			}
		]
	}`

	pullRequestsResponse := `{
		"value": [
			{
				"pullRequestId": 1,
				"status": "completed",
				"creationDate": "2024-01-01T10:00:00Z",
				"closedDate": "2024-01-01T11:00:00Z",
				"title": "Test PR",
				"sourceRefName": "refs/heads/feature-branch",
				"targetRefName": "refs/heads/main",
				"createdBy": {
					"displayName": "Test User",
					"id": "user-1"
				}
			},
			{
				"pullRequestId": 2,
				"status": "active",
				"creationDate": "2024-01-01T09:00:00Z",
				"title": "Active PR",
				"sourceRefName": "refs/heads/another-feature",
				"targetRefName": "refs/heads/main",
				"createdBy": {
					"displayName": "Test User 2",
					"id": "user-2"
				}
			}
		]
	}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, repositoriesResponse)
		},
		"/test-org/test-project/_apis/git/repositories/repo-1/refs": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, branchesResponse)
		},
		"/test-org/test-project/_apis/git/repositories/repo-1/pullrequests": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, pullRequestsResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  server.URL,
		LimitPullRequests:        100,
		ConcurrencyLimit:         5,
		MetricsBuilderConfig:     metadata.DefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Verify metrics were collected
	assert.Greater(t, metrics.MetricCount(), 0)
	assert.Greater(t, metrics.ResourceMetrics().Len(), 0)
}

func TestScrapeClientNotInitialized(t *testing.T) {
	cfg := &Config{
		Organization: "test-org",
		Project:      "test-project",
		BaseURL:      "https://dev.azure.com",
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	// Don't initialize client

	metrics, err := scraper.scrape(context.Background())
	assert.Error(t, err)
	assert.Equal(t, errClientNotInitErr, err)
	assert.NotNil(t, metrics)
}

func TestScrapeRepositoriesError(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  server.URL,
		MetricsBuilderConfig:     metadata.DefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	metrics, err := scraper.scrape(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed with status 500")
	assert.NotNil(t, metrics)
}

func TestGetRepositoriesSuccess(t *testing.T) {
	repositoriesResponse := `{
		"value": [
			{
				"id": "repo-1",
				"name": "test-repo",
				"url": "https://dev.azure.com/test-org/test-project/_apis/git/repositories/repo-1",
				"webUrl": "https://dev.azure.com/test-org/test-project/_git/test-repo",
				"defaultBranch": "refs/heads/main",
				"size": 1024,
				"project": {
					"id": "project-1",
					"name": "test-project"
				}
			}
		]
	}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories": func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, repositoriesResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization: "test-org",
		BaseURL:      server.URL,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	repos, err := scraper.getRepositories(context.Background(), "test-project")
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "repo-1", repos[0].ID)
	assert.Equal(t, "test-repo", repos[0].Name)
	assert.Equal(t, "refs/heads/main", repos[0].DefaultBranch)
}

func TestGetRepositoriesError(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization: "test-org",
		BaseURL:      server.URL,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	repos, err := scraper.getRepositories(context.Background(), "test-project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed with status 404")
	assert.Nil(t, repos)
}

func TestGetBranchesSuccess(t *testing.T) {
	branchesResponse := `{
		"value": [
			{
				"name": "refs/heads/main",
				"objectId": "abc123"
			},
			{
				"name": "refs/heads/feature-branch",
				"objectId": "def456"
			}
		]
	}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories/repo-1/refs": func(w http.ResponseWriter, r *http.Request) {
			// Verify query parameters
			assert.True(t, strings.Contains(r.URL.RawQuery, "filter=heads/"))
			assert.True(t, strings.Contains(r.URL.RawQuery, "api-version=7.1"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, branchesResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization: "test-org",
		BaseURL:      server.URL,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	branches, err := scraper.getBranches(context.Background(), "test-project", "repo-1")
	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.Equal(t, "main", branches[0].Name)
	assert.Equal(t, "abc123", branches[0].ObjectID)
	assert.Equal(t, "feature-branch", branches[1].Name)
	assert.Equal(t, "def456", branches[1].ObjectID)
}

func TestGetPullRequestsSuccess(t *testing.T) {
	pullRequestsResponse := `{
		"value": [
			{
				"pullRequestId": 1,
				"status": "completed",
				"creationDate": "2024-01-01T10:00:00Z",
				"closedDate": "2024-01-01T11:00:00Z",
				"title": "Test PR",
				"sourceRefName": "refs/heads/feature-branch",
				"targetRefName": "refs/heads/main",
				"createdBy": {
					"displayName": "Test User",
					"id": "user-1"
				}
			}
		]
	}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories/repo-1/pullrequests": func(w http.ResponseWriter, r *http.Request) {
			// Verify query parameters
			assert.True(t, strings.Contains(r.URL.RawQuery, "api-version=7.1"))
			// Verify time filter is applied when LimitPullRequests > 0
			assert.True(t, strings.Contains(r.URL.RawQuery, "searchCriteria.minTime="))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, pullRequestsResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:      "test-org",
		BaseURL:           server.URL,
		LimitPullRequests: 30, // 30 days in the past
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	prs, err := scraper.getPullRequests(context.Background(), "test-project", "repo-1", "completed", time.Time{})
	require.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, 1, prs[0].PullRequestID)
	assert.Equal(t, "completed", prs[0].Status)
	assert.Equal(t, "Test PR", prs[0].Title)
	assert.Equal(t, "refs/heads/feature-branch", prs[0].SourceRefName)
}

func TestGetPullRequestsWithTimeFilter(t *testing.T) {
	pullRequestsResponse := `{"value": []}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories/repo-1/pullrequests": func(w http.ResponseWriter, r *http.Request) {
			// Verify time filter is applied with custom days
			assert.True(t, strings.Contains(r.URL.RawQuery, "searchCriteria.minTime="))
			assert.True(t, strings.Contains(r.URL.RawQuery, "api-version=7.1"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, pullRequestsResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:      "test-org",
		BaseURL:           server.URL,
		LimitPullRequests: 7, // 7 days in the past
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	prs, err := scraper.getPullRequests(context.Background(), "test-project", "repo-1", "completed", time.Time{})
	require.NoError(t, err)
	assert.Len(t, prs, 0)
}

func TestGetPullRequestsWithoutTimeFilter(t *testing.T) {
	pullRequestsResponse := `{"value": []}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories/repo-1/pullrequests": func(w http.ResponseWriter, r *http.Request) {
			// Verify no time filter is applied when LimitPullRequests is 0
			assert.False(t, strings.Contains(r.URL.RawQuery, "searchCriteria.minTime="))
			assert.True(t, strings.Contains(r.URL.RawQuery, "api-version=7.1"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, pullRequestsResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:      "test-org",
		BaseURL:           server.URL,
		LimitPullRequests: 0, // No time filter
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	prs, err := scraper.getPullRequests(context.Background(), "test-project", "repo-1", "completed", time.Time{})
	require.NoError(t, err)
	assert.Len(t, prs, 0)
}

func TestMakeRequestSuccess(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/test/endpoint": func(w http.ResponseWriter, r *http.Request) {
			// Verify request headers
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"success": true}`)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization: "test-org",
		BaseURL:      server.URL,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	resp, err := scraper.makeRequest(context.Background(), "test/endpoint?api-version=7.1", "test-project")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestScrapeWithConcurrencyLimit(t *testing.T) {
	repositoriesResponse := `{
		"value": [
			{
				"id": "repo-1",
				"name": "test-repo-1",
				"webUrl": "https://dev.azure.com/test-org/test-project/_git/test-repo-1",
				"defaultBranch": "refs/heads/main"
			},
			{
				"id": "repo-2", 
				"name": "test-repo-2",
				"webUrl": "https://dev.azure.com/test-org/test-project/_git/test-repo-2",
				"defaultBranch": "refs/heads/main"
			}
		]
	}`

	branchesResponse := `{"value": []}`
	pullRequestsResponse := `{"value": []}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, repositoriesResponse)
		},
	}

	// Add handlers for both repositories
	for _, repoID := range []string{"repo-1", "repo-2"} {
		repoID := repoID // capture loop variable
		handlers[fmt.Sprintf("/test-org/test-project/_apis/git/repositories/%s/refs", repoID)] = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, branchesResponse)
		}
		handlers[fmt.Sprintf("/test-org/test-project/_apis/git/repositories/%s/pullrequests", repoID)] = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, pullRequestsResponse)
		}
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  server.URL,
		ConcurrencyLimit:         1, // Test with concurrency limit of 1
		MetricsBuilderConfig:     metadata.DefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Greater(t, metrics.MetricCount(), 0)
}

func TestScrapeWithBranchMetrics(t *testing.T) {
	repositoriesResponse := `{
		"value": [
			{
				"id": "repo-1",
				"name": "test-repo",
				"webUrl": "https://dev.azure.com/test-org/test-project/_git/test-repo",
				"defaultBranch": "refs/heads/main"
			}
		]
	}`

	branchesResponse := `{
		"value": [
			{
				"name": "refs/heads/main",
				"objectId": "abc123"
			},
			{
				"name": "refs/heads/feature-branch",
				"objectId": "def456"
			}
		]
	}`

	pullRequestsResponse := `{
		"value": [
			{
				"pullRequestId": 1,
				"status": "completed",
				"creationDate": "2024-01-01T10:00:00Z",
				"closedDate": "2024-01-01T11:00:00Z",
				"sourceRefName": "refs/heads/feature-branch"
			}
		]
	}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, repositoriesResponse)
		},
		"/test-org/test-project/_apis/git/repositories/repo-1/refs": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, branchesResponse)
		},
		"/test-org/test-project/_apis/git/repositories/repo-1/pullrequests": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, pullRequestsResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  server.URL,
		MetricsBuilderConfig:     metadata.DefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Verify that metrics include branch and PR data
	assert.Greater(t, metrics.MetricCount(), 0)
	resourceMetrics := metrics.ResourceMetrics()
	assert.Greater(t, resourceMetrics.Len(), 0)

	// Check that resource attributes are set correctly
	resource := resourceMetrics.At(0).Resource()
	attrs := resource.Attributes()
	vendorName, exists := attrs.Get("vcs.provider.name")
	assert.True(t, exists)
	assert.Equal(t, "azuredevops", vendorName.Str())

	orgName, exists := attrs.Get("vcs.owner.name")
	assert.True(t, exists)
	assert.Equal(t, "test-org", orgName.Str())
}

func TestGetBranchesError(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories/repo-1/refs": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization: "test-org",
		BaseURL:      server.URL,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	branches, err := scraper.getBranches(context.Background(), "test-project", "repo-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed with status 401")
	assert.Nil(t, branches)
}

func TestGetPullRequestsError(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories/repo-1/pullrequests": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization: "test-org",
		BaseURL:      server.URL,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	prs, err := scraper.getPullRequests(context.Background(), "test-project", "repo-1", "completed", time.Time{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed with status 403")
	assert.Nil(t, prs)
}

func TestMakeRequestWithoutBaseUrlModifier(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/_apis/test/endpoint": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"success": true}`)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization: "test-org",
		BaseURL:      server.URL,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	resp, err := scraper.makeRequest(context.Background(), "test/endpoint", "")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestScrapeWithPullRequestMetrics(t *testing.T) {
	// Test with realistic timestamps to verify time calculations
	now := time.Now()
	creationTime := now.Add(-2 * time.Hour)
	closedTime := now.Add(-1 * time.Hour)

	repositoriesResponse := `{
		"value": [
			{
				"id": "repo-1",
				"name": "test-repo",
				"webUrl": "https://dev.azure.com/test-org/test-project/_git/test-repo",
				"defaultBranch": "refs/heads/main"
			}
		]
	}`

	branchesResponse := `{"value": []}`

	pullRequestsResponse := fmt.Sprintf(`{
		"value": [
			{
				"pullRequestId": 1,
				"status": "completed",
				"creationDate": "%s",
				"closedDate": "%s",
				"sourceRefName": "refs/heads/feature-branch"
			},
			{
				"pullRequestId": 2,
				"status": "active",
				"creationDate": "%s",
				"sourceRefName": "refs/heads/another-feature"
			}
		]
	}`, creationTime.Format(time.RFC3339), closedTime.Format(time.RFC3339), creationTime.Format(time.RFC3339))

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/test-org/test-project/_apis/git/repositories": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, repositoriesResponse)
		},
		"/test-org/test-project/_apis/git/repositories/repo-1/refs": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, branchesResponse)
		},
		"/test-org/test-project/_apis/git/repositories/repo-1/pullrequests": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, pullRequestsResponse)
		},
	}

	server := setupMockServer(t, handlers)
	defer server.Close()

	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  server.URL,
		MetricsBuilderConfig:     metadata.DefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)
	scraper.client = &http.Client{}

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Greater(t, metrics.MetricCount(), 0)
}
