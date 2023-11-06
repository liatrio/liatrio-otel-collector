// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"github.com/xanzy/go-gitlab"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

/*
 * Testing the start fucntion
 */
func Test_gitlabScraper_start(t *testing.T) {
	type args struct {
		in0  context.Context
		host component.Host
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		config  *Config
	}{
		{
			name:    "Happy Nil",
			args:    args{context.Background(), componenttest.NewNopHost()},
			wantErr: false,
			config:  &Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gls := newGitLabScraper(context.Background(), receivertest.NewNopCreateSettings(), tt.config)
			if err := gls.start(tt.args.in0, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("gitlabScraper.start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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

// This is from https://github.com/google/go-github/blob/master/github/repos_collaborators_test.go, creates mock http server
// The github client this library gives us doesn't use interfaces so the current mocking setup doesn't work with it
// and had to find a different way to create tests
func setupMockHttpServer() (client *gitlab.Client, mux *http.ServeMux, serverURL string, teardown func()) {
	// mux is the HTTP request multiplexer used with the test server.
	baseURLPath := "/api/v4"
	mux = http.NewServeMux()

	// We want to ensure that tests catch mistakes where the endpoint URL is
	// specified as absolute rather than relative. It only makes a difference
	// when there's a non-empty base URL path. So, use that. See issue #752.
	apiHandler := http.NewServeMux()
	apiHandler.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		fmt.Fprintln(os.Stderr, "\tSee https://github.com/google/go-github/issues/752 for information.")
		http.Error(w, "Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the GitHub client being tested and is
	// configured to use test server.
	url, err := url.Parse(server.URL + baseURLPath + "/")
	if err != nil {
		return nil, nil, "", nil
	}
	client, err = gitlab.NewClient("", gitlab.WithBaseURL(url.String()))
	if err != nil {
		return nil, nil, "", nil
	}

	return client, mux, server.URL, server.Close
}

func TestGetContributorCount(t *testing.T) {
	client, mux, _, teardown := setupMockHttpServer()

	defer teardown()
	mux.HandleFunc("/projects/project/repository/contributors", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"name":"john"}, {"name":"doe"}]`)
	})

	testCases := []struct {
		desc          string
		projectPath   string
		resp          string
		expectedErr   error
		expectedCount int
	}{
		{
			desc:          "valid",
			resp:          `[{"name":"john"}, {"name":"doe"}]`,
			projectPath:   "project",
			expectedErr:   nil,
			expectedCount: 2,
		},
		{
			desc:        "error",
			projectPath: "junk",
			expectedErr: errors.New("GET " + client.BaseURL().String() +
				"projects/junk/repository/contributors: 404 failed to parse unknown error format: 404 page not found\n"),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))

			contribs, err := gls.getContributorCount(client, tc.projectPath)

			assert.Equal(t, tc.expectedCount, contribs)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
