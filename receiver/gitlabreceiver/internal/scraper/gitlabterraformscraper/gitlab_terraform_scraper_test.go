// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabterraformscraper

import (
	"context"
	"encoding/json"
	"fmt"
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
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewGitLabTerraformScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitLabTerraformScraper(context.Background(), receivertest.NewNopSettings(metadata.Type), defaultConfig.(*Config))

	assert.NotNil(t, s)
}

func scrapeTestServer(packages []*gitlab.GroupPackage, blobs []*gitlab.Blob, projects map[int]*gitlab.Project) *http.ServeMux {
	var mux http.ServeMux

	mux.HandleFunc("/api/v4/groups/testgroup/packages", func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(packages)
		if err != nil {
			fmt.Printf("error marshalling response: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write(data); err != nil {
			fmt.Printf("error writing response: %v", err)
		}
	})

	mux.HandleFunc("/api/v4/groups/testgroup/-/search", func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(blobs)
		if err != nil {
			fmt.Printf("error marshalling response: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write(data); err != nil {
			fmt.Printf("error writing response: %v", err)
		}
	})

	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		var projectID int
		_, err := fmt.Sscanf(r.URL.Path, "/api/v4/projects/%d", &projectID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if proj, ok := projects[projectID]; ok {
			data, err := json.Marshal(proj)
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			if _, err = w.Write(data); err != nil {
				fmt.Printf("error writing response: %v", err)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	return &mux
}

func TestScrape(t *testing.T) {
	testCases := []struct {
		desc     string
		server   *http.ServeMux
		testFile string
	}{
		{
			desc: "Happy Path",
			server: scrapeTestServer(
				[]*gitlab.GroupPackage{
					{
						Package: gitlab.Package{
							ID:          1,
							Name:        "my-vpc/aws",
							Version:     "1.0.0",
							PackageType: "terraform_module",
						},
						ProjectID:   10,
						ProjectPath: "testgroup/infra",
					},
				},
				[]*gitlab.Blob{
					{
						Filename:  "main.tf",
						Data:      `source = "gitlab.com/testgroup/my-vpc/aws"`,
						ProjectID: 100,
						Path:      "infra/main.tf",
					},
					{
						Filename:  "vpc.tf",
						Data:      `source = "gitlab.com/testgroup/my-vpc/aws"`,
						ProjectID: 200,
						Path:      "network/vpc.tf",
					},
				},
				map[int]*gitlab.Project{
					100: {
						ID:                100,
						PathWithNamespace: "testgroup/consumer-app",
						WebURL:            "https://gitlab.com/testgroup/consumer-app",
					},
					200: {
						ID:                200,
						PathWithNamespace: "testgroup/another-app",
						WebURL:            "https://gitlab.com/testgroup/another-app",
					},
				},
			),
			testFile: "expected_happy_path.yaml",
		},
		{
			desc: "No Modules",
			server: scrapeTestServer(
				[]*gitlab.GroupPackage{},
				[]*gitlab.Blob{},
				map[int]*gitlab.Project{},
			),
			testFile: "expected_no_modules.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.server)
			defer server.Close()

			cfg := &Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()}

			gts := newGitLabTerraformScraper(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg)
			gts.cfg.GitLabOrg = "testgroup"
			gts.cfg.ClientConfig.Endpoint = server.URL

			err := gts.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			actualMetrics, err := gts.scrape(context.Background())
			require.NoError(t, err)

			expectedFile := filepath.Join("testdata", "scraper", tc.testFile)

			// Uncomment to regenerate golden files:
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
