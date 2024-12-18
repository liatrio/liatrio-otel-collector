package gitlabscraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/Khan/genqlient/graphql"
)

type responses struct {
	projectResponse projectResponse
	branchResponse  branchResponse
	mrResponse      mrResponse
	contribResponse contribResponse
	compareResponse compareResponse
}

type branchResponse struct {
	branches     getBranchNamesProjectRepository
	responseCode int
}

type mrResponse struct {
	mrs          []getMergeRequestsProjectMergeRequestsMergeRequestConnection
	page         int
	responseCode int
}

type contribResponse struct {
	contribs     []*gitlab.Contributor
	responseCode int
}

type compareResponse struct {
	compare      *gitlab.Compare
	responseCode int
}

type projectResponse struct {
	projects     []*gitlab.Project
	responseCode int
}

func MockServer(responses *responses) *http.ServeMux {
	var mux http.ServeMux
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var reqBody graphql.Request
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			return
		}
		switch {
		// These OpNames need to be name of the GraphQL query as defined in genqlient.graphql
		case reqBody.OpName == "getBranchNames":
			branchResp := &responses.branchResponse
			w.WriteHeader(branchResp.responseCode)
			if branchResp.responseCode == http.StatusOK {
				branches := getBranchNamesResponse{
					Project: getBranchNamesProject{
						Repository: branchResp.branches,
					},
				}
				graphqlResponse := graphql.Response{Data: &branches}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
			}
		case reqBody.OpName == "getMergeRequests":
			mrResp := &responses.mrResponse
			w.WriteHeader(mrResp.responseCode)
			if mrResp.responseCode == http.StatusOK {
				mrs := getMergeRequestsResponse{
					Project: getMergeRequestsProject{
						MergeRequests: mrResp.mrs[mrResp.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &mrs}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				mrResp.page++

				//For getCombinedMergeRequests when the first call to getMergeRequests is finished
				//and the page count needs to be reset for the second or else you get index out of range
				if len(mrResp.mrs) == mrResp.page {
					mrResp.page = 0
				}
			}
		}
	})
	mux.HandleFunc("/api/v4/projects/project/repository/contributors", func(w http.ResponseWriter, r *http.Request) {
		contribResp := &responses.contribResponse
		if contribResp.responseCode == http.StatusOK {
			contribs, err := json.Marshal(contribResp.contribs)
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			_, err = w.Write(contribs)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
		}
	})

	mux.HandleFunc("/api/v4/projects/project/repository/compare", func(w http.ResponseWriter, r *http.Request) {
		compareResp := &responses.compareResponse
		if compareResp.responseCode == http.StatusOK {
			compare, err := json.Marshal(compareResp.compare)
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			_, err = w.Write(compare)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
		}
	})
	mux.HandleFunc("/api/v4/groups/project/projects", func(w http.ResponseWriter, r *http.Request) {
		projectResp := &responses.projectResponse
		if projectResp.responseCode == http.StatusOK {
			compare, err := json.Marshal(projectResp.projects)
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			_, err = w.Write(compare)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
		}
	})
	return &mux
}

