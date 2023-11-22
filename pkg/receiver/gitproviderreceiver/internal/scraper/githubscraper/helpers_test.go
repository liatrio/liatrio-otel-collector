package githubscraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type mockClient struct {
	err        bool
	errString  string
	commitData CommitNodeTargetCommit
}

type responses struct {
	repoResponse       repoResponse
	prResponse         prResponse
	branchResponse     branchResponse
	commitResponse     commitResponse
	checkLoginResponse loginResponse
	contribResponse    contribResponse
	scrape             bool
}

type repoResponse struct {
	repos        []getRepoDataBySearchSearchSearchResultItemConnection
	responseCode int
	page         int
}

type prResponse struct {
	prs          []getPullRequestDataRepositoryPullRequestsPullRequestConnection
	responseCode int
	page         int
}

type branchResponse struct {
	branches     []getBranchDataRepositoryRefsRefConnection
	responseCode int
	page         int
}

type commitResponse struct {
	commits      []CommitNodeTargetCommit
	responseCode int
	page         int
}

type loginResponse struct {
	checkLogin   checkLoginResponse
	responseCode int
}

type contribResponse struct {
	contribs     [][]*github.Contributor
	responseCode int
	page         int
}

func (m *mockClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	switch op := req.OpName; op {
	case "getCommitData":
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getCommitDataResponse)
		commitNodes := []CommitNode{
			{Target: &m.commitData},
		}
		r.Repository.Refs.Nodes = commitNodes
	}
	return nil
}

