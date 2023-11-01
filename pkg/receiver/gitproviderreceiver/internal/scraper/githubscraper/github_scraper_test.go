// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"errors"
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
			client:              &mockClient{err2: true, errString: "this is an error"},
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
			client:          &mockClient{err2: true, errString: "this is an error"},
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
		desc            string
		client          graphql.Client
		repo            SearchNodeRepository
		expectedErr     error
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
				openPrCount:   110,
				mergedPrCount: 50,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:     nil,
			expectedPrCount: 6, // 3 PRs per page, 2 pages
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
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:     nil,
			expectedPrCount: 3, // 3 PRs per page, 1 page
		},
		{
			desc: "error in getNumPrPages",
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
				openPrCount: 110,
				err:         true,
				errString:   "this is an error",
			},
			expectedErr: errors.New("this is an error"),
		},
		{
			desc: "error in getPrData",
			client: &mockClient{
				err2:        true,
				errString:   "this is an error",
				openPrCount: 110,
			},
			expectedErr: errors.New("this is an error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			now := pcommon.NewTimestampFromTime(time.Now())
			prs, err := getPullRequests(ghs, context.Background(), tc.client, tc.repo, now)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
			assert.Equal(t, tc.expectedPrCount, len(prs))
		})
	}
}
func TestGetBranches(t *testing.T) {
	testCases := []struct {
		desc                string
		client              graphql.Client
		repo                SearchNodeRepository
		expectedErr         error
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
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 3, //1 page, 3 branches per page
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
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 9, // 3 pages, 3 branches per page

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
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 3, //1 pages, 3 branches per page
		},
		{
			desc: "no branches",
			client: &mockClient{
				branchData: getBranchDataRepositoryRefsRefConnection{
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: true,
					},
					Nodes: []BranchNode{},
				},
				branchCount: 0,
			},
			repo: SearchNodeRepository{
				Name: "repo1",
				DefaultBranchRef: SearchNodeDefaultBranchRef{
					Name: "main",
				},
			},
			expectedErr:         nil,
			expectedBranchCount: 0,
		},
		{
			desc: "error in getNumBranchPages",
			client: &mockClient{
				err:       true,
				errString: "this is an error",
			},
			expectedErr:         errors.New("this is an error"),
			expectedBranchCount: 0,
		},
		{
			desc: "error in getBranchData",
			client: &mockClient{
				branchCount: 110,
				err2:        true,
				errString:   "this is an error",
			},
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
			now := pcommon.NewTimestampFromTime(time.Now())
			branches, err := ghs.getBranches(context.Background(), tc.client, tc.repo, now)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
			assert.Equal(t, tc.expectedBranchCount, len(branches))
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
