// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/shurcooL/githubv4"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/metadata"
)

var (
	errClientNotInitErr = errors.New("http client not initialized")
)

type PullRequest struct {
	Title       string
	CreatedDate time.Time
	ClosedDate  time.Time
}

// TODO: Keep this
type Repo struct {
	Name          string
	Owner         string
	DefaultBranch string
	//Branches      []Branch
	PullRequests []PullRequest
}

// TODO: Keep this
type githubScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
}

type PullRequestNode struct {
	Node struct {
		Title     string
		CreatedAt string
		ClosedAt  string
	}
}

func (ghs *githubScraper) start(_ context.Context, host component.Host) (err error) {
	ghs.logger.Sugar().Info("Starting the scraper inside scraper.go")
	// TODO: Fix the ToClient configuration
	ghs.client, err = ghs.cfg.ToClient(host, ghs.settings)
	return
}

func newGitHubScraper(
	_ context.Context,
	settings receiver.CreateSettings,
	cfg *Config,
) *githubScraper {
	return &githubScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}
}

func (ghs *githubScraper) getRepoPullRequestInformation(repo *Repo) {
	graphqlClient := githubv4.NewClient(ghs.client)

	var query struct {
		Repository struct {
			PullRequests struct {
				Edges    []PullRequestNode
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first: 100, after: $prCursor)"`
		} `graphql:"repository(name: $repoName, owner: $owner)"`
	}

	variables := map[string]interface{}{
		"repoName": githubv4.String(repo.Name),
		"owner":    githubv4.String(repo.Owner),
		"prCursor": (*githubv4.String)(nil),
	}

	var pullRequestsResults []PullRequestNode
	for {
		err := graphqlClient.Query(context.Background(), &query, variables)

		if err != nil {
			ghs.logger.Sugar().Errorf("Error getting branch details", zap.Error(err))
		}

		pullRequestsResults = append(pullRequestsResults, query.Repository.PullRequests.Edges...)

		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}

		variables["prCursor"] = githubv4.NewString(query.Repository.PullRequests.PageInfo.EndCursor)
	}

	for _, prNode := range pullRequestsResults {
		var closedDate time.Time

		creationDate, err := time.Parse(time.RFC3339, prNode.Node.CreatedAt)
		if err != nil {
			ghs.logger.Sugar().Errorf("Error converting timestamp for PR creation date", zap.Error(err))
		}

		if prNode.Node.ClosedAt != "" {
			closedDate, err = time.Parse(time.RFC3339, prNode.Node.ClosedAt)
			if err != nil {
				ghs.logger.Sugar().Errorf("Error converting timestamp for PR closed date", zap.Error(err))
			}
		} else {
			closedDate = time.Now()
		}

		pullRequest := &PullRequest{
			Title:       prNode.Node.Title,
			CreatedDate: creationDate,
			ClosedDate:  closedDate,
		}
		repo.PullRequests = append(repo.PullRequests, *pullRequest)
	}
}

