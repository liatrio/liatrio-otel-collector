// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

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
		desc               string
		client             graphql.Client
		repos              []SearchNodeRepository
		expectedErr        error
		expectedChannelLen int
		expectedPrCount    int
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
			repos: []SearchNodeRepository{
				{
					Name: "repo1",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo2",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo3",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
			},
			expectedErr:        nil,
			expectedChannelLen: 3,  //one per repo
			expectedPrCount:    18, // 3 PRs per page, 2 pages, 3 repos
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
			repos: []SearchNodeRepository{
				{
					Name: "repo1",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo2",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
			},
			expectedErr:        nil,
			expectedChannelLen: 2, //one per repo
			expectedPrCount:    6, // 3 PRs per page, 1 page, 2 repos
		},
		{
			desc: "no next page",
			client: &mockClient{
				prs: getPullRequestDataRepositoryPullRequestsPullRequestConnection{
					PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
						HasNextPage: false,
					},
					Nodes: []PullRequestNode{},
				},
				openPrCount:   0,
				mergedPrCount: 0,
			},
			repos: []SearchNodeRepository{
				{
					Name: "repo1",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo2",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
			},
			expectedErr:        nil,
			expectedChannelLen: 0, // adding empty prs
			expectedPrCount:    0, // no prs
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
			prChan := make(chan []PullRequestNode, 20)
			now := pcommon.NewTimestampFromTime(time.Now())
			var wg sync.WaitGroup
			wg.Add(1)
			getPullRequests(ghs, context.Background(), tc.client, tc.repos, now, prChan, &wg)
			close(prChan)
			wg.Wait()
			assert.Equal(t, tc.expectedChannelLen, len(prChan))
			var totalPrs int
			for prs := range prChan {
				totalPrs += len(prs)
			}
			assert.Equal(t, tc.expectedPrCount, totalPrs)
		})
	}
}
func TestGetBranches(t *testing.T) {
	testCases := []struct {
		desc                string
		client              graphql.Client
		repos               []SearchNodeRepository
		expectedErr         error
		expectedChannelLen  int
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
			repos: []SearchNodeRepository{
				{
					Name: "repo1",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo2",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo3",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 9, //3 repos, 1 page, 3 branches per page
			expectedChannelLen:  3, //one per repo
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
			repos: []SearchNodeRepository{
				{
					Name: "repo1",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo2",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo3",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 27, //3 repos, 3 pages, 3 branches per page
			expectedChannelLen:  3,  //only one page

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
			repos: []SearchNodeRepository{
				{
					Name: "repo1",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo2",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
				{
					Name: "repo3",
					DefaultBranchRef: SearchNodeDefaultBranchRef{
						Name: "main",
					},
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 9, //3 repos, 1 pages, 3 branches per page
			expectedChannelLen:  3, //only one page

		},
		{
			desc:                "error",
			client:              &mockClient{err: true, errString: "this is an error"},
			expectedErr:         errors.New("this is an error"),
			expectedBranchCount: 0,
			expectedChannelLen:  0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			branchChan := make(chan []BranchNode, 20)
			now := pcommon.NewTimestampFromTime(time.Now())
			var wg sync.WaitGroup
			wg.Add(1)
			ghs.getBranches(context.Background(), tc.client, tc.repos, now, branchChan, &wg)
			close(branchChan)
			wg.Wait()
			assert.Equal(t, tc.expectedChannelLen, len(branchChan))
			var totalBranches int
			for branches := range branchChan {
				totalBranches += len(branches)
			}
			assert.Equal(t, tc.expectedBranchCount, totalBranches)
		})
	}
}
