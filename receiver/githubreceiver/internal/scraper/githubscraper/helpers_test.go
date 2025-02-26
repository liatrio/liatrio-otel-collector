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

	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/go-github/v69/github"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type responses struct {
	searchRepoResponse    searchRepoResponse
	teamRepoResponse      teamRepoResponse
	prResponse            prResponse
	branchResponse        branchResponse
	commitResponse        commitResponse
	checkLoginResponse    loginResponse
	contribResponse       contribResponse
	depBotAlertResponse   depBotAlertResponse
	codeScanAlertResponse codeScanAlertResponse
	scrape                bool
}

type searchRepoResponse struct {
	repos        []getRepoDataBySearchSearchSearchResultItemConnection
	responseCode int
	page         int
}

type teamRepoResponse struct {
	repos        []getRepoDataByTeamOrganizationTeamRepositoriesTeamRepositoryConnection
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
	commits      []BranchHistoryTargetCommit
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

type depBotAlertResponse struct {
	depBotsAlerts []VulnerabilityAlerts
	responseCode  int
	page          int
}

type codeScanAlertResponse struct {
	codeScanAlerts [][]*github.Alert
	responseCode   int
	page           int
}

func MockServer(responses *responses) *http.ServeMux {
	var mux http.ServeMux
	contribRestEndpoint := "/api-v3/repos/o/r/contributors"
	codeScanRestEndpoint := "/api-v3/repos/o/r/code-scanning/alerts"

	graphEndpoint := "/"
	if responses.scrape {
		graphEndpoint = "/api/graphql"
		contribRestEndpoint = "/api/v3/repos/liatrio/repo1/contributors"
		codeScanRestEndpoint = "/api/v3/repos/liatrio/repo1/code-scanning/alerts"
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
		case reqBody.OpName == "getRepoDataByTeam":
			repoResp := &responses.teamRepoResponse
			w.WriteHeader(repoResp.responseCode)
			if repoResp.responseCode == http.StatusOK {
				repos := getRepoDataByTeamResponse{
					Organization: getRepoDataByTeamOrganization{
						Team: getRepoDataByTeamOrganizationTeam{
							Repositories: repoResp.repos[repoResp.page],
						},
					},
				}
				graphqlResponse := graphql.Response{Data: &repos}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				repoResp.page++
			}
		case reqBody.OpName == "getRepoDataBySearch":
			repoResp := &responses.searchRepoResponse
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
				branchHistory := []BranchHistory{
					{Target: &commitResp.commits[commitResp.page]},
				}
				commits := getCommitDataResponse{
					Repository: getCommitDataRepository{
						Refs: getCommitDataRepositoryRefsRefConnection{
							Nodes: branchHistory,
						},
					},
				}
				graphqlResponse := graphql.Response{Data: &commits}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				commitResp.page++
			}

		case reqBody.OpName == "getRepoCVEs":
			depBotAlertResp := &responses.depBotAlertResponse
			w.WriteHeader(depBotAlertResp.responseCode)
			if depBotAlertResp.responseCode == http.StatusOK {
				cves := getRepoCVEsResponse{
					Repository: getRepoCVEsRepository{
						VulnerabilityAlerts: depBotAlertResp.depBotsAlerts[depBotAlertResp.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &cves}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				depBotAlertResp.page++
			}
		}
	})
	mux.HandleFunc(contribRestEndpoint, func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc(codeScanRestEndpoint, func(w http.ResponseWriter, r *http.Request) {
		codeScanAlertResp := &responses.codeScanAlertResponse
		if codeScanAlertResp.responseCode == http.StatusOK {
			codeScanAlerts, err := json.Marshal(codeScanAlertResp.codeScanAlerts[codeScanAlertResp.page])
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			link := fmt.Sprintf(
				"<https://api.github.com/repositories/placeholder/code-scanning/alerts?per_page=50&page=%d>; rel=\"next\"",
				len(codeScanAlertResp.codeScanAlerts)-codeScanAlertResp.page-1,
			)
			w.Header().Set("Link", link)
			// Attempt to write data to the response writer.
			_, err = w.Write(codeScanAlerts)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
			codeScanAlertResp.page++
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

func TestGetAge(t *testing.T) {
	testCases := []struct {
		desc     string
		hrsAdd   time.Duration
		minsAdd  time.Duration
		expected float64
	}{
		{
			desc:     "TestHalfHourDiff",
			hrsAdd:   time.Duration(0) * time.Hour,
			minsAdd:  time.Duration(30) * time.Minute,
			expected: 60 * 30,
		},
		{
			desc:     "TestHourDiff",
			hrsAdd:   time.Duration(1) * time.Hour,
			minsAdd:  time.Duration(0) * time.Minute,
			expected: 60 * 60,
		},
		{
			desc:     "TestDayDiff",
			hrsAdd:   time.Duration(24) * time.Hour,
			minsAdd:  time.Duration(0) * time.Minute,
			expected: 60 * 60 * 24,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			min := time.Now()
			max := min.Add(tc.hrsAdd).Add(tc.minsAdd)

			actual := getAge(min, max)

			assert.Equal(t, int64(tc.expected), actual)
		})
	}
}

func TestGetSearchRepos(t *testing.T) {
	testCases := []struct {
		desc                    string
		server                  *http.ServeMux
		expectedErr             error
		expectedRepos           int
		expectedVulnerabilities int
	}{
		{
			desc: "TestSinglePageResponse",
			server: MockServer(&responses{
				scrape: false,
				searchRepoResponse: searchRepoResponse{
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 1,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo1",
									}},
							},
							PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedErr:             nil,
			expectedRepos:           1,
			expectedVulnerabilities: 0,
		},
		{
			desc: "TestMultiPageResponse",
			server: MockServer(&responses{
				scrape: false,
				searchRepoResponse: searchRepoResponse{
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 4,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo1",
									},
								},
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo2",
									},
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
									Repo: Repo{
										Name: "repo3",
									},
								},
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo4",
									},
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
			expectedErr:             nil,
			expectedRepos:           4,
			expectedVulnerabilities: 0,
		},
		{
			desc: "TestSinglePageWithVulnerabilitiesResponse",
			server: MockServer(&responses{
				scrape: false,
				searchRepoResponse: searchRepoResponse{
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 1,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo1",
									},
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
			expectedErr:             nil,
			expectedRepos:           1,
			expectedVulnerabilities: 2,
		},
		{
			desc: "TestMultiPageWithVulnerabilitieResponse",
			server: MockServer(&responses{
				scrape: false,
				searchRepoResponse: searchRepoResponse{
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 4,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo1",
									},
								},
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo2",
									},
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
									Repo: Repo{
										Name: "repo1",
									},
								},
								&SearchNodeRepository{
									Repo: Repo{
										Name: "repo4",
									},
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
			expectedErr:   nil,
			expectedRepos: 4,
			//expectedVulnerabilities: 8,
		},
		{
			desc: "Test404Response",
			server: MockServer(&responses{
				scrape: false,
				searchRepoResponse: searchRepoResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			expectedErr:   errors.New("returned error 404"),
			expectedRepos: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			ghs := newGitHubScraper(settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer func() { server.Close() }()
			client := graphql.NewClient(server.URL, ghs.client)

			_, count, err := ghs.getRepos(context.Background(), client, "fake query")
			assert.Equal(t, tc.expectedRepos, count)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestCheckOwnerExists(t *testing.T) {
	testCases := []struct {
		desc              string
		login             string
		expectedError     bool
		expectedOwnerType string
		server            *http.ServeMux
	}{
		{
			desc:  "TestOrgOwnerExists",
			login: "liatrio",
			server: MockServer(&responses{
				checkLoginResponse: loginResponse{
					checkLogin: checkLoginResponse{
						Organization: checkLoginOrganization{
							Login: "liatrio",
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedOwnerType: "org",
		},
		{
			desc:  "TestUserOwnerExists",
			login: "liatrio",
			server: MockServer(&responses{
				checkLoginResponse: loginResponse{
					checkLogin: checkLoginResponse{
						User: checkLoginUser{
							Login: "liatrio",
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedOwnerType: "user",
		},
		{
			desc:  "TestLoginError",
			login: "liatrio",
			server: MockServer(&responses{
				checkLoginResponse: loginResponse{
					checkLogin: checkLoginResponse{
						Organization: checkLoginOrganization{
							Login: "liatrio",
						},
					},
					responseCode: http.StatusNotFound,
				},
			}),
			expectedOwnerType: "",
			expectedError:     true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			ghs := newGitHubScraper(settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client := graphql.NewClient(server.URL, ghs.client)
			loginType, err := ghs.login(context.Background(), client, tc.login)

			assert.Equal(t, tc.expectedOwnerType, loginType)
			if !tc.expectedError {
				assert.NoError(t, err)
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
			expectedErr: errors.New("returned error 404"),
			expected:    0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			ghs := newGitHubScraper(settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()
			client := graphql.NewClient(server.URL, ghs.client)

			_, count, err := ghs.getBranches(context.Background(), client, "deathstarrepo", "main")

			assert.Equal(t, tc.expected, count)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
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
								ID: github.Ptr(int64(1)),
							},
							{
								ID: github.Ptr(int64(2)),
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
								ID: github.Ptr(int64(1)),
							},
							{
								ID: github.Ptr(int64(2)),
							},
						},
						{
							{
								ID: github.Ptr(int64(3)),
							},
							{
								ID: github.Ptr(int64(4)),
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
			settings := receivertest.NewNopSettings(metadata.Type)
			ghs := newGitHubScraper(settings, defaultConfig.(*Config))
			ghs.cfg.GitHubOrg = tc.org

			server := httptest.NewServer(tc.server)
			defer func() { server.Close() }()

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
			expectedErr:     errors.New("returned error 404"),
			expectedPrCount: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			ghs := newGitHubScraper(settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()
			client := graphql.NewClient(server.URL, ghs.client)

			prs, err := ghs.getPullRequests(context.Background(), client, "repo name")

			assert.Equal(t, tc.expectedPrCount, len(prs))
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestEvalCommits(t *testing.T) {
	testCases := []struct {
		desc              string
		server            *http.ServeMux
		expectedErr       error
		branch            BranchNode
		expectedAge       int64
		expectedAdditions int
		expectedDeletions int
	}{
		{
			desc: "TestNoBranchChanges",
			server: MockServer(&responses{
				scrape: false,
				commitResponse: commitResponse{
					commits: []BranchHistoryTargetCommit{
						{
							History: BranchHistoryTargetCommitHistoryCommitHistoryConnection{
								Nodes: []CommitNode{},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
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
		},
		{
			desc: "TestNoCommitsResponse",
			server: MockServer(&responses{
				scrape: false,
				commitResponse: commitResponse{
					commits: []BranchHistoryTargetCommit{
						{
							History: BranchHistoryTargetCommitHistoryCommitHistoryConnection{
								Nodes: []CommitNode{},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			branch: BranchNode{
				Name: "branch1",
				Compare: BranchNodeCompareComparison{
					AheadBy:  0,
					BehindBy: 1,
				},
			},
			expectedAge:       0,
			expectedAdditions: 0,
			expectedDeletions: 0,
			expectedErr:       nil,
		},
		{
			desc: "TestSinglePageResponse",
			server: MockServer(&responses{
				scrape: false,
				commitResponse: commitResponse{
					commits: []BranchHistoryTargetCommit{
						{
							History: BranchHistoryTargetCommitHistoryCommitHistoryConnection{
								Nodes: []CommitNode{
									{

										CommittedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
										Additions:     10,
										Deletions:     9,
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			branch: BranchNode{
				Name: "branch1",
				Compare: BranchNodeCompareComparison{
					AheadBy:  0,
					BehindBy: 1,
				},
			},
			expectedAge:       int64(time.Since(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)).Seconds()),
			expectedAdditions: 10,
			expectedDeletions: 9,
			expectedErr:       nil,
		},
		{
			desc: "TestMultiplePageResponse",
			server: MockServer(&responses{
				scrape: false,
				commitResponse: commitResponse{
					commits: []BranchHistoryTargetCommit{
						{
							History: BranchHistoryTargetCommitHistoryCommitHistoryConnection{
								Nodes: []CommitNode{
									{

										CommittedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
										Additions:     10,
										Deletions:     9,
									},
								},
							},
						},
						{
							History: BranchHistoryTargetCommitHistoryCommitHistoryConnection{
								Nodes: []CommitNode{
									{

										CommittedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
										Additions:     1,
										Deletions:     1,
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			branch: BranchNode{
				Name: "branch1",
				Compare: BranchNodeCompareComparison{
					AheadBy:  0,
					BehindBy: 101, // 100 per page, so this is 2 pages
				},
			},
			expectedAge:       int64(time.Since(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)).Seconds()),
			expectedAdditions: 11,
			expectedDeletions: 10,
			expectedErr:       nil,
		},
		{
			desc: "Test404ErrorResponse",
			server: MockServer(&responses{
				scrape: false,
				commitResponse: commitResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			branch: BranchNode{
				Name: "branch1",
				Compare: BranchNodeCompareComparison{
					AheadBy:  0,
					BehindBy: 1,
				},
			},
			expectedAge:       0,
			expectedAdditions: 0,
			expectedDeletions: 0,
			expectedErr:       errors.New("returned error 404"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			ghs := newGitHubScraper(settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()
			client := graphql.NewClient(server.URL, ghs.client)
			adds, dels, age, err := ghs.evalCommits(context.Background(), client, "repo1", tc.branch)

			assert.Equal(t, tc.expectedAge, age)
			assert.Equal(t, tc.expectedDeletions, dels)
			assert.Equal(t, tc.expectedAdditions, adds)

			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestGetCVEs(t *testing.T) {
	testCases := []struct {
		desc             string
		server           *http.ServeMux
		repo             string
		org              string
		expectedErr      error
		expectedCVECount int64
		expectedMap      map[metadata.AttributeCveSeverity]int64
	}{
		{
			desc: "TestSinglePageRespDepBotAlert",
			server: MockServer(&responses{
				scrape: false,
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{
						{
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "HIGH",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "MODERATE",
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedErr:      nil,
			expectedCVECount: 2,
			expectedMap: map[metadata.AttributeCveSeverity]int64{
				metadata.AttributeCveSeverityHigh:   1,
				metadata.AttributeCveSeverityMedium: 1,
			},
		},
		{
			desc: "TestMultiPageRespDepBotAlert",
			server: MockServer(&responses{
				scrape: false,
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{
						{
							PageInfo: VulnerabilityAlertsPageInfo{
								HasNextPage: true,
							},
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "HIGH",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "MODERATE",
									},
								},
							},
						},
						{
							PageInfo: VulnerabilityAlertsPageInfo{
								HasNextPage: false,
							},
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "HIGH",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "MODERATE",
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedErr:      nil,
			expectedCVECount: 4,
			expectedMap: map[metadata.AttributeCveSeverity]int64{
				metadata.AttributeCveSeverityHigh:   2,
				metadata.AttributeCveSeverityMedium: 2,
			},
		},
		{
			desc: "TestSinglePageRespCodeScanningAlert",
			server: MockServer(&responses{
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{{}},
					responseCode:  http.StatusOK,
				},
				codeScanAlertResponse: codeScanAlertResponse{
					codeScanAlerts: [][]*github.Alert{
						{
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("HIGH"),
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedMap: map[metadata.AttributeCveSeverity]int64{
				metadata.AttributeCveSeverityHigh: 1,
			},
			repo:             "r",
			org:              "o",
			expectedCVECount: 1,
			expectedErr:      nil,
		},
		{
			desc: "TestMultiPageRespCodeScanningAlert",
			server: MockServer(&responses{
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{{}},
					responseCode:  http.StatusOK,
				},
				codeScanAlertResponse: codeScanAlertResponse{
					codeScanAlerts: [][]*github.Alert{
						{
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("HIGH"),
								},
							},
						},
						{
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("HIGH"),
								},
							},
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("MEDIUM"),
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedMap: map[metadata.AttributeCveSeverity]int64{
				metadata.AttributeCveSeverityHigh:   2,
				metadata.AttributeCveSeverityMedium: 1,
			},
			repo:             "r",
			org:              "o",
			expectedCVECount: 3,
			expectedErr:      nil,
		},
		{
			desc: "TestSinglePageDepBotAndCodeScanningAlert",
			server: MockServer(&responses{
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{
						{
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "HIGH",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "MODERATE",
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
				codeScanAlertResponse: codeScanAlertResponse{
					codeScanAlerts: [][]*github.Alert{
						{
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("HIGH"),
								},
							},
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("MEDIUM"),
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedMap: map[metadata.AttributeCveSeverity]int64{
				metadata.AttributeCveSeverityHigh:   2,
				metadata.AttributeCveSeverityMedium: 2,
			},
			repo:             "r",
			org:              "o",
			expectedCVECount: 4,
			expectedErr:      nil,
		},
		{
			desc: "TestMultiPageDepBotAndCodeScanningAlert",
			server: MockServer(&responses{
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{
						{
							PageInfo: VulnerabilityAlertsPageInfo{
								HasNextPage: true,
							},
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "CRITICAL",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "MODERATE",
									},
								},
							},
						},
						{
							PageInfo: VulnerabilityAlertsPageInfo{
								HasNextPage: false,
							},
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "HIGH",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "LOW",
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
				codeScanAlertResponse: codeScanAlertResponse{
					codeScanAlerts: [][]*github.Alert{
						{
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("CRITICAL"),
								},
							},
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("LOW"),
								},
							},
						},
						{
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("HIGH"),
								},
							},
							{
								Rule: &github.Rule{
									SecuritySeverityLevel: github.Ptr("MEDIUM"),
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedMap: map[metadata.AttributeCveSeverity]int64{
				metadata.AttributeCveSeverityHigh:     2,
				metadata.AttributeCveSeverityMedium:   2,
				metadata.AttributeCveSeverityLow:      2,
				metadata.AttributeCveSeverityCritical: 2,
			},
			repo:             "r",
			org:              "o",
			expectedCVECount: 8,
			expectedErr:      nil,
		},
		{
			desc: "TestEmptyInputDepBotAndCodeScanningAlert",
			server: MockServer(&responses{
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{{}},
					responseCode:  http.StatusOK,
				},
				codeScanAlertResponse: codeScanAlertResponse{
					codeScanAlerts: [][]*github.Alert{{}},
					responseCode:   http.StatusOK,
				},
			}),
			expectedMap:      map[metadata.AttributeCveSeverity]int64{},
			repo:             "r",
			org:              "o",
			expectedCVECount: 0,
			expectedErr:      nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			ghs := newGitHubScraper(settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer func() { server.Close() }()
			ghs.cfg.GitHubOrg = tc.org

			gClient := graphql.NewClient(server.URL, ghs.client)
			rClient := github.NewClient(nil)

			url, err := url.Parse(server.URL + "/api-v3" + "/")
			assert.NoError(t, err)
			rClient.BaseURL = url
			rClient.UploadURL = url

			cves, err := ghs.getCVEs(context.Background(), gClient, rClient, tc.repo)
			totalCVEs := int64(0)

			for _, sevCount := range cves {
				totalCVEs += sevCount // Assuming 'cves' is a slice of VulnerabilityAlerts
			}

			assert.Equal(t, tc.expectedCVECount, totalCVEs)

			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}
