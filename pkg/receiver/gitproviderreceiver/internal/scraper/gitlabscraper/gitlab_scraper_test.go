// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gitlabscraper

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/Khan/genqlient/graphql"
)

/*
 *Testing for newGitLabScraper
 */
func TestNewGitLabScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitLabScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

/*
 *Testing for getBranches
 */
type mockClient struct {
	BranchNames []string
	RootRef     string
	err         bool
	errString   string
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
	}
	return nil
}

func TestGetBranches(t *testing.T) {
	testCases := []struct {
		desc             string
		client           graphql.Client
		expectedErr      error
		expectedBranches []string
	}{
		{
			desc:             "valid client",
			client:           &mockClient{BranchNames: []string{"string1", "string2"}, RootRef: "rootref"},
			expectedErr:      nil,
			expectedBranches: []string{"string1", "string2"},
		},
		{
			desc:             "produce error in client",
			client:           &mockClient{BranchNames: []string{"string1", "string2"}, RootRef: "rootref", err: true, errString: "An error has occured"},
			expectedErr:      errors.New("An error has occured"),
			expectedBranches: []string{"string1", "string2"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))

			var wg sync.WaitGroup
			ch := make(chan projectData, 1)

			wg.Add(1)
			gls.getBranches(context.Background(), tc.client, "silly-string", ch, &wg)
			wg.Wait()
			close(ch)

			if tc.expectedErr != nil {
				assert.Equal(t, len(ch), 0)
			} else {
				assert.Equal(t, len(ch), 1)
				for project := range ch {
					assert.Equal(t, tc.expectedBranches, project.Branches)
				}
			}

		})
	}
}
