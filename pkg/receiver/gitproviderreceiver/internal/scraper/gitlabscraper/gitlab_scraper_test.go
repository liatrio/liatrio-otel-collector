// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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

	scraper := newGitLabScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	assert.NotNil(t, scraper)
}

/*
 * Mocks
 */
type mockClient struct {
	BranchNames         []string
	openMergeRequests   []getMergeRequestsProjectMergeRequestsMergeRequestConnection
	mergedMergeRequests []getMergeRequestsProjectMergeRequestsMergeRequestConnection
	RootRef             string
	err                 bool
	mergedErr           bool
	openErr             bool
	errString           string
	curPage             int
}

func (m *mockClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	switch op := req.OpName; op {

	case "getBranchNames":
		if m.err {
			return errors.New(m.errString)
		}

		response := resp.Data.(*getBranchNamesResponse)
		response.Project.Repository.BranchNames = m.BranchNames
		response.Project.Repository.RootRef = m.RootRef

	case "getMergeRequests":
		response := resp.Data.(*getMergeRequestsResponse)

		if req.Variables.(*__getMergeRequestsInput).State == "opened" {
			if m.openErr {
				return errors.New(m.errString)
			}

			if len(m.openMergeRequests) == 0 {
				return nil
			}

			response.Project.MergeRequests = m.openMergeRequests[m.curPage]
			if m.openMergeRequests[m.curPage].PageInfo.HasNextPage == false {
				m.curPage = 0
			} else {
				m.curPage++
			}

		} else if req.Variables.(*__getMergeRequestsInput).State == "merged" {
			if m.mergedErr {
				return errors.New(m.errString)
			}

			if len(m.mergedMergeRequests) == 0 {
				return nil
			}

			response.Project.MergeRequests = m.mergedMergeRequests[m.curPage]
			if m.mergedMergeRequests[m.curPage].PageInfo.HasNextPage == false {
				return nil
			} else {
				m.curPage++
			}
		}
	}

	return nil
}

/*
 * Testing for getMergeRequests
 */
func TestGetMergeRequests(t *testing.T) {
	testCases := []struct {
		desc                      string
		client                    graphql.Client
		expectedErr               error
		expectedMergeRequestCount int
		state                     string
	}{
		{
			desc:                      "empty mergeRequestData",
			client:                    &mockClient{},
			expectedErr:               nil,
			expectedMergeRequestCount: 0,
		},
		{
			desc:                      "produce error for open merge requests",
			client:                    &mockClient{openErr: true, errString: "An error has occurred"},
			expectedErr:               errors.New("An error has occurred"),
			expectedMergeRequestCount: 0,
			state:                     "opened",
		},
		{
			desc:                      "produce error for merged merge requests",
			client:                    &mockClient{mergedErr: true, errString: "An error has occurred"},
			expectedErr:               errors.New("An error has occurred"),
			expectedMergeRequestCount: 0,
			state:                     "merged",
		},
		{
			desc: "valid mergeRequestData",
			client: &mockClient{
				mergedMergeRequests: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: false,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
				},
			},
			state:                     "merged",
			expectedErr:               nil,
			expectedMergeRequestCount: 3,
		},
		{
			desc: "valid mergeRequestData, multiple pages",
			client: &mockClient{
				mergedMergeRequests: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: true,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: true,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: false,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
				},
			},
			expectedErr:               nil,
			expectedMergeRequestCount: 9,
			state:                     "merged",
		},
	}
	for _, testCases := range testCases {
		t.Run(testCases.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))

			mergeRequestData, err := gls.getMergeRequests(context.Background(), testCases.client, "projectPath", MergeRequestState(testCases.state))

			assert.Equal(t, testCases.expectedMergeRequestCount, len(mergeRequestData))
			assert.Equal(t, testCases.expectedErr, err)
		})
	}
}

func TestGetCombinedMergeRequests(t *testing.T) {
	testCases := []struct {
		desc                      string
		client                    graphql.Client
		expectedErr               error
		expectedMergeRequestCount int
	}{
		{
			desc:                      "empty mergeRequestData",
			client:                    &mockClient{},
			expectedErr:               nil,
			expectedMergeRequestCount: 0,
		},
		{
			desc:                      "produce error for open merge requests",
			client:                    &mockClient{openErr: true, errString: "An error has occurred"},
			expectedErr:               errors.New("An error has occurred"),
			expectedMergeRequestCount: 0,
		},
		{
			desc:                      "produce error for merged merge requests",
			client:                    &mockClient{mergedErr: true, errString: "An error has occurred"},
			expectedErr:               errors.New("An error has occurred"),
			expectedMergeRequestCount: 0,
		},
		{
			desc: "valid mergeRequestData",
			client: &mockClient{
				mergedMergeRequests: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: false,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
				},
			},
			expectedErr:               nil,
			expectedMergeRequestCount: 3,
		},
		{
			desc: "valid open mergeRequestData, valid merged mergeRequestData with multiple pages",
			client: &mockClient{
				mergedMergeRequests: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: true,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: true,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: false,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
				},
				openMergeRequests: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
					{
						PageInfo: getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo{
							HasNextPage: false,
						},
						Nodes: []MergeRequestNode{
							{
								SourceBranch: "main",
							},
							{
								SourceBranch: "dev",
							},
							{
								SourceBranch: "feature",
							},
						},
					},
				},
			},
			expectedErr:               nil,
			expectedMergeRequestCount: 12,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))

			mergeRequestData, err := gls.getCombinedMergeRequests(context.Background(), tc.client, "projectPath")

			assert.Equal(t, tc.expectedMergeRequestCount, len(mergeRequestData))
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