func TestGetProjects(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		expectedErr   error
		expectedCount int
	}{
		{
			desc: "TestSingleProject",
			server: MockServer(&responses{
				projectResponse: projectResponse{
					projects: []*gitlab.Project{
						{
							Name:              "project1",
							PathWithNamespace: "project1",
							CreatedAt:         gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
							LastActivityAt:    gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			desc: "TestMultipleProjects",
			server: MockServer(&responses{
				projectResponse: projectResponse{
					projects: []*gitlab.Project{
						{
							Name:              "project1",
							PathWithNamespace: "project1",
							CreatedAt:         gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
							LastActivityAt:    gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
						},
						{
							Name:              "project2",
							PathWithNamespace: "project2",
							CreatedAt:         gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
							LastActivityAt:    gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
						},
						{
							Name:              "project3",
							PathWithNamespace: "project3",
							CreatedAt:         gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
							LastActivityAt:    gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 3,
			expectedErr:   nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			gls.cfg.GitLabOrg = "project"
			client, err := gitlab.NewClient("", gitlab.WithBaseURL(server.URL))
			assert.NoError(t, err)
			projects, err := gls.getProjects(client)

			assert.Equal(t, tc.expectedCount, len(projects))
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetContributorCount(t *testing.T) {
	testCases := []struct {
		desc          string
		resp          string
		expectedErr   error
		server        *http.ServeMux
		expectedCount int
	}{
		{
			desc: "TestSingleContributor",
			server: MockServer(&responses{
				contribResponse: contribResponse{
					contribs: []*gitlab.Contributor{
						{
							Name:    "contrib1",
							Commits: 1,
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			desc: "TestMultipleContributors",
			server: MockServer(&responses{
				contribResponse: contribResponse{
					contribs: []*gitlab.Contributor{
						{
							Name:    "contrib1",
							Commits: 1,
						},
						{
							Name:    "contrib2",
							Commits: 1,
						},
						{
							Name:    "contrib3",
							Commits: 1,
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 3,
			expectedErr:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()
			client, err := gitlab.NewClient("", gitlab.WithBaseURL(server.URL))
			assert.NoError(t, err)
			contribs, err := gls.getContributorCount(client, "project")

			assert.Equal(t, tc.expectedCount, contribs)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/*
 * Testing for getMergeRequests
 */
func TestGetMergeRequests(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		state         string
		expectedErr   error
		expectedCount int
	}{
		{
			desc: "TestSinglePage",
			server: MockServer(&responses{
				mrResponse: mrResponse{
					mrs: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr1",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			desc: "TestMultiplePages",
			server: MockServer(&responses{
				mrResponse: mrResponse{
					mrs: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr1",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: true,
							},
						},
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr2",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: true,
							},
						},
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr3",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 3,
			expectedErr:   nil,
		},
		{
			desc: "Test404Error",
			server: MockServer(&responses{
				mrResponse: mrResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			expectedCount: 0,
			expectedErr:   errors.New("returned error 404 Not Found: "),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client := graphql.NewClient(server.URL, gls.client)

			mergeRequestData, err := gls.getMergeRequests(context.Background(), client, "projectPath", MergeRequestState("opened"))

			assert.Equal(t, tc.expectedCount, len(mergeRequestData))
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetCombinedMergeRequests(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		state         string
		expectedErr   error
		expectedCount int
	}{
		{
			desc: "TestSinglePage",
			server: MockServer(&responses{
				mrResponse: mrResponse{
					mrs: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr1",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 2, //because theres two calls to getMergeRequests using this same data
			expectedErr:   nil,
		},
		{
			desc: "TestMultiplePages",
			server: MockServer(&responses{
				mrResponse: mrResponse{
					mrs: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr1",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: true,
							},
						},
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr2",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: true,
							},
						},
						{
							Nodes: []MergeRequestNode{
								{
									Title: "mr3",
								},
							},
							PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 6,
			expectedErr:   nil,
		},
		{
			desc: "Test404Error",
			server: MockServer(&responses{
				mrResponse: mrResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			expectedCount: 0,
			expectedErr:   errors.New("returned error 404 Not Found: "),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client := graphql.NewClient(server.URL, gls.client)

			mergeRequestData, err := gls.getCombinedMergeRequests(context.Background(), client, "projectPath")

			assert.Equal(t, tc.expectedCount, len(mergeRequestData))
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetBranchNames(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		expectedErr   error
		expectedCount int
	}{
		{
			desc: "TestSingleBranch",
			server: MockServer(&responses{
				branchResponse: branchResponse{
					branches: getBranchNamesProjectRepository{
						BranchNames: []string{"branch1"},
						RootRef:     "main",
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			desc: "TestMultipleBranches",
			server: MockServer(&responses{
				branchResponse: branchResponse{
					branches: getBranchNamesProjectRepository{
						BranchNames: []string{"branch1", "branch2", "branch3"},
						RootRef:     "main",
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 3,
			expectedErr:   nil,
		},
		{
			desc: "Test404Error",
			server: MockServer(&responses{
				branchResponse: branchResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			expectedCount: 0,
			expectedErr:   errors.New("returned error 404 Not Found: "),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client := graphql.NewClient(server.URL, gls.client)

			branches, err := gls.getBranchNames(context.Background(), client, "projectPath")

			if tc.expectedErr != nil {
				//nil is returned like this for some reason
				assert.Equal(t, (*getBranchNamesProjectRepository)(nil), branches)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, len(branches.BranchNames))

			}
		})
	}
}

func TestGetInitialCommit(t *testing.T) {
	testCases := []struct {
		desc           string
		resp           string
		expectedErr    error
		server         *http.ServeMux
		expectedCommit *gitlab.Commit
	}{
		{
			desc: "TestSingleCommit",
			server: MockServer(&responses{
				compareResponse: compareResponse{
					compare: &gitlab.Compare{
						Commits: []*gitlab.Commit{
							{
								Title: "commit1",
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCommit: &gitlab.Commit{
				Title: "commit1",
			},
		},
		{
			desc: "TestMultipleCommits",
			server: MockServer(&responses{
				compareResponse: compareResponse{
					compare: &gitlab.Compare{
						Commits: []*gitlab.Commit{
							{
								Title: "commit1",
							},
							{
								Title: "commit2",
							},
							{
								Title: "commit3",
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCommit: &gitlab.Commit{
				Title: "commit1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer func() { server.Close() }()
			client, err := gitlab.NewClient("", gitlab.WithBaseURL(server.URL))
			assert.NoError(t, err)
			commit, err := gls.getInitialCommit(client, "project", "defaultBranch", "branch")

			assert.Equal(t, tc.expectedCommit, commit)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