// scrape and return metrics
func (ghs *githubScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	ghs.logger.Sugar().Debug("checking if client is initialized")
	if ghs.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	ghs.logger.Sugar().Debugf("current time: %v", now)

	currentDate := time.Now().Day()
	ghs.logger.Sugar().Debugf("current date: %v", currentDate)

	ghs.logger.Sugar().Debug("creating a new github client")

	// TODO: Below is the beginnning of the refactor to using genqlient
	// This is a secondary instantiation of the GraphQL client for the purpose of
	// using genqlient during the refactor.
	genClient := graphql.NewClient("https://api.github.com/graphql", ghs.client)

	exists, ownertype, err := ghs.checkOwnerExists(ctx, genClient, ghs.cfg.GitHubOrg)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner exists", zap.Error(err))
	}

	typeValid, err := checkOwnerTypeValid(ownertype)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error checking if owner type is valid", zap.Error(err))
	}

	var data interface{}
	var repoCursor *string

	if !exists || !typeValid {
		ghs.logger.Sugar().Error("error logging in and getting data from github")
		return ghs.mb.Emit(), err
	}

	sq := genDefaultSearchQuery(ownertype, ghs.cfg.GitHubOrg)

	if ghs.cfg.SearchQuery != "" {
		sq = ghs.cfg.SearchQuery
		ghs.logger.Sugar().Debugf("using search query where query is: %v", ghs.cfg.SearchQuery)
	}

	data, err = getRepoData(ctx, genClient, sq, ownertype, repoCursor)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	// TODO: setting this here for access from the proceeding for statement
	// gathering repo data
	var searchRepos []getRepoDataBySearchSearchSearchResultItemConnectionEdgesSearchResultItemEdge

	if searchData, ok := data.(*getRepoDataBySearchResponse); ok {
		ghs.logger.Sugar().Debug("successful search response")
		ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(searchData.Search.RepositoryCount))

		pages := getNumPages(float64(100), float64(searchData.Search.RepositoryCount))
		ghs.logger.Sugar().Debugf("pages: %v", pages)

		for i := 0; i < pages; i++ {
			results := searchData.GetSearch()
			searchRepos = append(searchRepos, results.Edges...)

			repoCursor = &searchData.Search.PageInfo.EndCursor
			data, err = getRepoData(ctx, genClient, sq, ownertype, repoCursor)
			if err != nil {
				ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
			}
		}

		ghs.logger.Sugar().Debugf("repos: %v", searchRepos)

	}

	// TODO: End of refactor to using genqlient

	// Slightly refactoring this and making it more nested during the refactor
	// to maintain parady with the original code while using genqlient and
	// not having to use the original query login interspection and types
	var branchCursor *string
	var branches []BranchNode

	if _, ok := data.(*getRepoDataBySearchResponse); ok {
		for _, repo := range searchRepos {
			var name string
			var defaultBranch string

			if n, ok := repo.Node.(*SearchNodeRepository); ok {
				name = n.Name
				defaultBranch = n.DefaultBranchRef.Name
			}

			// TODO: I don't think there's a need for this if we've stored the
			// entirety of the data previously & can iterate through the slice
			// directly. Might consider refactoring this later.
			repoInfo := &Repo{
				Name:          name,
				Owner:         ghs.cfg.GitHubOrg,
				DefaultBranch: defaultBranch,
			}

			count, err := getBranchCount(ctx, genClient, name, ghs.cfg.GitHubOrg)
			if err != nil {
				ghs.logger.Sugar().Errorf("error getting branch count", zap.Error(err))
			}
			ghs.logger.Sugar().Debugf("branch count: %v for repo %v", count.Repository.Refs.TotalCount, repo)

			ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(count.Repository.Refs.TotalCount), name)

			bp := getNumPages(float64(50), float64(count.Repository.Refs.TotalCount))
			ghs.logger.Sugar().Debugf("branch pages: %v for repo %v", bp, repo)

			for i := 0; i < bp; i++ {
				r, err := getBranchData(ctx, genClient, name, ghs.cfg.GitHubOrg, 50, defaultBranch, branchCursor)
				if err != nil {
					ghs.logger.Sugar().Errorf("error getting branch data", zap.Error(err))
				}

				branches = append(branches, r.Repository.Refs.Nodes...)

				branchCursor = &r.Repository.Refs.PageInfo.EndCursor

			}

			for _, branch := range branches {
				// We're using BehindBy here because we're comparing against the target
				// branch, which is the default branch. In essence the response is saying
				// the default branch is behind the queried branch by X commits which is
				// the number of commits made to the queried branch but not merged into
				// the default branch. Doing it this way involves less queries because
				// we don't have to know the queried branch name ahead of time.
				cp := getNumPages(float64(100), float64(branch.Compare.BehindBy))

				var cc *string

				for i := 0; i < cp; i++ {
					if branch.Name == defaultBranch || branch.Compare.BehindBy == 0 {
						break
					}

					c, err := getCommitData(ctx, genClient, name, ghs.cfg.GitHubOrg, 1, 100, cc, branch.Name)
					if err != nil {
						ghs.logger.Sugar().Errorf("error getting commit data", zap.Error(err))
					}

					tar := c.Repository.GetRefs().Nodes[0].GetTarget()
					if ct, ok := tar.(*CommitNodeTargetCommit); ok {
						cc = &ct.History.PageInfo.EndCursor

						if i == cp-1 {
							e := ct.History.GetEdges()

							oldest := e[len(e)-1].Node.GetCommittedDate()
							age := int64(time.Since(oldest).Hours())

							ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, age, name, branch.Name)
						}
					}
				}
			}

			ghs.getRepoPullRequestInformation(repoInfo)
		}

	}

	return ghs.mb.Emit(), nil
}
