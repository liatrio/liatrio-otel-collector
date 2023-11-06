// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		r := resp.Data.(*getBranchNamesResponse)
		r.Project.Repository.BranchNames = m.BranchNames
		r.Project.Repository.RootRef = m.RootRef

	case "getMergeRequests":
		r := resp.Data.(*getMergeRequestsResponse)

		if req.Variables.(*__getMergeRequestsInput).State == "opened" {
			if m.openErr {
				return errors.New(m.errString)
			}
			if len(m.openMergeRequests) == 0 {
				return nil
			}
			r.Project.MergeRequests = m.openMergeRequests[m.curPage]
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
			r.Project.MergeRequests = m.mergedMergeRequests[m.curPage]
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
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))

			mergeRequestData, err := gls.getMergeRequests(context.Background(), tc.client, "projectPath", MergeRequestState(tc.state))

			assert.Equal(t, tc.expectedMergeRequestCount, len(mergeRequestData))
			assert.Equal(t, tc.expectedErr, err)
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

// func TestScrape(t *testing.T) {

// 	type fields struct {
// 		client   *http.Client
// 		cfg      *Config
// 		settings component.TelemetrySettings
// 		logger   *zap.Logger
// 		mb       *metadata.MetricsBuilder
// 	}
// 	testCases := []struct {
// 		desc    string
// 		fields  fields
// 		ctx     context.Context
// 		want    pmetric.Metrics
// 		wantErr bool
// 	}{
// 		{
// 			desc:    "valid test",
// 			ctx:     context.Background(),
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range testCases {
// 		t.Run(tt.desc, func(t *testing.T) {
// 			gls := &gitlabScraper{
// 				client:   tt.fields.client,
// 				cfg:      tt.fields.cfg,
// 				settings: tt.fields.settings,
// 				logger:   tt.fields.logger,
// 				mb:       tt.fields.mb,
// 			}
// 			got, err := gls.scrape(tt.ctx)
// 			if (err != nil) != tt.wantErr {
// 				// t.Errorf("gitlabScraper.scrape() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				// t.Errorf("gitlabScraper.scrape() = %v, want %v", got, tt.want)
// 				return
// 			}
// 		})
// 	}
// }

func TestScrape(t *testing.T) {
	type testCase struct {
		name string
		// bootTimeFunc        func(context.Context) (uint64, error)
		// timesFunc           func(context.Context, bool) ([]cpu.TimesStat, error)
		metricsConfig       metadata.MetricsBuilderConfig
		expectedMetricCount int
		// expectedStartTime   pcommon.Timestamp
		expectedEndpoint  string
		initializationErr string
		expectedErr       string
	}

	// disabledMetric := metadata.DefaultMetricsBuilderConfig()
	// disabledMetric.Metrics.SystemCPUTime.Enabled = false

	testCases := []testCase{
		{
			name:                "Standard",
			metricsConfig:       metadata.DefaultMetricsBuilderConfig(),
			expectedMetricCount: 7,
		},
		{
			name:             "non-empty endpoint", // unknown assert
			metricsConfig:    metadata.DefaultMetricsBuilderConfig(),
			expectedEndpoint: "endpoint",
		},
		// {
		// 	name:                "Validate Start Time",
		// 	bootTimeFunc:        func(context.Context) (uint64, error) { return 100, nil },
		// 	metricsConfig:       metadata.DefaultMetricsBuilderConfig(),
		// 	expectedMetricCount: 1,
		// 	expectedStartTime:   100 * 1e9,
		// },
		// {
		// 	name:                "Boot Time Error",
		// 	bootTimeFunc:        func(context.Context) (uint64, error) { return 0, errors.New("err1") },
		// 	metricsConfig:       metadata.DefaultMetricsBuilderConfig(),
		// 	expectedMetricCount: 1,
		// 	initializationErr:   "err1",
		// },
		// {
		// 	name:                "Times Error",
		// 	timesFunc:           func(context.Context, bool) ([]cpu.TimesStat, error) { return nil, errors.New("err2") },
		// 	metricsConfig:       metadata.DefaultMetricsBuilderConfig(),
		// 	expectedMetricCount: 1,
		// 	expectedErr:         "err2",
		// },
		// {
		// 	name:                "SystemCPUTime metric is disabled ",
		// 	metricsConfig:       disabledMetric,
		// 	expectedMetricCount: 0,
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gls := newGitLabScraper(context.Background(), receivertest.NewNopCreateSettings(), &Config{MetricsBuilderConfig: tc.metricsConfig})
			gls.cfg.HTTPClientSettings.Endpoint = tc.expectedEndpoint
			gls.cfg.GitLabOrg = "liatrioinc"

			// if tc.bootTimeFunc != nil {
			// 	scraper.bootTime = tc.bootTimeFunc
			// }
			// if tc.timesFunc != nil {
			// 	scraper.times = tc.timesFunc
			// }

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

			// if tc.expectedMetricCount > 0 {
			// 	metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
			// 	assertCPUMetricValid(t, metrics.At(0), tc.expectedStartTime)

			// 	if runtime.GOOS == "linux" {
			// 		assertCPUMetricHasLinuxSpecificStateLabels(t, metrics.At(0))
			// 	}

			// 	internal.AssertSameTimeStampForAllMetrics(t, metrics)
			// }
		})
	}
}