func MockServer(responses *responses) *http.ServeMux {
	var mux http.ServeMux
	restEndpoint := "/api-v3/repos/o/r/contributors"
	graphEndpoint := "/"
	if responses.scrape {
		graphEndpoint = "/api/graphql"
		restEndpoint = "/api/v3/repos/liatrio/repo1/contributors"
	}
	mux.HandleFunc(graphEndpoint, func(w http.ResponseWriter, r *http.Request) {
		var reqBody graphql.Request
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			return
		}
		switch {
		// These OpNames need to be name of the GraphQL query as defined in genqlient.graphql
		case reqBody.OpName == "checkLogin":
			loginResp := &responses.checkLoginResponse
			w.WriteHeader(loginResp.responseCode)
			if loginResp.responseCode == http.StatusOK {
				login := loginResp.checkLogin
				graphqlResponse := graphql.Response{Data: &login}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
			}
		case reqBody.OpName == "getRepoDataBySearch":
			repoResp := &responses.repoResponse
			w.WriteHeader(repoResp.responseCode)
			if repoResp.responseCode == http.StatusOK {
				repos := getRepoDataBySearchResponse{
					Search: repoResp.repos[repoResp.page],
				}
				graphqlResponse := graphql.Response{Data: &repos}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				repoResp.page++
			}
		case reqBody.OpName == "getBranchData":
			branchResp := &responses.branchResponse
			w.WriteHeader(branchResp.responseCode)
			if branchResp.responseCode == http.StatusOK {
				branches := getBranchDataResponse{
					Repository: getBranchDataRepository{
						Refs: branchResp.branches[branchResp.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &branches}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				branchResp.page++
			}
		case reqBody.OpName == "getPullRequestData":
			prResp := &responses.prResponse
			w.WriteHeader(prResp.responseCode)
			if prResp.responseCode == http.StatusOK {
				repos := getPullRequestDataResponse{
					Repository: getPullRequestDataRepository{
						PullRequests: prResp.prs[prResp.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &repos}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				prResp.page++
			}

		case reqBody.OpName == "getCommitData":
			commitResp := &responses.commitResponse
			w.WriteHeader(commitResp.responseCode)
			if commitResp.responseCode == http.StatusOK {
				commitNodes := []CommitNode{
					{Target: &commitResp.commits[commitResp.page]},
				}
				commits := getCommitDataResponse{
					Repository: getCommitDataRepository{
						Refs: getCommitDataRepositoryRefsRefConnection{
							Nodes: commitNodes,
						},
					},
				}
				graphqlResponse := graphql.Response{Data: &commits}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				commitResp.page++
			}
		}
	})
	mux.HandleFunc(restEndpoint, func(w http.ResponseWriter, r *http.Request) {
		contribResp := &responses.contribResponse
		if contribResp.responseCode == http.StatusOK {
			contribs, err := json.Marshal(contribResp.contribs[contribResp.page])
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			link := fmt.Sprintf(
				"<https://api.github.com/repositories/placeholder/contributors?per_page=100&page=%d>; rel=\"next\"",
				len(contribResp.contribs)-contribResp.page-1,
			)
			w.Header().Set("Link", link)
			// Attempt to write data to the response writer.
			_, err = w.Write(contribs)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
			contribResp.page++
		}
	})
	return &mux
}

func TestGetNumPages100(t *testing.T) {
	p := float64(100)
	n := float64(375)

	expected := 4

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestGetNumPages10(t *testing.T) {
	p := float64(10)
	n := float64(375)

	expected := 38

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestGetNumPages1(t *testing.T) {
	p := float64(10)
	n := float64(1)

	expected := 1

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestAddInt(t *testing.T) {
	a := 100
	b := 100

	expected := 200

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddZero(t *testing.T) {
	a := 0
	b := 1

	expected := 1

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddFloat(t *testing.T) {
	a := 10.5
	b := 10.5

	expected := 21.0

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddNegativeInt(t *testing.T) {
	a := 1
	b := -1

	expected := 0

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddNegativeFloat(t *testing.T) {
	a := 1.5
	b := -10.0

	expected := -8.5

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestGenDefaultSearchQueryOrg(t *testing.T) {
	st := "org"
	org := "empire"

	expected := "org:empire archived:false"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}

func TestGenDefaultSearchQueryUser(t *testing.T) {
	st := "user"
	org := "vader"

	expected := "user:vader archived:false"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}

func TestGetRepos(t *testing.T) {
	testCases := []struct {
		desc        string
		server      *http.ServeMux
		expectedErr error
		expected    int
	}{
		{
			desc: "TestSinglePageResponse",
			server: MockServer(&responses{
				scrape: false,
				repoResponse: repoResponse{
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 1,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Name: "repo1",
								},
							},
							PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedErr: nil,
			expected:    1,
		},
		{
			desc: "TestMultiPageResponse",
			server: MockServer(&responses{
				scrape: false,
				repoResponse: repoResponse{
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 4,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Name: "repo1",
								},
								&SearchNodeRepository{
									Name: "repo2",
								},
							},
							PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
								HasNextPage: true,
							},
						},
						{
							RepositoryCount: 4,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Name: "repo3",
								},
								&SearchNodeRepository{
									Name: "repo4",
								},
							},
							PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedErr: nil,
			expected:    4,
		},
		{
			desc: "Test404Response",
			server: MockServer(&responses{
				scrape: false,
				repoResponse: repoResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			expectedErr: errors.New("returned error 404 Not Found: "),
			expected:    0,
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

			_, count, err := ghs.getRepos(context.Background(), client, "fake query")

			assert.Equal(t, tc.expected, count)
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
		desc        string
		server      *http.ServeMux
		expectedErr error
		expected    int
	}{
		{
			desc: "TestSinglePageResponse",
			server: MockServer(&responses{
				scrape: false,
				branchResponse: branchResponse{
					branches: []getBranchDataRepositoryRefsRefConnection{
						{
							TotalCount: 1,
							Nodes: []BranchNode{
								{
									Name: "main",
								},
							},
							PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedErr: nil,
			expected:    1,
		},
		{
			desc: "TestMultiPageResponse",
			server: MockServer(&responses{
				scrape: false,
				branchResponse: branchResponse{
					branches: []getBranchDataRepositoryRefsRefConnection{
						{
							TotalCount: 4,
							Nodes: []BranchNode{
								{
									Name: "main",
								},
								{
									Name: "vader",
								},
							},
							PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
								HasNextPage: true,
							},
						},
						{
							TotalCount: 4,
							Nodes: []BranchNode{
								{
									Name: "skywalker",
								},
								{
									Name: "rebelalliance",
								},
							},
							PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedErr: nil,
			expected:    4,
		},
		{
			desc: "Test404Response",
			server: MockServer(&responses{
				scrape: false,
				branchResponse: branchResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			expectedErr: errors.New("returned error 404 Not Found: "),
			expected:    0,
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

			_, count, err := ghs.getBranches(context.Background(), client, "deathstarrepo", "main")

			assert.Equal(t, tc.expected, count)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
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
			desc: "TestSingleListContributorsResponse",
			server: MockServer(&responses{
				scrape: false,
				contribResponse: contribResponse{
					contribs: [][]*github.Contributor{
						{
							{
								ID: github.Int64(1),
							},
							{
								ID: github.Int64(2),
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			repo:          "r",
			org:           "o",
			expectedErr:   nil,
			expectedCount: 2,
		},
		{
			desc: "TestMultipleListContributorsResponse",
			server: MockServer(&responses{
				contribResponse: contribResponse{
					contribs: [][]*github.Contributor{
						{
							{
								ID: github.Int64(1),
							},
							{
								ID: github.Int64(2),
							},
						},
						{
							{
								ID: github.Int64(3),
							},
							{
								ID: github.Int64(4),
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			repo:          "r",
			org:           "o",
			expectedErr:   nil,
			expectedCount: 4,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			ghs.cfg.GitHubOrg = tc.org

			server := httptest.NewServer(tc.server)

			client := github.NewClient(nil)
			url, err := url.Parse(server.URL + "/api-v3" + "/")
			assert.NoError(t, err)
			client.BaseURL = url
			client.UploadURL = url

			contribs, err := ghs.getContributorCount(context.Background(), client, tc.repo)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCount, contribs)
		})
	}
}

func TestGetPullRequests(t *testing.T) {
	testCases := []struct {
		desc            string
		server          *http.ServeMux
		expectedErr     error
		expectedPrCount int
	}{
		{
			desc: "TestSinglePageResponse",
			server: MockServer(&responses{
				scrape: false,
				prResponse: prResponse{
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
				},
			}),
			expectedErr:     nil,
			expectedPrCount: 3, // 3 PRs per page, 1 pages
		},
		{
			desc: "TestMultiPageResponse",
			server: MockServer(&responses{
				scrape: false,
				prResponse: prResponse{
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
				},
			}),
			expectedErr:     nil,
			expectedPrCount: 6, // 3 PRs per page, 2 pages
		},
		{
			desc: "Test404Response",
			server: MockServer(&responses{
				scrape: false,
				prResponse: prResponse{
					responseCode: http.StatusNotFound,
				},
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
