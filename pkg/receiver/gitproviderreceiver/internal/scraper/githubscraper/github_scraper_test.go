// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type restResponse struct {
	responseCode int
	response     []*github.Contributor
	page         int
}

func createRestServer(response restResponse) *http.ServeMux {
	var mux http.ServeMux
	mux.HandleFunc("/api-v3/repos/o/r/contributors", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(response.responseCode)
		if response.responseCode == http.StatusOK {
			//need to write to specific headers to be able to paginate correctly, for now only does 1 page
			contributors, _ := json.Marshal(response.response)
			w.Write(contributors)
			//response.page++
		}
	})
	return &mux
}

func TestGetContributors(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		repo          string
		org           string
		expectedErr   error
		expectedCount int
	}{
		{
			//currently the mock http server can only return one page for this function
			desc: "one page",
			server: createRestServer(restResponse{
				response: []*github.Contributor{

					{
						ID: github.Int64(1),
					},
					{
						ID: github.Int64(2),
					},
				},
				responseCode: http.StatusOK,
			}),
			repo:          "r",
			org:           "o",
			expectedErr:   nil,
			expectedCount: 2,
		},
		{
			//currently the mock http server can only return one page for this function
			desc: "404 error",
			server: createRestServer(restResponse{
				response: []*github.Contributor{

					{
						ID: github.Int64(1),
					},
					{
						ID: github.Int64(2),
					},
				},
				responseCode: http.StatusNotFound,
			}),
			repo:          "r",
			org:           "o",
			expectedErr:   errors.New("GET http://127.0.0.1:52832/api-v3/repos/o/r/contributors?per_page=100: 404  []"),
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

			//This part lets us have the same port and can test the 404 error
			//Not sure if this is the best way to do this or worth it
			l, err := net.Listen("tcp", "127.0.0.1:52832")
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewUnstartedServer(tc.server)
			server.Listener.Close()
			server.Listener = l
			server.Start()
			defer server.Close()

			client := github.NewClient(nil)
			url, _ := url.Parse(server.URL + "/api-v3" + "/")
			client.BaseURL = url
			client.UploadURL = url

			contribs, err := ghs.getContributorCount(context.Background(), client, tc.repo)
			if tc.expectedErr != nil {
				assert.Error(t, err)
				//server url changes every time, need to handle somehow
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
		server          *http.ServeMux
		expectedErr     error
		expectedPrCount int
	}{
		{
			desc: "one page",
			server: createServer("/", &responses{
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
				responseCode: http.StatusOK,
			}),
			expectedErr:     nil,
			expectedPrCount: 3, // 3 PRs per page, 1 pages
		},
		{
			desc: "multiple pages",
			server: createServer("/", &responses{
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
				responseCode: http.StatusOK,
			}),
			expectedErr:     nil,
			expectedPrCount: 6, // 3 PRs per page, 2 pages
		},
		{
			desc: "404 not found",
			server: createServer("/", &responses{
				responseCode: http.StatusNotFound,
			}),
			expectedErr:     errors.New("returned error 404 Not Found: "),
			expectedPrCount: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()
			client := graphql.NewClient(server.URL, ghs.client)

			prs, err := ghs.getPullRequests(context.Background(), client, "repo name")

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
						TotalCount: 0,
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
						TotalCount: 3,
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
						TotalCount: 9,
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
						TotalCount: 9,
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
						TotalCount: 9,
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
			_, count, err := ghs.getBranches(context.Background(), tc.client, "repo", "main")

			assert.Equal(t, tc.expectedBranchCount, count)
			//for now this is just meant to use the branch variable so that
			//assert.NotEqual(t, nil, branches)
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

func TestGetRepoData(t *testing.T) {
	testCases := []struct {
		desc          string
		client        graphql.Client
		expectedErr   error
		expectedRepos int
	}{
		{
			desc: "no pages",
			client: &mockClient{repoData: []getRepoDataBySearchSearchSearchResultItemConnection{
				{
					Nodes: []SearchNode{},
					PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
						HasNextPage: false,
					},
				},
			}},
			expectedErr:   nil,
			expectedRepos: 0,
		},
		{
			desc: "valid, one page",
			client: &mockClient{repoData: []getRepoDataBySearchSearchSearchResultItemConnection{
				{
					Nodes: []SearchNode{
						&SearchNodeRepository{
							Name: "repo1",
						},
					},
					PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
						HasNextPage: false,
					},
				},
			}},
			expectedErr:   nil,
			expectedRepos: 1,
		},
		{
			desc: "valid, 3 pages",
			client: &mockClient{repoData: []getRepoDataBySearchSearchSearchResultItemConnection{
				{
					Nodes: []SearchNode{
						&SearchNodeRepository{
							Name: "repo1",
						},
					},
					PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
						HasNextPage: true,
					},
				},
				{
					Nodes: []SearchNode{
						&SearchNodeRepository{
							Name: "repo2",
						},
					},
					PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
						HasNextPage: true,
					},
				},
				{
					Nodes: []SearchNode{
						&SearchNodeRepository{
							Name: "repo3",
						},
					},
					PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
						HasNextPage: false,
					},
				},
			}},
			expectedErr:   nil,
			expectedRepos: 3,
		},
		{
			desc:        "error",
			client:      &mockClient{err: true, errString: "this is an error"},
			expectedErr: errors.New("this is an error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			repos, count, err := ghs.getRepos(context.Background(), tc.client, "search query")

			assert.Equal(t, tc.expectedRepos, count)
			assert.NotEqual(t, nil, repos)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, tc.expectedErr, err.Error())
			}
		})
	}
}
