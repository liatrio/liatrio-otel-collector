// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/scrapererror"
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

func TestScrape(t *testing.T) {
	type testCase struct {
		name                string
		metricsConfig       metadata.MetricsBuilderConfig
		expectedMetricCount int
		expectedEndpoint    string
		initializationErr   string
		expectedErr         string
	}

	testCases := []testCase{
		{
			name:                "Standard",
			metricsConfig:       metadata.DefaultMetricsBuilderConfig(),
			expectedMetricCount: 7, //this will change
		},
		{
			name:                "non-empty endpoint", // unknown assert
			metricsConfig:       metadata.DefaultMetricsBuilderConfig(),
			expectedEndpoint:    "endpoint",
			expectedMetricCount: 0, //this will not change
		},
		{
			name:                "invalid client",
			metricsConfig:       metadata.DefaultMetricsBuilderConfig(),
			expectedMetricCount: 7, //this will change if new repo is added
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gls := newGitLabScraper(context.Background(), receivertest.NewNopCreateSettings(), &Config{MetricsBuilderConfig: tc.metricsConfig})
			gls.cfg.HTTPClientSettings.Endpoint = tc.expectedEndpoint

			//This line hit's alot of code
			gls.cfg.GitLabOrg = "liatrioinc"

			err := gls.start(context.Background(), componenttest.NewNopHost())
			if tc.initializationErr != "" {
				assert.EqualError(t, err, tc.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize gitlab scraper: %v", err)

			md, err := gls.scrape(context.Background())
			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)

				isPartial := scrapererror.IsPartialScrapeError(err)
				assert.True(t, isPartial)
				if isPartial {
					var scraperErr scrapererror.PartialScrapeError
					require.ErrorAs(t, err, &scraperErr)
					assert.Equal(t, 2, scraperErr.Failed)
				}

				return
			}
			// require.NoError(t, err, "Failed to scrape metrics: %v", err)

			assert.Equal(t, tc.expectedMetricCount, md.MetricCount())

		})
	}
}

// testing golden to get "expected.yam" which contains metric from a actual scrape
func TestScraper2(t *testing.T) {
	cfg := &Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()}
	require.NoError(t, component.ValidateConfig(cfg))

	gls := newGitLabScraper(context.Background(), receivertest.NewNopCreateSettings(), cfg)
	gls.cfg.GitLabOrg = "liatrioinc"

	err := gls.start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	actualMetrics, err := gls.scrape(context.Background())
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "scraper", "expected.yaml")

	// golden.WriteMetrics(t, expectedFile, actualMetrics) // This line is temporary! TODO remove this!!

	expectedMetrics, err := golden.ReadMetrics(expectedFile)
	require.NoError(t, err)

	//Timestamps are not accurate. Not sure why
	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics))
}

// func Test_gitlabScraper_processBranches(t *testing.T) {
// 	type args struct {
// 		client      *gitlab.Client
// 		branches    *getBranchNamesProjectRepository
// 		projectPath string
// 		now         pcommon.Timestamp
// 	}
// 	tests := []struct {
// 		name   string
// 		config *Config
// 		args   args

// 	}{
// 		{
// 			name: "happy test",
// 			config: &Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()},
// 			args: args{

// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			gls := newGitLabScraper(context.Background(), receivertest.NewNopCreateSettings(), tt.config)

// 			gls.processBranches(tt.args.client, tt.args.branches, tt.args.projectPath, tt.args.now)
// 		})
// 	}
// }
