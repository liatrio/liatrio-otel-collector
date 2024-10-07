// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper // import "github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/scraper/githubscraper"

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/go-github/v65/github"
	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"
	"go.uber.org/zap"
)

const (
	// The default public GitHub GraphQL Endpoint
	defaultGraphURL = "https://api.github.com/graphql"
	// The default maximum number of items to be returned in a GraphQL query.
	defaultReturnItems = 100
)

func (ghs *githubScraper) getRepos(
	ctx context.Context,
	client graphql.Client,
	searchQuery string,
) ([]Repo, int, error) {
	// here we use a pointer to a string so that graphql will receive null if the
	// value is not set since the after: $repoCursor is optional to graphql
	var cursor *string
	var repos []Repo
	var count int

	switch {
	case ghs.cfg.GitHubTeam != "":
		for next := true; next; {
			r, err := getRepoDataByTeam(ctx, client, ghs.cfg.GitHubOrg, ghs.cfg.GitHubTeam, cursor)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting repo data by team", zap.Error(err))
				return nil, 0, err
			}
			for _, repo := range r.Organization.Team.Repositories.Nodes {

				repos = append(repos, repo.Repo)

			}
			count = r.Organization.Team.Repositories.TotalCount
			cursor = &r.Organization.Team.Repositories.PageInfo.EndCursor
			next = r.Organization.Team.Repositories.PageInfo.HasNextPage
		}
	default:
		for next := true; next; {
			r, err := getRepoDataBySearch(ctx, client, searchQuery, cursor)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting repo data by search", zap.Error(err))
				return nil, 0, err
			}

			for _, repo := range r.Search.Nodes {
				if r, ok := repo.(*SearchNodeRepository); ok {
					repos = append(repos, r.Repo)
				}
			}

			count = r.Search.RepositoryCount
			cursor = &r.Search.PageInfo.EndCursor
			next = r.Search.PageInfo.HasNextPage
		}
	}

	return repos, count, nil
}

func (ghs *githubScraper) getBranches(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	defaultBranch string,
) ([]BranchNode, int, error) {
	var cursor *string
	var count int
	var branches []BranchNode

	for next := true; next; {
		// Instead of using the defaultReturnItems (100) we chose to set it to
		// 50 because GitHub has been known to kill the connection server side
		// when trying to get items over 80 on the getBranchData query.
		items := 50
		r, err := getBranchData(ctx, client, repoName, ghs.cfg.GitHubOrg, items, defaultBranch, cursor)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting branch data", zap.Error(err))
			return nil, 0, err
		}
		count = r.Repository.Refs.TotalCount
		cursor = &r.Repository.Refs.PageInfo.EndCursor
		next = r.Repository.Refs.PageInfo.HasNextPage
		branches = append(branches, r.Repository.Refs.Nodes...)
	}
	return branches, count, nil
}

// Login via the GraphQL checkLogin query in order to ensure that the user
// and it's credentials are valid and return the type of user being authenticated.
func (ghs *githubScraper) login(
	ctx context.Context,
	client graphql.Client,
	owner string,
) (string, error) {
	var loginType string

	// The checkLogin GraphQL query will always return an error. We only return
	// the error if the login response for User and Organization are both nil.
	// This is represented by checking to see if each resp.*.Login resolves to equal the owner.
	resp, err := checkLogin(ctx, client, ghs.cfg.GitHubOrg)

	// These types are used later to generate the default string for the search query
	// and thus must match the convention for user: and org: searches in GitHub
	switch {
	case resp.User.Login == owner:
		loginType = "user"
	case resp.Organization.Login == owner:
		loginType = "org"
	default:
		return "", err
	}

	return loginType, nil
}

// Returns the default search query string based on input of owner type
// and GitHubOrg name with a default of archived:false to ignore archived repos
func genDefaultSearchQuery(ownertype string, ghorg string) string {
	return fmt.Sprintf("%s:%s archived:false", ownertype, ghorg)
}

