// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v53/github"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewGitHubScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitHubScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

func TestScrape(t *testing.T) {
	testCases := []struct {
		desc   string
		server *http.ServeMux
	}{
		// {
		// 	desc: "TestNoRepos",
		// 	server: graphqlMockServer(&responses{
		// 		repos: []getRepoDataBySearchSearchSearchResultItemConnection{
		// 			{
		// 				RepositoryCount: 0,
		// 				Nodes:           []SearchNode{},
		// 			},
		// 		},
		// 		responseCode: http.StatusOK,
		// 	}),
		// },
		{
			desc: "TestHappyPath",
			server: graphqlMockServer(&responses{
				repoResponse: repoResponse{
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
				},
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
							},
						},
					},
					responseCode: http.StatusOK,
				},
				contribResponse: contribResponse{
					contribs: []*github.Contributor{
						{
							ID: github.Int64(1),
						},
					},
					responseCode: http.StatusOK,
				},
				checkLoginResponse: LoginResponse{
					checkLogin: checkLoginResponse{
						Organization: checkLoginOrganization{
							Login: "liatrio",
						},
					},
					responseCode: http.StatusOK,
				},
			}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.server)
			defer server.Close()

			cfg := &Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()}

			ghs := newGitHubScraper(context.Background(), receivertest.NewNopCreateSettings(), cfg)
			ghs.cfg.GitHubOrg = "liatrio"
			ghs.cfg.HTTPClientSettings.Endpoint = server.URL

			err := ghs.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			actualMetrics, err := ghs.scrape(context.Background())
			require.NoError(t, err)

			expectedFile := filepath.Join("testdata", "scraper", "expected_happy_path.yaml")

			//golden.WriteMetrics(t, expectedFile, actualMetrics) // This line is temporary! TODO remove this!!
			expectedMetrics, err := golden.ReadMetrics(expectedFile)
			require.NoError(t, err)

			//Timestamps are not accurate. Not sure why
			require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreMetricDataPointsOrder(), pmetrictest.IgnoreTimestamp(), pmetrictest.IgnoreStartTimestamp()))

		})
	}
}
