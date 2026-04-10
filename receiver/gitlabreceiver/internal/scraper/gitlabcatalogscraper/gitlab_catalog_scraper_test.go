// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabcatalogscraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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

func TestNewGitLabCatalogScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitLabCatalogScraper(context.Background(), receiver.Settings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

func TestScrape(t *testing.T) {
	testCases := []struct {
		desc        string
		expectedErr bool
		server      *http.ServeMux
		testFile    string
	}{
		{
			desc: "Happy Path",
			server: MockServer(&responses{
				catalogResourceResponse: catalogResourceResponse{
					resources: map[string]getCatalogResourceCiCatalogResource{
						"components/secret-detection": {
							Name: "Secret Detection", FullPath: "components/secret-detection",
							StarCount: 50, Last30DayUsageCount: 8000,
						},
						"components/opentofu": {
							Name: "OpenTofu", FullPath: "components/opentofu",
							StarCount: 161, Last30DayUsageCount: 5445,
						},
					},
				},
				ciConfigResponse: ciConfigResponse{
					configs: map[string]string{
						"my-app": "include:\n  - component: gitlab.com/components/secret-detection/sast@2.3.0\n" +
							"  - component: gitlab.com/components/opentofu/fmt@4.5.0\n",
					},
				},
				projectResponse: projectResponse{
					projects: []*gitlab.Project{
						{
							Name:              "my-app",
							ID:                1,
							PathWithNamespace: "my-app",
							WebURL:            "https://gitlab.com/project/my-app",
						},
					},
					responseCode: http.StatusOK,
				},
				componentUsagesResponse: componentUsagesResponse{
					usages: []getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnection{
						{
							Nodes: []ComponentUsageNode{
								{Name: "sast"},
								{Name: "fmt"},
							},
							PageInfo: getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnectionPageInfo{
								HasNextPage: false,
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

			gcs := newGitLabCatalogScraper(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg)
			gcs.cfg.GitLabOrg = "project"
			gcs.cfg.ClientConfig.Endpoint = server.URL

			err := gcs.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			actualMetrics, err := gcs.scrape(context.Background())
			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			expectedFile := filepath.Join("testdata", "scraper", tc.testFile)

			// Uncomment the line below to regenerate golden files:
			// golden.WriteMetrics(t, expectedFile, actualMetrics)

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
