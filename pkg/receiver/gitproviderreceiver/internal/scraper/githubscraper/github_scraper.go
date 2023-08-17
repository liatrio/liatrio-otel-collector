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

type Commit struct {
	Date string
}

type Branch struct {
	Name            string
	CommitCount     int
	CreatedDate     time.Time
	LastUpdatedDate string
	EndCursor       string
}

type PullRequest struct {
	Title       string
	CreatedDate time.Time
	ClosedDate  time.Time
}

type Repo struct {
	Name          string
	Owner         string
	DefaultBranch string
	Branches      []Branch
	PullRequests  []PullRequest
}

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

func (ghs *githubScraper) getRepoBranchInformation(repo *Repo) {
	graphqlClient := githubv4.NewClient(ghs.client)

	var query struct {
		Repository struct {
			Refs struct {
				TotalCount int
				Nodes      []struct {
					Name   string
					Target struct {
						Commit struct {
							History struct {
								TotalCount int
								Edges      []struct {
									Node struct {
										CommittedDate string
									}
								}
								PageInfo struct {
									EndCursor string
								}
							}
						} `graphql:"... on Commit"`
					}
				}
			} `graphql:"refs(refPrefix: \"refs/heads/\", first: 100)"`
		} `graphql:"repository(name: $repoName, owner: $owner)"`
	}

	variables := map[string]interface{}{
		"repoName": githubv4.String(repo.Name),
		"owner":    githubv4.String(repo.Owner),
	}

	err := graphqlClient.Query(context.Background(), &query, variables)

	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting branch details", zap.Error(err))
	}

	for _, branch := range query.Repository.Refs.Nodes {
		newBranch := &Branch{
			Name:            branch.Name,
			CommitCount:     branch.Target.Commit.History.TotalCount,
			LastUpdatedDate: branch.Target.Commit.History.Edges[0].Node.CommittedDate,
			EndCursor:       branch.Target.Commit.History.PageInfo.EndCursor,
		}
		repo.Branches = append(repo.Branches, *newBranch)
	}
}

// func (ghs *githubScraper) getOldestBranchCommit(repo *Repo, branch *Branch) {
func (ghs *githubScraper) getOldestBranchCommit(repo *Repo, branch *Branch) {
	graphqlClient := githubv4.NewClient(ghs.client)

	var query struct {
		Repository struct {
			Ref struct {
				Target struct {
					Commit struct {
						History struct {
							Edges []struct {
								Node struct {
									CommittedDate string
								}
							}
							PageInfo struct {
								EndCursor string
							}
						} `graphql:"history(last: 1, before: $endCursor)"`
					} `graphql:"... on Commit"`
				}
			} `graphql:"ref(qualifiedName: $branchName)"`
		} `graphql:"repository(name: $repoName, owner: $owner)"`
	}

	variables := map[string]interface{}{
		"repoName":   githubv4.String(repo.Name),
		"owner":      githubv4.String(repo.Owner),
		"branchName": githubv4.String(branch.Name),
		"endCursor":  githubv4.String(branch.EndCursor),
	}

	err := graphqlClient.Query(context.Background(), &query, variables)

	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting oldest commit", zap.Error(err))
	}

	if len(query.Repository.Ref.Target.Commit.History.Edges) > 0 {
		oldestCommit, err := time.Parse(time.RFC3339, query.Repository.Ref.Target.Commit.History.Edges[0].Node.CommittedDate)

		if err != nil {
			ghs.logger.Sugar().Errorf("Error converting timestamp for oldest commit", zap.Error(err))
		}

		branch.CreatedDate = oldestCommit
	} else {
		branch.CreatedDate = time.Now()
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

		pages := getNumPages(float64(searchData.Search.RepositoryCount))
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
	if _, ok := data.(*getRepoDataBySearchResponse); ok {
		for _, repo := range searchRepos {
			var name string
			var defaultBranch string

			if n, ok := repo.Node.(*SearchNodeRepository); ok {
				name = n.Name
				defaultBranch = n.DefaultBranchRef.Name
			}

			repoInfo := &Repo{
				Name:          name,
				Owner:         ghs.cfg.GitHubOrg,
				DefaultBranch: defaultBranch,
			}

			ghs.getRepoBranchInformation(repoInfo)

			numOfBranches := len(repoInfo.Branches)

			ghs.logger.Sugar().Debugf("Repo Name: %v", repoInfo.Name)
			ghs.logger.Sugar().Debugf("Num of Branches: %v", numOfBranches)

			ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(numOfBranches), repoInfo.Name)

			if numOfBranches > 1 {
				for _, branch := range repoInfo.Branches {
					if branch.Name != repoInfo.DefaultBranch {
						branch := branch
						ghs.getOldestBranchCommit(repoInfo, &branch)
						branchAge := int64(time.Since(branch.CreatedDate).Hours())
						ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, branchAge, repoInfo.Name, branch.Name)
					}
				}
			}

			ghs.getRepoPullRequestInformation(repoInfo)
		}

	}

	return ghs.mb.Emit(), nil
}
