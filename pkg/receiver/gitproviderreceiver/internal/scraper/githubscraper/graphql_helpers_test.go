package githubscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	err        bool
	errString  string
	prs        []getPullRequestDataRepositoryPullRequestsPullRequestConnection
	branchData []getBranchDataRepositoryRefsRefConnection
	commitData CommitNodeTargetCommit
	curPage    int
}

func (m *mockClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	switch op := req.OpName; op {
	case "getPullRequestData":
		// for forcing arbitrary errors
		if m.err {
			return errors.New(m.errString)
		}
		response := resp.Data.(*getPullRequestDataResponse)
		response.Repository.PullRequests = m.prs[m.curPage]
		m.curPage++

	case "getBranchData":
		if m.err {
			return errors.New(m.errString)
		}
		response := resp.Data.(*getBranchDataResponse)
		response.Repository.Refs = m.branchData[m.curPage]
		m.curPage++

	case "getCommitData":
		if m.err {
			return errors.New(m.errString)
		}
		response := resp.Data.(*getCommitDataResponse)
		commitNodes := []CommitNode{
			{Target: &m.commitData},
		}
		response.Repository.Refs.Nodes = commitNodes

		// case "getRepoDataBySearch":
	}
	return nil
}

func TestGetNumPages100(t *testing.T) {
	perPage := float64(100)
	total := float64(375)

	expected := 4

	num := getNumPages(perPage, total)

	assert.Equal(t, expected, num)
}

func TestGetNumPages10(t *testing.T) {
	perPage := float64(10)
	total := float64(375)

	expected := 38

	num := getNumPages(perPage, total)

	assert.Equal(t, expected, num)
}

func TestGetNumPages1(t *testing.T) {
	perPage := float64(10)
	total := float64(1)

	expected := 1

	num := getNumPages(perPage, total)

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
	ownerType := "org"
	org := "empire"

	expected := "org:empire archived:false"

	actual := genDefaultSearchQuery(ownerType, org)

	assert.Equal(t, expected, actual)
}

func TestGenDefaultSearchQueryUser(t *testing.T) {
	ownerType := "user"
	org := "vader"

	expected := "user:vader archived:false"

	actual := genDefaultSearchQuery(ownerType, org)

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