// Returns the graphql and rest clients for GitHub.
// By default, the graphql client will use the public GitHub API URL as will
// the rest client. If the user has specified an endpoint in the config via the
// inherited ClientConfig, then the both clients will use that endpoint.
// The endpoint defined needs to be the root server.
// See the GitHub documentation for more information.
// https://docs.github.com/en/graphql/guides/forming-calls-with-graphql#the-graphql-endpoint
// https://docs.github.com/en/enterprise-server@3.8/graphql/guides/forming-calls-with-graphql#the-graphql-endpoint
// https://docs.github.com/en/enterprise-server@3.8/rest/guides/getting-started-with-the-rest-api#making-a-request
func (ghs *githubScraper) createClients() (gClient graphql.Client, rClient *github.Client, err error) {
	rClient = github.NewClient(ghs.client)
	gClient = graphql.NewClient(defaultGraphURL, ghs.client)

	if ghs.cfg.ClientConfig.Endpoint != "" {

		// Given endpoint set as `https://myGHEserver.com` we need to join the path
		// with `api/graphql`
		gu, err := url.JoinPath(ghs.cfg.ClientConfig.Endpoint, "api/graphql")
		if err != nil {
			ghs.logger.Sugar().Errorf("error joining graphql endpoint: %v", err)
			return nil, nil, err
		}
		gClient = graphql.NewClient(gu, ghs.client)

		// The rest client needs the endpoint to be the root of the server
		ru := ghs.cfg.ClientConfig.Endpoint
		rClient, err = github.NewClient(ghs.client).WithEnterpriseURLs(ru, ru)
		if err != nil {
			ghs.logger.Sugar().Errorf("error creating enterprise client: %v", err)
			return nil, nil, err
		}
	}

	return gClient, rClient, nil
}

// Get the contributor count for a repository via the REST API
func (ghs *githubScraper) getContributorCount(
	ctx context.Context,
	client *github.Client,
	repoName string,
) (int, error) {
	var all []*github.Contributor

	// Options for Pagination support, default from GitHub was 30
	// https://docs.github.com/en/rest/repos/repos#list-repository-contributors
	opt := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{PerPage: defaultReturnItems},
	}

	for {
		contribs, resp, err := client.Repositories.ListContributors(ctx, ghs.cfg.GitHubOrg, repoName, opt)
		if err != nil {
			ghs.logger.Sugar().Errorf("error getting contributor count", zap.Error(err))
			return 0, err
		}

		all = append(all, contribs...)
		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return len(all), nil
}

// Get the pull request data from the GraphQL API.
func (ghs *githubScraper) getPullRequests(
	ctx context.Context,
	client graphql.Client,
	repoName string,
) ([]PullRequestNode, error) {
	var cursor *string
	var pullRequests []PullRequestNode

	for hasNextPage := true; hasNextPage; {
		prs, err := getPullRequestData(
			ctx,
			client,
			repoName,
			ghs.cfg.GitHubOrg,
			defaultReturnItems,
			cursor,
			[]PullRequestState{"OPEN", "MERGED"},
		)
		if err != nil {
			return nil, err
		}

		pullRequests = append(pullRequests, prs.Repository.PullRequests.Nodes...)
		cursor = &prs.Repository.PullRequests.PageInfo.EndCursor
		hasNextPage = prs.Repository.PullRequests.PageInfo.HasNextPage
	}

	return pullRequests, nil
}

func (ghs *githubScraper) evalCommits(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	branch BranchNode,
) (additions int, deletions int, age int64, err error) {
	var cursor *string
	items := defaultReturnItems

	// See https://github.com/liatrio/liatrio-otel-collector/blob/main/receiver/githubreceiver/internal/scraper/githubscraper/README.md#github-limitations
	// for more information as to why `BehindBy` and `AheadBy` are
	// swapped.
	pages := getNumPages(float64(defaultReturnItems), float64(branch.Compare.BehindBy))

	for page := 1; page <= pages; page++ {
		if page == pages {
			// We need to make sure that the last page is retrieved properly
			// when it's a completely full page, so if the remainder is 0 we'll
			// reset to the defaultReturnItems value to ensure the items
			// request sent to the getCommitData function is accurate.
			items = branch.Compare.BehindBy % defaultReturnItems
			if items == 0 {
				items = defaultReturnItems
			}
		}
		c, err := ghs.getCommitData(ctx, client, repoName, items, cursor, branch.Name)
		if err != nil {
			ghs.logger.Sugar().Errorf("error making graphql query to get commit data", zap.Error(err))
			return 0, 0, 0, err
		}

		// GraphQL could return empty commit nodes so here we confirm that
		// commits were returned to prevent an index out of range error. This
		// technically should never be triggered because of other preceding
		// catches, but to be safe we check.
		if len(c.Nodes) == 0 {
			break
		}

		cursor = &c.PageInfo.EndCursor
		if page == pages {
			node := c.GetNodes()
			oldest := node[len(node)-1].GetCommittedDate()
			age = int64(time.Since(oldest).Seconds())
		}
		for b := 0; b < len(c.Nodes); b++ {
			additions += c.Nodes[b].Additions
			deletions += c.Nodes[b].Deletions
		}

	}
	return additions, deletions, age, nil
}

