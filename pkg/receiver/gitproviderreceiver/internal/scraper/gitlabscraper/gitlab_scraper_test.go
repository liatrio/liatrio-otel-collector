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

	s := newGitLabScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

/*
 * Mocks
 */
type mockClient struct {
	BranchNames   []string
	MergeRequests getMergeRequestsProjectMergeRequestsMergeRequestConnection
	RootRef       string
	err           bool
	errString     string
	maxPages      int
	curPage       int
}

func (m *mockClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	if m.err {
		return errors.New(m.errString)
	}

	switch op := req.OpName; op {

	case "getBranchNames":
		r := resp.Data.(*getBranchNamesResponse)
		r.Project.Repository.BranchNames = m.BranchNames
		r.Project.Repository.RootRef = m.RootRef

	case "getMergeRequests":
		m.curPage++

		if m.curPage == m.maxPages {
			m.MergeRequests.PageInfo.HasNextPage = false
		}

		r := resp.Data.(*getMergeRequestsResponse)
		r.Project.MergeRequests = m.MergeRequests
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
	}{
		{
			desc:                      "empty mergeRequestData",
			client:                    &mockClient{},
			expectedErr:               nil,
			expectedMergeRequestCount: 0,
		},
		{
			desc:                      "produce error in client",
			client:                    &mockClient{err: true, errString: "An error has occurred"},
			expectedErr:               errors.New("An error has occurred"),
			expectedMergeRequestCount: 0,
		},
		{
			desc: "valid mergeRequestData",
			client: &mockClient{
				maxPages: 1,
				MergeRequests: getMergeRequestsProjectMergeRequestsMergeRequestConnection{
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
			},
			expectedErr:               nil,
			expectedMergeRequestCount: 3,
		},
		{
			desc: "valid mergeRequestData",
			client: &mockClient{
				maxPages: 5,
				MergeRequests: getMergeRequestsProjectMergeRequestsMergeRequestConnection{
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
			},
			expectedErr:               nil,
			expectedMergeRequestCount: 15, // 5 pages * 3 merge requests
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			const state MergeRequestState = "merged"

			mergeRequestData, err := gls.getMergeRequests(context.Background(), tc.client, "projectPath", state)

			assert.Equal(t, tc.expectedMergeRequestCount, len(mergeRequestData))
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
