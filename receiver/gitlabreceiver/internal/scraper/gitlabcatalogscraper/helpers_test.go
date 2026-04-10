package gitlabcatalogscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/liatrio/liatrio-otel-collector/receiver/gitlabreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type responses struct {
	projectResponse         projectResponse
	componentUsagesResponse componentUsagesResponse
	catalogResourceResponse catalogResourceResponse
	ciConfigResponse        ciConfigResponse
}

type catalogResourceResponse struct {
	resources    map[string]getCatalogResourceCiCatalogResource
	responseCode int
}

type ciConfigResponse struct {
	configs map[string]string
}

type projectResponse struct {
	projects     []*gitlab.Project
	responseCode int
}

type componentUsagesResponse struct {
	usages       []getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnection
	page         int
	responseCode int
}

func MockServer(responses *responses) *http.ServeMux {
	var mux http.ServeMux

	// GraphQL endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var reqBody graphql.Request
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			return
		}
		switch {
		case reqBody.OpName == "getCatalogResource":
			catalogResp := &responses.catalogResourceResponse
			code := catalogResp.responseCode
			if code == 0 {
				code = http.StatusOK
			}
			w.WriteHeader(code)
			if code == http.StatusOK {
				vars, _ := reqBody.Variables.(map[string]interface{})
				fullPath, _ := vars["fullPath"].(string)
				resource, found := catalogResp.resources[fullPath]
				if !found {
					resource = getCatalogResourceCiCatalogResource{}
				}
				resp := getCatalogResourceResponse{
					CiCatalogResource: resource,
				}
				graphqlResponse := graphql.Response{Data: &resp}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
			}

		case reqBody.OpName == "getProjectComponentUsages":
			usageResp := &responses.componentUsagesResponse
			w.WriteHeader(usageResp.responseCode)
			if usageResp.responseCode == http.StatusOK {
				if usageResp.page >= len(usageResp.usages) {
					resp := getProjectComponentUsagesResponse{
						Project: getProjectComponentUsagesProject{
							ComponentUsages: getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnection{},
						},
					}
					graphqlResponse := graphql.Response{Data: &resp}
					if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
						return
					}
					return
				}
				resp := getProjectComponentUsagesResponse{
					Project: getProjectComponentUsagesProject{
						ComponentUsages: usageResp.usages[usageResp.page],
					},
				}
				graphqlResponse := graphql.Response{Data: &resp}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				usageResp.page++
			}
		}
	})

	// REST API: list group projects
	mux.HandleFunc("/api/v4/groups/project/projects", func(w http.ResponseWriter, r *http.Request) {
		projectResp := &responses.projectResponse
		if projectResp.responseCode == http.StatusOK {
			data, err := json.Marshal(projectResp.projects)
			if err != nil {
				fmt.Printf("error marshalling response: %v", err)
			}
			_, err = w.Write(data)
			if err != nil {
				fmt.Printf("error writing response: %v", err)
			}
		}
	})

	// REST API: get raw .gitlab-ci.yml
	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		ciResp := &responses.ciConfigResponse
		if ciResp.configs != nil {
			// Extract project path from URL
			for projectPath, config := range ciResp.configs {
				expectedPrefix := fmt.Sprintf("/api/v4/projects/%s/repository/files/", projectPath)
				if len(r.URL.Path) >= len(expectedPrefix) && r.URL.Path[:len(expectedPrefix)] == expectedPrefix {
					_, _ = w.Write([]byte(config))
					return
				}
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})

	return &mux
}

func TestGetProjects(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		expectedCount int
	}{
		{
			desc: "SingleProject",
			server: MockServer(&responses{
				projectResponse: projectResponse{
					projects: []*gitlab.Project{
						{
							Name:              "project1",
							PathWithNamespace: "group/project1",
							WebURL:            "https://gitlab.com/group/project1",
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1,
		},
		{
			desc: "MultipleProjects",
			server: MockServer(&responses{
				projectResponse: projectResponse{
					projects: []*gitlab.Project{
						{
							Name:              "project1",
							PathWithNamespace: "group/project1",
							WebURL:            "https://gitlab.com/group/project1",
						},
						{
							Name:              "project2",
							PathWithNamespace: "group/project2",
							WebURL:            "https://gitlab.com/group/project2",
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 2,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopSettings(metadata.Type)
			gcs := newGitLabCatalogScraper(context.Background(), settings, defaultConfig.(*Config))
			gcs.cfg.GitLabOrg = "project"
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client, err := gitlab.NewClient("", gitlab.WithBaseURL(server.URL))
			require.NoError(t, err)
			projects, err := gcs.getProjects(context.Background(), client)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(projects))
		})
	}
}

func TestGetProjectComponentUsages(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		expectedCount int
	}{
		{
			desc: "WithUsages",
			server: MockServer(&responses{
				componentUsagesResponse: componentUsagesResponse{
					usages: []getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnection{
						{
							Nodes: []ComponentUsageNode{
								{Name: "sast"},
								{Name: "secret-detection"},
							},
							PageInfo: getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 2,
		},
		{
			desc: "Empty",
			server: MockServer(&responses{
				componentUsagesResponse: componentUsagesResponse{
					usages: []getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnection{
						{
							Nodes: []ComponentUsageNode{},
							PageInfo: getProjectComponentUsagesProjectComponentUsagesCiComponentUsageConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
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
			gcs := newGitLabCatalogScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client := graphql.NewClient(server.URL, gcs.client)
			usages, err := gcs.getProjectComponentUsages(context.Background(), client, "group/project1")

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(usages))
		})
	}
}