func (ghs *githubScraper) getCommitData(
	ctx context.Context,
	client graphql.Client,
	repoName string,
	items int,
	cursor *string,
	branchName string,
) (*BranchHistoryTargetCommitHistoryCommitHistoryConnection, error) {
	data, err := getCommitData(ctx, client, repoName, ghs.cfg.GitHubOrg, 1, items, cursor, branchName)
	if err != nil {
		return nil, err
	}

	// This checks to ensure that the query returned a BranchHistory Node. The
	// way the GraphQL query functions allows for a successful query to take
	// place, but have an empty set of branches. The only time this query would
	// return an empty BranchHistory Node is if the branch was deleted between
	// the time the list of branches was retrieved, and the query for the
	// commits on the branch.
	if len(data.Repository.Refs.Nodes) == 0 {
		return nil, errors.New("no branch history returned from the commit data request")
	}

	tar := data.Repository.Refs.Nodes[0].GetTarget()

	// We do a sanity type check just to make sure the GraphQL response was
	// indead for commits. This is a byproduct of the `... on Commit` syntax
	// within the GraphQL query and then return the actual history if the
	// returned Target is inded of type Commit.
	if ct, ok := tar.(*BranchHistoryTargetCommit); ok {
		return &ct.History, nil
	}

	return nil, errors.New("GraphQL query did not return the Commit Target")
}

func getNumPages(p float64, n float64) int {
	numPages := math.Ceil(n / p)

	return int(numPages)
}

// Get the age/duration between two times in seconds.
func getAge(start time.Time, end time.Time) int64 {
	return int64(end.Sub(start).Seconds())
}

func (ghs *githubScraper) getCVEs(
	ctx context.Context,
	gClient graphql.Client,
	rClient *github.Client,
	repo string,
) (map[metadata.AttributeCveSeverity]int64, error) {
	d := ghs.getDepBotAlerts(ctx, gClient, repo)
	c := ghs.getCodeScanAlerts(ctx, rClient, repo)

	return mapSeverities(d, c), nil
}

func (ghs *githubScraper) getDepBotAlerts(
	ctx context.Context,
	gClient graphql.Client,
	repo string,
) []CVENode {

	var alerts []CVENode
	var cursor *string

	for hasNextPage := true; hasNextPage; {
		a, err := getRepoCVEs(ctx, gClient, ghs.cfg.GitHubOrg, repo, cursor)

		if err != nil {
			ghs.logger.Sugar().Errorf("error %v getting dependabot alerts from repo %s", zap.Error(err), repo)
			return nil
		}

		hasNextPage = a.Repository.VulnerabilityAlerts.PageInfo.HasNextPage
		cursor = &a.Repository.VulnerabilityAlerts.PageInfo.EndCursor
		alerts = append(alerts, a.Repository.VulnerabilityAlerts.Nodes...)

	}

	return alerts
}

// Get the Code Scanning Alerts count for a repository via the REST API
func (ghs *githubScraper) getCodeScanAlerts(
	ctx context.Context,
	rClient *github.Client,
	repo string,
) []*github.Alert {
	var alerts []*github.Alert

	// Options for Pagination support, default from GitHub was 30. Max is 100
	// https://docs.github.com/en/rest/code-scanning/code-scanning?apiVersion=2022-11-28
	opt := &github.AlertListOptions{
		ListOptions: github.ListOptions{PerPage: 50},
		State:       "open",
	}

	for {
		a, resp, err := rClient.CodeScanning.ListAlertsForRepo(ctx, ghs.cfg.GitHubOrg, repo, opt)
		if err != nil {
			if resp.StatusCode == 404 || resp.StatusCode == 403 {
				ghs.logger.Sugar().Debugf("%s repo does not have any alerts or does not have alerts enabled", repo)
				break
			}
			ghs.logger.Sugar().Errorf("error getting code scanning alerts from repo", zap.Error(err))
			return nil
		}

		alerts = append(alerts, a...)
		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	return alerts
}

func mapSeverities(
	nodes []CVENode,
	alerts []*github.Alert,
) map[metadata.AttributeCveSeverity]int64 {

	// Allows us to map the "MODERATE" to the conventional "medium" and support
	// the capital cased values that are returned from GitHub's API.
	mapping := map[string]metadata.AttributeCveSeverity{
		"CRITICAL": metadata.AttributeCveSeverityCritical,
		"HIGH":     metadata.AttributeCveSeverityHigh,
		"MODERATE": metadata.AttributeCveSeverityMedium,
		"MEDIUM":   metadata.AttributeCveSeverityMedium,
		"LOW":      metadata.AttributeCveSeverityLow,
	}
	m := make(map[metadata.AttributeCveSeverity]int64)

	for _, node := range nodes {
		if val, found := mapping[strings.ToUpper(string(node.SecurityVulnerability.Severity))]; found {
			m[val]++
		}
	}

	for _, alert := range alerts {
		if val, found := mapping[strings.ToUpper(*alert.Rule.SecuritySeverityLevel)]; found {
			m[val]++
		}
	}

	return m
}
