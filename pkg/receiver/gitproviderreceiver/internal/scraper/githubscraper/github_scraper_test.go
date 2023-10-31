// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// This is from https://github.com/google/go-github/blob/master/github/repos_collaborators_test.go, creates mock http server
func setupMockHttpServer() (client *github.Client, mux *http.ServeMux, serverURL string, teardown func()) {
	baseURLPath := "/api-v3"

	// mux is the HTTP request multiplexer used with the test server.
	mux = http.NewServeMux()

	// We want to ensure that tests catch mistakes where the endpoint URL is
	// specified as absolute rather than relative. It only makes a difference
	// when there's a non-empty base URL path. So, use that. See issue #752.
	apiHandler := http.NewServeMux()
	apiHandler.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		fmt.Fprintln(os.Stderr, "\tSee https://github.com/google/go-github/issues/752 for information.")
		http.Error(w, "Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the GitHub client being tested and is
	// configured to use test server.
	client = github.NewClient(nil)
	url, _ := url.Parse(server.URL + baseURLPath + "/")
	client.BaseURL = url
	client.UploadURL = url

	return client, mux, server.URL, server.Close
}

func TestGetContributorCount(t *testing.T) {
	client, mux, _, teardown := setupMockHttpServer()
	defer teardown()
	mux.HandleFunc("/repos/o/r/contributors", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"id":1}, {"id":2}]`)
	})

	testCases := []struct {
		desc          string
		repo          SearchNodeRepository
		org           string
		resp          string
		expectedErr   error
		expectedCount int
	}{
		{
			desc:          "valid",
			repo:          SearchNodeRepository{Name: "r"},
			org:           "o",
			resp:          `[{"id":1}, {"id":2}]`,
			expectedErr:   nil,
			expectedCount: 2,
		},
		{
			desc:          "error",
			repo:          SearchNodeRepository{Name: "junk"},
			org:           "junk",
			resp:          `[{"id":1}, {"id":2}]`,
			expectedErr:   errors.New("GET " + client.BaseURL.String() + "repos/junk/r/contributors: 404  []"),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			ghs.cfg.GitHubOrg = tc.org
			ctx := context.Background()
			now := pcommon.NewTimestampFromTime(time.Now())

			contribs, err := ghs.getContributorCount(ctx, client, SearchNodeRepository{Name: "r"}, now)
			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
				assert.Equal(t, tc.expectedCount, contribs)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, contribs)
			}
		})
	}
}

func TestNewGitHubScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitHubScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

func TestGetNumPrPages(t *testing.T) {
	testCases := []struct {
		desc          string
		client        graphql.Client
		expectedErr   error
		expectedPages int
	}{
		{
			desc:          "valid",
			client:        &mockClient{openPrCount: 110, mergedPrCount: 20},
			expectedErr:   nil,
			expectedPages: 2,
		},
		{
			desc:          "error on open pr count",
			client:        &mockClient{openPrCount: 10, err: true, errString: "this is an error"},
			expectedErr:   errors.New("this is an error"),
			expectedPages: 0,
		},
		{
			desc:          "error on merged pr count",
			client:        &mockClient{mergedPrCount: 10, err: true, errString: "this is an error"},
			expectedErr:   errors.New("this is an error"),
			expectedPages: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			now := pcommon.NewTimestampFromTime(time.Now())
			pages, err := getNumPrPages(ghs, context.Background(), tc.client, "repo", now)

			assert.Equal(t, tc.expectedPages, pages)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestGetNumBranchPages(t *testing.T) {
	testCases := []struct {
		desc          string
		client        graphql.Client
		expectedErr   error
		expectedPages int
	}{
		{
			desc:          "valid",
			client:        &mockClient{branchCount: 10},
			expectedErr:   nil,
			expectedPages: 1,
		},
		{
			desc:          "error",
			client:        &mockClient{branchCount: 10, err: true, errString: "this is an error"},
			expectedErr:   errors.New("this is an error"),
			expectedPages: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			now := pcommon.NewTimestampFromTime(time.Now())
			pages, err := getNumBranchPages(ghs, context.Background(), tc.client, "repo", now)

			assert.Equal(t, tc.expectedPages, pages)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestGetBranchData(t *testing.T) {
	testCases := []struct {
		desc                string
		client              graphql.Client
		expectedErr         error
		branchPages         int
		expectedBranchCount int
	}{
		{
			desc: "valid",
			client: &mockClient{
				branchData: getBranchDataRepositoryRefsRefConnection{
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: true,
					},
					Nodes: []BranchNode{
						{
							Name: "main",
						},
						{
							Name: "dev",
						},
						{
							Name: "feature",
						},
					},
				},
			},
			expectedErr:         nil,
			branchPages:         2,
			expectedBranchCount: 6,
		},
		{
			desc: "no next page",
			client: &mockClient{
				branchData: getBranchDataRepositoryRefsRefConnection{
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: false,
					},
					Nodes: []BranchNode{
						{
							Name: "main",
						},
						{
							Name: "dev",
						},
						{
							Name: "feature",
						},
					},
				},
			},
			expectedErr:         nil,
			branchPages:         2, // 3 PRs per page but shouldnt move to next page
			expectedBranchCount: 3,
		},
		{
			desc:                "error",
			client:              &mockClient{err: true, errString: "this is an error"},
			expectedErr:         errors.New("this is an error"),
			expectedBranchCount: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			branches, err := getBranchInfo(ghs, context.Background(), tc.client, "repo", "owner", 2, "main")
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
			assert.Equal(t, tc.expectedBranchCount, len(branches))
		})
	}
}

func TestGetPrData(t *testing.T) {
	testCases := []struct {
		desc            string
		client          graphql.Client
		expectedErr     error
		prPages         int
		expectedPrCount int
	}{
		{
			desc: "valid",
			client: &mockClient{
				prs: getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
						HasNextPage: true,
					},
					Nodes: []PullRequestNode{
						{
							CreatedAt: time.Now(),
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(24 * time.Hour), // 1 day later
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(48 * time.Hour), // 2 days later
							Merged:    false,
						},
					},
				},
			},
			expectedErr:     nil,
			prPages:         2, // 3 PRs per page
			expectedPrCount: 6,
		},
		{
			desc: "no next page",
			client: &mockClient{
				prs: getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
						HasNextPage: false,
					},
					Nodes: []PullRequestNode{
						{
							CreatedAt: time.Now(),
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(24 * time.Hour), // 1 day later
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(48 * time.Hour), // 2 days later
							Merged:    false,
						},
					},
				},
			},
			expectedErr:     nil,
			prPages:         2, // 3 PRs per page but shouldnt move to next page
			expectedPrCount: 3,
		},
		{
			desc:            "error",
			client:          &mockClient{err: true, errString: "this is an error"},
			expectedErr:     errors.New("this is an error"),
			expectedPrCount: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			pullRequests, err := getPrData(ghs, context.Background(), tc.client, 2, "repo", "owner")
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
			assert.Equal(t, tc.expectedPrCount, len(pullRequests))
		})
	}
}

func TestGetPullRequests(t *testing.T) {
	testCases := []struct {
		desc            string
		client          graphql.Client
		repo            SearchNodeRepository
		expectedErr     error
		expectedPrCount int
	}{
		{
			desc: "valid",
			client: &mockClient{
				prs: getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
						HasNextPage: true,
					},
					Nodes: []PullRequestNode{
						{
							CreatedAt: time.Now(),
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(24 * time.Hour), // 1 day later
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(48 * time.Hour), // 2 days later
							Merged:    false,
						},
					},
				},
				openPrCount:   110,
				mergedPrCount: 50,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:     nil,
			expectedPrCount: 6, // 3 PRs per page, 2 pages
		},
		{
			desc: "no next page",
			client: &mockClient{
				prs: getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
						HasNextPage: false,
					},
					Nodes: []PullRequestNode{
						{
							CreatedAt: time.Now(),
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(24 * time.Hour), // 1 day later
							Merged:    false,
						},
						{
							CreatedAt: time.Now().Add(48 * time.Hour), // 2 days later
							Merged:    false,
						},
					},
				},
				openPrCount:   110,
				mergedPrCount: 50,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:     nil,
			expectedPrCount: 3, // 3 PRs per page, 1 page
		},
		// {
		// 	desc:        "error",
		// 	client:      &mockClient{err: true, errString: "this is an error"},
		// 	expectedErr: errors.New("this is an error"),
		// },
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			now := pcommon.NewTimestampFromTime(time.Now())
			prs, err := getPullRequests(ghs, context.Background(), tc.client, tc.repo, now)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
			assert.Equal(t, tc.expectedPrCount, len(prs))
		})
	}
}
func TestGetBranches(t *testing.T) {
	testCases := []struct {
		desc                string
		client              graphql.Client
		repo                SearchNodeRepository
		expectedErr         error
		expectedBranchCount int
	}{
		{
			desc: "valid",
			client: &mockClient{
				branchData: getBranchDataRepositoryRefsRefConnection{
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: true,
					},
					Nodes: []BranchNode{
						{
							Name: "main",
						},
						{
							Name: "dev",
						},
						{
							Name: "feature",
						},
					},
				},
				branchCount: 3,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 3, //1 page, 3 branches per page
		},
		{
			desc: "three pages",
			client: &mockClient{
				branchData: getBranchDataRepositoryRefsRefConnection{
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: true,
					},
					Nodes: []BranchNode{
						{
							Name: "main",
						},
						{
							Name: "dev",
						},
						{
							Name: "feature",
						},
					},
				},
				branchCount: 110,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 9, // 3 pages, 3 branches per page

		},
		{
			desc: "three pages but no next page",
			client: &mockClient{
				branchData: getBranchDataRepositoryRefsRefConnection{
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: false,
					},
					Nodes: []BranchNode{
						{
							Name: "main",
						},
						{
							Name: "dev",
						},
						{
							Name: "feature",
						},
					},
				},
				branchCount: 110,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 3, //1 pages, 3 branches per page
		},
		{
			desc: "no branches",
			client: &mockClient{
				branchData: getBranchDataRepositoryRefsRefConnection{
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: true,
					},
					Nodes: []BranchNode{},
				},
				branchCount: 0,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 0,
		},
		{
			desc:                "error",
			client:              &mockClient{err: true, errString: "this is an error"},
			expectedErr:         errors.New("this is an error"),
			expectedBranchCount: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			now := pcommon.NewTimestampFromTime(time.Now())
			branches, err := ghs.getBranches(context.Background(), tc.client, tc.repo, now)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
			assert.Equal(t, tc.expectedBranchCount, len(branches))
		})
	}
}
