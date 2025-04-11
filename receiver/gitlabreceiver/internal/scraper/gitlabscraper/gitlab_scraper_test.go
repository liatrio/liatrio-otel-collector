// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

/*
 * Testing for newGitLabScraper
 */
func TestNewGitLabScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitLabScraper(context.Background(), receiver.Settings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

func TestScrape(t *testing.T) {
	testCases := []struct {
		desc          string
		expectedErr   error
		server        *http.ServeMux
		expectedCount int
		testFile      string
	}{
		{
			desc: "No Projects",
			server: MockServer(&responses{
				projectResponse: projectResponse{
					projects:     []*gitlab.Project{},
					responseCode: http.StatusOK,
				},
			}),
			testFile:    "expected_no_projects.yaml",
			expectedErr: backoff.Permanent(errors.New("no GitLab projects found for the given group/org: project")),
		},
		{
			desc: "Happy Path",
			server: MockServer(&responses{
				projectResponse: projectResponse{
					projects: []*gitlab.Project{
						{
							Name:              "project",
							PathWithNamespace: "project",
							CreatedAt:         gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
							LastActivityAt:    gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
						},
					},
					responseCode: http.StatusOK,
				},
				branchResponse: branchResponse{
					branches: getBranchNamesProjectRepository{
						BranchNames: []string{"branch1"},
					},
					responseCode: http.StatusOK,
				},
				mrResponse: mrResponse{
					mrs: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
						{
							Nodes: []MergeRequestNode{
								{
									Title:     "mr1",
									CreatedAt: time.Now().AddDate(0, 0, -1),
								},
								{
									Title:    "mr1",
									MergedAt: time.Now().AddDate(0, 0, -1),
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
				contribResponse: contribResponse{
					contribs: []*gitlab.Contributor{
						{
							Name:      "contrib1",
							Additions: 1,
							Deletions: 1,
						},
					},
					responseCode: http.StatusOK,
				},
				compareResponse: compareResponse{
					compare: &gitlab.Compare{
						Commits: []*gitlab.Commit{
							{
								Title:     "commit1",
								CreatedAt: gitlab.Ptr(time.Now().AddDate(0, 0, -1)),
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			testFile: "expected_happy_path.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.server)
			defer server.Close()

			cfg := &Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()}

			gls := newGitLabScraper(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg)
			gls.cfg.GitLabOrg = "project"
			gls.cfg.ClientConfig.Endpoint = server.URL

			err := gls.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			actualMetrics, err := gls.scrape(context.Background())
			if tc.expectedErr != nil {
				require.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
			}

			expectedFile := filepath.Join("testdata", "scraper", tc.testFile)

			// Due to the generative nature of the code we're using through genqlient. The tests happy path changes,
			// and needs to be rebuilt to satisfy the unit tests. When the metadata.yaml changes, and code is
			// introduced, or removed. We'll need to update the metrics by uncommenting the below and running
			// `make test` to generate it. Then we're safe to comment this out again and see happy tests.
			// golden.WriteMetrics(t, expectedFile, actualMetrics) // This line is temporary! TODO remove this!!

			expectedMetrics, err := golden.ReadMetrics(expectedFile)
			require.NoError(t, err)

			require.NoError(t, pmetrictest.CompareMetrics(
				expectedMetrics,
				actualMetrics,
				pmetrictest.IgnoreMetricDataPointsOrder(),
				pmetrictest.IgnoreTimestamp(),
				pmetrictest.IgnoreStartTimestamp(),
			))
		})
	}
}
