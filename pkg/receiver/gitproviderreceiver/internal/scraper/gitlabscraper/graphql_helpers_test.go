package gitlabscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/Khan/genqlient/graphql"
)

func TestGetBranchNames(t *testing.T) {
	rootRef := "rootref"
	branchNames := []string{"string1", "string2"}
	testCases := []struct {
		desc         string
		client       graphql.Client
		expectedErr  error
		expectedResp *getBranchNamesProjectRepository
	}{
		{
			desc:        "valid client",
			client:      &mockClient{BranchNames: branchNames, RootRef: rootRef},
			expectedErr: nil,
			expectedResp: &getBranchNamesProjectRepository{
				BranchNames: branchNames,
				RootRef:     rootRef,
			},
		},
		{
			desc:         "produce error in client",
			client:       &mockClient{BranchNames: branchNames, RootRef: rootRef, err: true, errString: "An error has occurred"},
			expectedErr:  errors.New("An error has occurred"),
			expectedResp: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			gls := newGitLabScraper(context.Background(), settings, defaultConfig.(*Config))

			branches, err := gls.getBranchNames(context.Background(), tc.client, "projectPath")
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedResp, branches)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.Equal(t, tc.expectedResp, branches)
				assert.Equal(t, err, nil)
			}
		})
	}
}
