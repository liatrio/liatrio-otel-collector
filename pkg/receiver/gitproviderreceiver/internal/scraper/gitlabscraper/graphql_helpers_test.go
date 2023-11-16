package gitlabscraper

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/Khan/genqlient/graphql"
)

func TestGetBranchNames(t *testing.T) {
	testCases := []struct {
		desc          string
		server        *http.ServeMux
		expectedErr   error
		expectedCount int
	}{
		{
			desc: "TestSingleBranch",
			server: MockServer(&responses{
				branchResponse: branchResponse{
					branches: getBranchNamesProjectRepository{
						BranchNames: []string{"branch1"},
						RootRef:     "main",
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			desc: "TestMultipleBranches",
			server: MockServer(&responses{
				branchResponse: branchResponse{
					branches: getBranchNamesProjectRepository{
						BranchNames: []string{"branch1", "branch2", "branch3"},
						RootRef:     "main",
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCount: 3,
			expectedErr:   nil,
		},
		{
			desc: "Test404Error",
			server: MockServer(&responses{
				branchResponse: branchResponse{
					responseCode: http.StatusNotFound,
				},
			}),
			expectedCount: 0,
			expectedErr:   errors.New("returned error 404 Not Found: "),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client := graphql.NewClient(server.URL, gls.client)

			branches, err := gls.getBranchNames(context.Background(), client, "projectPath")

			if tc.expectedErr != nil {
				//nil is returned like this for some reason
				assert.Equal(t, (*getBranchNamesProjectRepository)(nil), branches)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, len(branches.BranchNames))

			}
		})
	}
}

func TestGetInitialCommit(t *testing.T) {
	testCases := []struct {
		desc           string
		resp           string
		expectedErr    error
		server         *http.ServeMux
		expectedCommit *gitlab.Commit
	}{
		{
			desc: "TestSingleCommit",
			server: MockServer(&responses{
				compareResponse: compareResponse{
					compare: &gitlab.Compare{
						Commits: []*gitlab.Commit{
							{
								Title: "commit1",
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCommit: &gitlab.Commit{
				Title: "commit1",
			},
		},
		{
			desc: "TestMultipleCommits",
			server: MockServer(&responses{
				compareResponse: compareResponse{
					compare: &gitlab.Compare{
						Commits: []*gitlab.Commit{
							{
								Title: "commit1",
							},
							{
								Title: "commit2",
							},
							{
								Title: "commit3",
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			expectedCommit: &gitlab.Commit{
				Title: "commit1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()
			client, err := gitlab.NewClient("", gitlab.WithBaseURL(server.URL))
			assert.NoError(t, err)
			commit, err := gls.getInitialCommit(client, "project", "defaultBranch", "branch")

			assert.Equal(t, tc.expectedCommit, commit)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
