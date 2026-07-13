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
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
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
							ID:                1,
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
				// getCombinedMergeRequests issues two GraphQL calls: the first for
				// opened MRs, the second for merged MRs. MockServer serves mrs[0] to
				// the opened call and mrs[1] to the merged call, so each state returns
				// only state-appropriate MRs with distinct Iids (mirroring production,
				// where an MR is in exactly one state).
				//
				// Both MRs deliberately share the source branch "feature-a" (a reused
				// branch name) with different diff stats. This exercises the
				// vcs.ref.lines_delta path, whose attribute set keys datapoints by
				// branch rather than by change: two MRs on one branch collide on
				// identity+timestamp and are merged by the metrics builder.
				mrResponse: mrResponse{
					mrs: []getMergeRequestsProjectMergeRequestsMergeRequestConnection{
						{
							Nodes: []MergeRequestNode{
								{
									Title:        "mr1",
									Iid:          "1",
									SourceBranch: "feature-a",
									TargetBranch: "main",
									CreatedAt:    time.Now().AddDate(0, 0, -1),
									DiffStatsSummary: MergeRequestNodeDiffStatsSummary{
										Additions: 10,
										Deletions: 5,
									},
								},
							},
						},
						{
							Nodes: []MergeRequestNode{
								{
									Title:        "mr2",
									Iid:          "2",
									SourceBranch: "feature-a",
									TargetBranch: "main",
									CreatedAt:    time.Now().AddDate(0, 0, -2),
									MergedAt:     time.Now().AddDate(0, 0, -1),
									DiffStatsSummary: MergeRequestNodeDiffStatsSummary{
										Additions: 20,
										Deletions: 8,
									},
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

			cfg := &Config{MetricsBuilderConfig: metadata.NewDefaultMetricsBuilderConfig()}

			gls := newGitLabScraper(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg)
			gls.cfg.GitLabOrg = "project"
			gls.cfg.Endpoint = server.URL

			err := gls.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			actualMetrics, err := gls.scrape(context.Background())
			if tc.expectedErr != nil {
				require.Error(t, err)
				// Compare error messages directly since backoff.Retry may unwrap PermanentError
				var expectedMsg, actualMsg string
				if permErr, ok := tc.expectedErr.(*backoff.PermanentError); ok && permErr.Err != nil {
					expectedMsg = permErr.Err.Error()
				} else {
					expectedMsg = tc.expectedErr.Error()
				}

				if permErr, ok := err.(*backoff.PermanentError); ok && permErr.Err != nil {
					actualMsg = permErr.Err.Error()
				} else {
					actualMsg = err.Error()
				}

				require.Equal(t, expectedMsg, actualMsg)
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
