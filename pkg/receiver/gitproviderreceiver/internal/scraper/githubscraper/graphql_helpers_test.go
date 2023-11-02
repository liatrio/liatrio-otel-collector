package githubscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type mockClient struct {
	openPrCount   int
	mergedPrCount int
	err           bool
	err2          bool
	errString     string
	prs           getPullRequestDataRepositoryPullRequestsPullRequestConnection
	branchData    []getBranchDataRepositoryRefsRefConnection
	commitData    CommitNodeTargetCommit
	curPage       int
}

func (m *mockClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	switch op := req.OpName; op {
	case "getPullRequestCount":
		//for forcing arbitrary errors

		r := resp.Data.(*getPullRequestCountResponse)
		if len(req.Variables.(*__getPullRequestCountInput).States) == 0 {
			return errors.New("no pull request state provided")
		} else if req.Variables.(*__getPullRequestCountInput).States[0] == "OPEN" {
			if m.err && m.openPrCount != 0 {
				return errors.New(m.errString)
			}
			r.Repository.PullRequests.TotalCount = m.openPrCount
		} else if req.Variables.(*__getPullRequestCountInput).States[0] == "MERGED" {
			if m.err && m.mergedPrCount != 0 {
				return errors.New(m.errString)
			}
			r.Repository.PullRequests.TotalCount = m.mergedPrCount
		} else {
			return errors.New("invalid pull request state")
		}
	case "getPullRequestData":
		//for forcing arbitrary errors
		if m.err2 {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getPullRequestDataResponse)
		r.Repository.PullRequests = m.prs
	case "getBranchData":
		if m.err2 {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getBranchDataResponse)
		r.Repository.Refs = m.branchData[m.curPage]
		m.curPage++

	case "getCommitData":
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getCommitDataResponse)
		commitNodes := []CommitNode{
			{Target: &m.commitData},
		}
		r.Repository.Refs.Nodes = commitNodes

		// case "getRepoDataBySearch":
	}
	return nil
}

func TestGetPrCount(t *testing.T) {
	testCases := []struct {
		desc          string
		client        graphql.Client
		expectedErr   error
		expectedCount int
		state         []PullRequestState
	}{
		{
			desc:          "valid open pr count",
			client:        &mockClient{openPrCount: 3},
			expectedErr:   nil,
			expectedCount: 3,
			state:         []PullRequestState{PullRequestStateOpen},
		},
		{
			desc:          "valid merged pr count",
			client:        &mockClient{mergedPrCount: 20},
			expectedErr:   nil,
			expectedCount: 20,
			state:         []PullRequestState{PullRequestStateMerged},
		},
		{
			desc:          "no state",
			client:        &mockClient{},
			expectedErr:   errors.New("no pull request state provided"),
			expectedCount: 0,
			state:         []PullRequestState{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))

			count, err := ghs.getPrCount(context.Background(), tc.client, "repo", "owner", tc.state)

			assert.Equal(t, tc.expectedCount, count)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestGetNumPages100(t *testing.T) {
	p := float64(100)
	n := float64(375)

	expected := 4

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestGetNumPages10(t *testing.T) {
	p := float64(10)
	n := float64(375)

	expected := 38

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestGetNumPages1(t *testing.T) {
	p := float64(10)
	n := float64(1)

	expected := 1

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestAddInt(t *testing.T) {
	a := 100
	b := 100

	expected := 200

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddZero(t *testing.T) {
	a := 0
	b := 1

	expected := 1

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddFloat(t *testing.T) {
	a := 10.5
	b := 10.5

	expected := 21.0

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddNegativeInt(t *testing.T) {
	a := 1
	b := -1

	expected := 0

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddNegativeFloat(t *testing.T) {
	a := 1.5
	b := -10.0

	expected := -8.5

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestSubInt(t *testing.T) {
	a := 100
	b := 10

	expected := 90

	num := sub(a, b)

	assert.Equal(t, expected, num)
}

func TestSubFloat(t *testing.T) {
	a := 10.5
	b := 10.5

	expected := 0.0

	num := sub(a, b)

	assert.Equal(t, expected, num)
}

func TestSubNegativeInt(t *testing.T) {
	a := 1
	b := -1

	expected := 2

	num := sub(a, b)

	assert.Equal(t, expected, num)
}

func TestSubNegativeFloat(t *testing.T) {
	a := 1.5
	b := -10.0

	expected := 11.5

	num := sub(a, b)

	assert.Equal(t, expected, num)
}

func TestGenDefaultSearchQueryOrg(t *testing.T) {
	st := "org"
	org := "empire"

	expected := "org:empire archived:false"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}

func TestGenDefaultSearchQueryUser(t *testing.T) {
	st := "user"
	org := "vader"

	expected := "user:vader archived:false"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}

func TestCheckOwnerTypeValid(t *testing.T) {
	validOptions := []string{"org", "user"}

	for _, option := range validOptions {
		valid, err := checkOwnerTypeValid(option)

		assert.True(t, valid)
		assert.Nil(t, err)
	}
}

func TestCheckOwnerTypeValidRandom(t *testing.T) {
	invalidOptions := []string{"sorg", "suser", "users", "orgs", "invalid", "text"}

	for _, option := range invalidOptions {
		valid, err := checkOwnerTypeValid(option)

		assert.False(t, valid)
		assert.NotNil(t, err)
	}
}
