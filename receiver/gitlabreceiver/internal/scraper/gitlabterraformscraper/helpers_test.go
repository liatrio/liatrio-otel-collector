package gitlabterraformscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type responses struct {
	packagesResponse packagesResponse
	searchResponse   searchResponse
	projectResponses map[int]projectResponse
}

type packagesResponse struct {
	packages     []*gitlab.GroupPackage
	responseCode int
}

type searchResponse struct {
	blobs        []*gitlab.Blob
	responseCode int
}

type projectResponse struct {
	project      *gitlab.Project
	responseCode int
}

func MockServer(resp *responses) *http.ServeMux {
	var mux http.ServeMux

	mux.HandleFunc("/api/v4/groups/testgroup/packages", func(w http.ResponseWriter, r *http.Request) {
		pkgResp := &resp.packagesResponse
		w.WriteHeader(pkgResp.responseCode)
		if pkgResp.responseCode == http.StatusOK {
			data, err := json.Marshal(pkgResp.packages)
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			_, err = w.Write(data)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
		}
	})

	mux.HandleFunc("/api/v4/groups/testgroup/-/search", func(w http.ResponseWriter, r *http.Request) {
		searchResp := &resp.searchResponse
		w.WriteHeader(searchResp.responseCode)
		if searchResp.responseCode == http.StatusOK {
			data, err := json.Marshal(searchResp.blobs)
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			_, err = w.Write(data)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
		}
	})

	// Handle project info lookups (e.g., /api/v4/projects/100)
	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		// Extract project ID from path
		var projectID int
		_, err := fmt.Sscanf(r.URL.Path, "/api/v4/projects/%d", &projectID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if resp.projectResponses != nil {
			if projResp, ok := resp.projectResponses[projectID]; ok {
				w.WriteHeader(projResp.responseCode)
				if projResp.responseCode == http.StatusOK {
					data, err := json.Marshal(projResp.project)
					if err != nil {
						fmt.Printf("error marshalling response: %v", err)
					}
					if _, err = w.Write(data); err != nil {
						fmt.Printf("error writing response: %v", err)
					}
				}
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})

	return &mux
}

func TestGetModules(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		expectedErr   bool
		expectedCount int
	}{
		{
			desc: "SingleModule",
			server: MockServer(&responses{
				packagesResponse: packagesResponse{
					packages: []*gitlab.GroupPackage{
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
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1,
		},
		{
			desc: "MultipleModules",
			server: MockServer(&responses{
				packagesResponse: packagesResponse{
					packages: []*gitlab.GroupPackage{
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
						{
							Package: gitlab.Package{
								ID:          2,
								Name:        "my-iam/aws",
								Version:     "2.0.0",
								PackageType: "terraform_module",
							},
							ProjectID:   11,
							ProjectPath: "testgroup/iam",
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 2,
		},
		{
			desc: "DuplicateModuleVersions",
			server: MockServer(&responses{
				packagesResponse: packagesResponse{
					packages: []*gitlab.GroupPackage{
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
						{
							Package: gitlab.Package{
								ID:          2,
								Name:        "my-vpc/aws",
								Version:     "2.0.0",
								PackageType: "terraform_module",
							},
							ProjectID:   10,
							ProjectPath: "testgroup/infra",
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1, // Deduplicated by name+system
		},
		{
			desc: "EmptyResponse",
			server: MockServer(&responses{
				packagesResponse: packagesResponse{
					packages:     []*gitlab.GroupPackage{},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			gts := newGitLabTerraformScraper(context.Background(), settings, defaultConfig.(*Config))

			server := httptest.NewServer(tc.server)
			defer server.Close()

			gts.cfg.GitLabOrg = "testgroup"
			client, err := gitlab.NewClient("", gitlab.WithBaseURL(server.URL))
			require.NoError(t, err)

			modules, err := gts.getModules(context.Background(), client)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedCount, len(modules))
		})
	}
}

func TestSearchModuleConsumers(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		module        terraformModule
		expectedErr   bool
		expectedCount int
	}{
		{
			desc: "SingleConsumer",
			server: MockServer(&responses{
				searchResponse: searchResponse{
					blobs: []*gitlab.Blob{
						{
							Filename:  "main.tf",
							Data:      `source = "gitlab.com/testgroup/my-vpc/aws"`,
							ProjectID: 100,
							Path:      "infra/main.tf",
						},
					},
					responseCode: http.StatusOK,
				},
				projectResponses: map[int]projectResponse{
					100: {
						project: &gitlab.Project{
							ID:                100,
							PathWithNamespace: "testgroup/consumer-app",
							WebURL:            "https://gitlab.com/testgroup/consumer-app",
						},
						responseCode: http.StatusOK,
					},
				},
			}),
			module:        terraformModule{Name: "my-vpc", System: "aws"},
			expectedCount: 1,
		},
		{
			desc: "MultipleConsumers",
			server: MockServer(&responses{
				searchResponse: searchResponse{
					blobs: []*gitlab.Blob{
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
					responseCode: http.StatusOK,
				},
				projectResponses: map[int]projectResponse{
					100: {
						project: &gitlab.Project{
							ID:                100,
							PathWithNamespace: "testgroup/consumer-app",
							WebURL:            "https://gitlab.com/testgroup/consumer-app",
						},
						responseCode: http.StatusOK,
					},
					200: {
						project: &gitlab.Project{
							ID:                200,
							PathWithNamespace: "testgroup/another-app",
							WebURL:            "https://gitlab.com/testgroup/another-app",
						},
						responseCode: http.StatusOK,
					},
				},
			}),
			module:        terraformModule{Name: "my-vpc", System: "aws"},
			expectedCount: 2,
		},
		{
			desc: "DuplicateProjectDeduplication",
			server: MockServer(&responses{
				searchResponse: searchResponse{
					blobs: []*gitlab.Blob{
						{
							Filename:  "main.tf",
							Data:      `source = "gitlab.com/testgroup/my-vpc/aws"`,
							ProjectID: 100,
							Path:      "infra/main.tf",
						},
						{
							Filename:  "modules.tf",
							Data:      `source = "gitlab.com/testgroup/my-vpc/aws"`,
							ProjectID: 100,
							Path:      "infra/modules.tf",
						},
					},
					responseCode: http.StatusOK,
				},
				projectResponses: map[int]projectResponse{
					100: {
						project: &gitlab.Project{
							ID:                100,
							PathWithNamespace: "testgroup/consumer-app",
							WebURL:            "https://gitlab.com/testgroup/consumer-app",
						},
						responseCode: http.StatusOK,
					},
				},
			}),
			module:        terraformModule{Name: "my-vpc", System: "aws"},
			expectedCount: 1, // Same project, two files — should deduplicate
		},
		{
			desc: "ZeroConsumers",
			server: MockServer(&responses{
				searchResponse: searchResponse{
					blobs:        []*gitlab.Blob{},
					responseCode: http.StatusOK,
				},
			}),
			module:        terraformModule{Name: "my-vpc", System: "aws"},
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			gts := newGitLabTerraformScraper(context.Background(), settings, defaultConfig.(*Config))

			server := httptest.NewServer(tc.server)
			defer server.Close()

			gts.cfg.GitLabOrg = "testgroup"
			client, err := gitlab.NewClient("", gitlab.WithBaseURL(server.URL))
			require.NoError(t, err)

			consumers, err := gts.searchModuleConsumers(context.Background(), client, tc.module)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedCount, len(consumers))
		})
	}
}

func TestParseModuleName(t *testing.T) {
	testCases := []struct {
		input          string
		expectedName   string
		expectedSystem string
	}{
		{"my-vpc/aws", "my-vpc", "aws"},
		{"my-iam/gcp", "my-iam", "gcp"},
		{"simple-module", "simple-module", "generic"},
		{"nested/name/extra", "nested", "name/extra"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			name, system := parseModuleName(tc.input)
			assert.Equal(t, tc.expectedName, name)
			assert.Equal(t, tc.expectedSystem, system)
		})
	}
}
