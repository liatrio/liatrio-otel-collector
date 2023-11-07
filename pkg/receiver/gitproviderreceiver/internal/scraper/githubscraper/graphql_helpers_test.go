package githubscraper

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type mockClient struct {
	err        bool
	errString  string
	prs        []getPullRequestDataRepositoryPullRequestsPullRequestConnection
	branchData []getBranchDataRepositoryRefsRefConnection
	repoData   []getRepoDataBySearchSearchSearchResultItemConnection
	commitData CommitNodeTargetCommit
	curPage    int
}

type responses struct {
	responseCode int
	checkLogin   checkLoginResponse
	prs          []getPullRequestDataRepositoryPullRequestsPullRequestConnection
	curPage      int
}

func (m *mockClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	switch op := req.OpName; op {
	case "getPullRequestData":
		//for forcing arbitrary errors
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getPullRequestDataResponse)
		r.Repository.PullRequests = m.prs[m.curPage]
		m.curPage++

	case "getBranchData":
		if m.err {
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

	case "getRepoDataBySearch":
		if m.err {
			return errors.New(m.errString)
		}
		r := resp.Data.(*getRepoDataBySearchResponse)
		r.Search = m.repoData[m.curPage]
		m.curPage++
	}
	return nil
}

func createServer(endpoint string, responses *responses) *http.ServeMux {
	var mux http.ServeMux
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		var reqBody graphql.Request
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			return
		}
		switch {
		case reqBody.OpName == "checkLogin":
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				login := responses.checkLogin
				graphqlResponse := graphql.Response{Data: &login}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
			}
		case reqBody.OpName == "getPullRequestData":
			w.WriteHeader(responses.responseCode)
			if responses.responseCode == http.StatusOK {
				prs := getPullRequestDataResponse{
					Repository: getPullRequestDataRepository{
						PullRequests: responses.prs[responses.curPage],
					},
				}
				graphqlResponse := graphql.Response{Data: &prs}
				if err := json.NewEncoder(w).Encode(graphqlResponse); err != nil {
					return
				}
				responses.curPage++
			}
		}
	})
	return &mux
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

func TestCheckOwnerExists(t *testing.T) {
	testCases := []struct {
		desc                string
		login               string
		expectedError       bool
		expectedOwnerType   string
		expectedOwnerExists bool
		server              *http.ServeMux
	}{
		{
			desc:  "check org owner exists",
			login: "liatrio",
			server: createServer("/", &responses{
				checkLogin: checkLoginResponse{
					Organization: checkLoginOrganization{
						Login: "liatrio",
					},
				},
				responseCode: http.StatusOK,
			}),
			expectedOwnerType:   "org",
			expectedOwnerExists: true,
		},
		{
			desc:  "check user owner exists",
			login: "liatrio",
			server: createServer("/", &responses{
				checkLogin: checkLoginResponse{
					User: checkLoginUser{
						Login: "liatrio",
					},
				},
				responseCode: http.StatusOK,
			}),
			expectedOwnerType:   "user",
			expectedOwnerExists: true,
		},
		{
			desc:  "error",
			login: "liatrio",
			server: createServer("/", &responses{
				checkLogin: checkLoginResponse{
					User: checkLoginUser{
						Login: "liatrio",
					},
				},
				responseCode: http.StatusNotFound,
			}),
			expectedOwnerExists: false,
			expectedOwnerType:   "",
			expectedError:       true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			factory := Factory{}
			defaultConfig := factory.CreateDefaultConfig()
			settings := receivertest.NewNopCreateSettings()
			ghs := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
			server := httptest.NewServer(tc.server)
			defer server.Close()

			client := graphql.NewClient(server.URL, ghs.client)
			ownerExists, ownerType, err := ghs.checkOwnerExists(context.Background(), client, tc.login)

			assert.Equal(t, tc.expectedOwnerExists, ownerExists)
			assert.Equal(t, tc.expectedOwnerType, ownerType)
			if !tc.expectedError {
				assert.NoError(t, err)
			}
		})
	}
}
