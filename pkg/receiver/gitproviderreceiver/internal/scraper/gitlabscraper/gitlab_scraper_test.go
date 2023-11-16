// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/Khan/genqlient/graphql"
)

/*
 * Testing for newGitLabScraper
 */
func TestNewGitLabScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitLabScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

type responses struct {
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
			responses := &responses.branchResponse
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				branches := getBranchNamesResponse{
					Project: getBranchNamesProject{
						Repository: responses.branches,
					},
				}
				graphqlResponse := graphql.Response{Data: &branches}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
			}
		case reqBody.OpName == "getMergeRequests":
			responses := &responses.mrResponse
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				mrs := getMergeRequestsResponse{
					Project: getMergeRequestsProject{
						MergeRequests: responses.mrs[responses.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &mrs}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				responses.page++

				//For getCombinedMergeRequests when the first call to getMergeRequests is finished
				//and the page count needs to be reset for the second or else you get index out of range
				if len(responses.mrs) == responses.page {
					responses.page = 0
				}
			}
		}
	})
	mux.HandleFunc("/api/v4/projects/project/repository/contributors", func(w http.ResponseWriter, r *http.Request) {
		responses := &responses.contribResponse
		if responses.responseCode == http.StatusOK {
			contribs, err := json.Marshal(responses.contribs)
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
		responses := &responses.compareResponse
		if responses.responseCode == http.StatusOK {
			compare, err := json.Marshal(responses.compare)
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
			settings := receivertest.NewNopCreateSettings()
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
			settings := receivertest.NewNopCreateSettings()
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
			settings := receivertest.NewNopCreateSettings()
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
