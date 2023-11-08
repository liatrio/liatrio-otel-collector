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
	prs        []getPullRequestDataRepositoryPullRequestsPullRequestConnection
	branchData []getBranchDataRepositoryRefsRefConnection
	repoData   []getRepoDataBySearchSearchSearchResultItemConnection
	commitData CommitNodeTargetCommit
	curPage    int
}

type responses struct {
	responseCode int
	checkLogin   checkLoginResponse
	repos        []getRepoDataBySearchSearchSearchResultItemConnection
	branches     []getBranchDataRepositoryRefsRefConnection
	prs          []getPullRequestDataRepositoryPullRequestsPullRequestConnection
	page         int
	contribs     []*github.Contributor
}

func (m *mockClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	switch op := req.OpName; op {
	case "getPullRequestData":
		//for forcing arbitrary errors
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getPullRequestDataResponse)
		r.Repository.PullRequests = m.prs[m.curPage]
		m.curPage++

	case "getBranchData":
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getBranchDataResponse)
		r.Repository.Refs = m.branchData[m.curPage]
		m.curPage++

	case "getCommitData":
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getCommitDataResponse)
		commitNodes := []CommitNode{
			{Target: &m.commitData},
		}
		r.Repository.Refs.Nodes = commitNodes

	case "getRepoDataBySearch":
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getRepoDataBySearchResponse)
		r.Search = m.repoData[m.curPage]
		m.curPage++
	}
	return nil
}

func graphqlMockServer(responses *responses) *http.ServeMux {
	var mux http.ServeMux
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var reqBody graphql.Request
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			return
		}
		switch {
		// These OpNames need to be name of the GraphQL query as defined in genqlient.graphql
		case reqBody.OpName == "checkLogin":
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				login := responses.checkLogin
				graphqlResponse := graphql.Response{Data: &login}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
			}
		case reqBody.OpName == "getRepoDataBySearch":
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				repos := getRepoDataBySearchResponse{
					Search: responses.repos[responses.page],
				}
				graphqlResponse := graphql.Response{Data: &repos}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				responses.page++
			}
		case reqBody.OpName == "getBranchData":
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				repos := getBranchDataResponse{
					Repository: getBranchDataRepository{
						Refs: responses.branches[responses.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &repos}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				responses.page++
			}
		case reqBody.OpName == "getPullRequestData":
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				repos := getPullRequestDataResponse{
					Repository: getPullRequestDataRepository{
						PullRequests: responses.prs[responses.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &repos}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				responses.page++
			}
		}
	})
	return &mux
}

func restMockServer(resp responses) *http.ServeMux {
	var mux http.ServeMux
	mux.HandleFunc("/api-v3/repos/o/r/contributors", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(resp.responseCode)
		if resp.responseCode == http.StatusOK {
			contribs, _ := json.Marshal(resp.contribs)
			// Attempt to write data to the response writer.
			_, err := w.Write(contribs)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
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

func TestSubInt(t *testing.T) {
	a := 100
	b := 10

	expected := 90

	num := sub(a, b)

	assert.Equal(t, expected, num)
}

func TestSubFloat(t *testing.T) {
	a := 10.5
	b := 10.5

	expected := 0.0

	num := sub(a, b)

	assert.Equal(t, expected, num)
}

func TestSubNegativeInt(t *testing.T) {
	a := 1
	b := -1

	expected := 2

	num := sub(a, b)

	assert.Equal(t, expected, num)
}

func TestSubNegativeFloat(t *testing.T) {
	a := 1.5
	b := -10.0

	expected := 11.5

	num := sub(a, b)

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

func TestCheckOwnerTypeValid(t *testing.T) {
	validOptions := []string{"org", "user"}

	for _, option := range validOptions {
		valid, err := checkOwnerTypeValid(option)

		assert.True(t, valid)
		assert.Nil(t, err)
	}
}

func TestCheckOwnerTypeValidRandom(t *testing.T) {
	invalidOptions := []string{"sorg", "suser", "users", "orgs", "invalid", "text"}

	for _, option := range invalidOptions {
		valid, err := checkOwnerTypeValid(option)

		assert.False(t, valid)
		assert.NotNil(t, err)
	}
}

func TestCheckOwnerExists(t *testing.T) {
	testCases := []struct {
		desc                string
		login               string
		expectedError       bool
		expectedOwnerType   string
		expectedOwnerExists bool
		server              *http.ServeMux
	}{
		{
			desc:  "TestOrgOwnerExists",
			login: "liatrio",
			server: graphqlMockServer(&responses{
				checkLogin: checkLoginResponse{
					Organization: checkLoginOrganization{
						Login: "liatrio",
					},
				},
				responseCode: http.StatusOK,
			}),
			expectedOwnerType:   "org",
			expectedOwnerExists: true,
		},
		{
			desc:  "TestUserOwnerExists",
			login: "liatrio",
			server: graphqlMockServer(&responses{
				checkLogin: checkLoginResponse{
					User: checkLoginUser{
						Login: "liatrio",
					},
				},
				responseCode: http.StatusOK,
			}),
			expectedOwnerType:   "user",
			expectedOwnerExists: true,
		},
		{
			desc:  "TestLoginError",
			login: "liatrio",
			server: graphqlMockServer(&responses{
				checkLogin: checkLoginResponse{
					User: checkLoginUser{
						Login: "liatrio",
					},
				},
				responseCode: http.StatusNotFound,
			}),
			expectedOwnerExists: false,
			expectedOwnerType:   "",
			expectedError:       true,
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
			ownerExists, ownerType, err := ghs.checkOwnerExists(context.Background(), client, tc.login)

			assert.Equal(t, tc.expectedOwnerExists, ownerExists)
			assert.Equal(t, tc.expectedOwnerType, ownerType)
			if !tc.expectedError {
				assert.NoError(t, err)
			}
		})
	}
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
			server: graphqlMockServer(&responses{
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
			}),
			expectedErr: nil,
			expected:    1,
		},
		{
			desc: "TestMultiPageResponse",
			server: graphqlMockServer(&responses{
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
			}),
			expectedErr: nil,
			expected:    4,
		},
		{
			desc: "Test404Response",
			server: graphqlMockServer(&responses{
				responseCode: http.StatusNotFound,
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
			server: graphqlMockServer(&responses{
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
			}),
			expectedErr: nil,
			expected:    1,
		},
		{
			desc: "TestMultiPageResponse",
			server: graphqlMockServer(&responses{
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
			}),
			expectedErr: nil,
			expected:    4,
		},
		{
			desc: "Test404Response",
			server: graphqlMockServer(&responses{
				responseCode: http.StatusNotFound,
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
			desc: "TestListContributorsResponse",
			server: restMockServer(responses{
				contribs: []*github.Contributor{

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
			url, _ := url.Parse(server.URL + "/api-v3" + "/")
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
			server: graphqlMockServer(&responses{
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
			desc: "TestMultiPageResponse",
			server: graphqlMockServer(&responses{
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
			desc: "Test404Response",
			server: graphqlMockServer(&responses{
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
