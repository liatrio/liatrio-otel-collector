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
// The github client this library gives us doesn't use interfaces so the current mocking setup doesn't work with it
// and had to find a different way to create tests
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
	url, err := url.Parse(server.URL + baseURLPath + "/")
	if err != nil {
		return nil, nil, "", nil
	}
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
			expectedErr:   errors.New("GET " + client.BaseURL.String() + "repos/junk/junk/contributors: 404  []"),
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

			contribs, err := ghs.getContributorCount(ctx, client, tc.repo, now)
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
func TestGetPullRequests(t *testing.T) {
	testCases := []struct {
		desc            string
		client          graphql.Client
		expectedErr     error
		expectedPrCount int
	}{
		{
			desc: "no page",
			client: &mockClient{
				prs: []getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					{
						PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
							HasNextPage: false,
						},
					},
				},
			},
			expectedErr:     nil,
			expectedPrCount: 0,
		},
		{
			desc: "one page",
			client: &mockClient{
				prs: []getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					{
						PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
							HasNextPage: false,
						},
						Nodes: []PullRequestNode{
							{
								Merged: false,
							},
							{
								Merged: false,
							},
							{
								Merged: false,
							},
						},
					},
				},
			},
			expectedErr:     nil,
			expectedPrCount: 3, // 3 PRs per page, 1 pages
		},
		{
			desc: "multiple pages",
			client: &mockClient{
				prs: []getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					{
						PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
							HasNextPage: true,
						},
						Nodes: []PullRequestNode{
							{
								Merged: false,
							},
							{
								Merged: false,
							},
							{
								Merged: false,
							},
						},
					},
					{
						PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
							HasNextPage: true,
						},
						Nodes: []PullRequestNode{
							{
								Merged: false,
							},
							{
								Merged: false,
							},
							{
								Merged: false,
							},
						},
					},
					{
						PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
							HasNextPage: false,
						},
						Nodes: []PullRequestNode{
							{
								Merged: false,
							},
							{
								Merged: false,
							},
							{
								Merged: false,
							},
						},
					},
				},
			},
			expectedErr:     nil,
			expectedPrCount: 9, // 3 PRs per page, 3 pages
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
			prs, err := ghs.getPullRequests(context.Background(), tc.client, "repo name")

			assert.Equal(t, tc.expectedPrCount, len(prs))
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}
func TestGetBranches(t *testing.T) {
	testCases := []struct {
		desc                string
		client              graphql.Client
		expectedErr         error
		branchPages         int
		expectedBranchCount int
	}{
		{
			desc: "no pages",
			client: &mockClient{
				branchData: []getBranchDataRepositoryRefsRefConnection{
					{
						PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
							HasNextPage: false,
						},
					},
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 0,
		},
		{
			desc: "one page",
			client: &mockClient{
				branchData: []getBranchDataRepositoryRefsRefConnection{
					{
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
			},
			expectedErr:         nil,
			expectedBranchCount: 3,
		},
		{
			desc: "multiple pages",
			client: &mockClient{
				branchData: []getBranchDataRepositoryRefsRefConnection{
					{
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
					{
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
					{
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
			},
			expectedErr:         nil,
			expectedBranchCount: 9,
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
			branches, err := ghs.getBranches(context.Background(), tc.client, "repo", "main")

			assert.Equal(t, tc.expectedBranchCount, len(branches))
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestGetCommitInfo(t *testing.T) {
	testCases := []struct {
		desc        string
		client      graphql.Client
		expectedErr error
		pages       int
		branch      BranchNode
		//commits      CommitNodeTargetCommit
		expectedAge       int64
		expectedAdditions int
		expectedDeletions int
	}{
		{
			desc: "valid",
			client: &mockClient{commitData: CommitNodeTargetCommit{
				History: CommitNodeTargetCommitHistoryCommitHistoryConnection{
					Edges: []CommitNodeTargetCommitHistoryCommitHistoryConnectionEdgesCommitEdge{
						{
							Node: CommitNodeTargetCommitHistoryCommitHistoryConnectionEdgesCommitEdgeNodeCommit{
								CommittedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
								Additions:     10,
								Deletions:     9,
							},
						},
					},
				},
			}},
			branch: BranchNode{
				Name: "branch1",
				Compare: BranchNodeCompareComparison{
					AheadBy:  0,
					BehindBy: 1,
				},
			},
			expectedAge:       int64(time.Since(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)).Hours()),
			expectedAdditions: 10,
			expectedDeletions: 9,
			expectedErr:       nil,
			pages:             1,
		},
		{
			desc: "valid with multiple pages",
			client: &mockClient{commitData: CommitNodeTargetCommit{
				History: CommitNodeTargetCommitHistoryCommitHistoryConnection{
					Edges: []CommitNodeTargetCommitHistoryCommitHistoryConnectionEdgesCommitEdge{
						{
							Node: CommitNodeTargetCommitHistoryCommitHistoryConnectionEdgesCommitEdgeNodeCommit{
								CommittedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
								Additions:     10,
								Deletions:     9,
							},
						},
					},
				},
			}},
			branch: BranchNode{
				Name: "branch1",
				Compare: BranchNodeCompareComparison{
					AheadBy:  0,
					BehindBy: 1,
				},
			},
			expectedAge:       int64(time.Since(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)).Hours()),
			expectedAdditions: 20,
			expectedDeletions: 18,
			expectedErr:       nil,
			pages:             2,
		},
		{
			desc: "no commits",
			client: &mockClient{commitData: CommitNodeTargetCommit{
				History: CommitNodeTargetCommitHistoryCommitHistoryConnection{
					Edges: []CommitNodeTargetCommitHistoryCommitHistoryConnectionEdgesCommitEdge{},
				},
			}},
			branch: BranchNode{
				Name: "branch1",
				Compare: BranchNodeCompareComparison{
					AheadBy:  0,
					BehindBy: 0,
				},
			},
			expectedAge:       0,
			expectedAdditions: 0,
			expectedDeletions: 0,
			expectedErr:       nil,
			pages:             1,
		},
		{
			desc:              "no pages to iterate over",
			pages:             0,
			expectedAge:       0,
			expectedAdditions: 0,
			expectedDeletions: 0,
		},
		{
			desc:              "error",
			client:            &mockClient{err: true, errString: "this is an error"},
			expectedErr:       errors.New("this is an error"),
			pages:             1,
			expectedAge:       0,
			expectedAdditions: 0,
			expectedDeletions: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			now := pcommon.NewTimestampFromTime(time.Now())
			adds, dels, age, err := ghs.getCommitInfo(context.Background(), tc.client, "repo1", now, tc.pages, tc.branch)

			assert.Equal(t, tc.expectedAge, age)
			assert.Equal(t, tc.expectedDeletions, dels)
			assert.Equal(t, tc.expectedAdditions, adds)

			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}
