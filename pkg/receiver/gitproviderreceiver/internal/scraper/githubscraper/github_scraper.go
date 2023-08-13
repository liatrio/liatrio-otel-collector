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

type RepositoryEdge struct {
	Node struct {
		Name             string
		DefaultBranchRef struct {
			Name string
		}
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

	oldestCommit, err := time.Parse(time.RFC3339, query.Repository.Ref.Target.Commit.History.Edges[0].Node.CommittedDate)

	if err != nil {
		ghs.logger.Sugar().Errorf("Error converting timestamp for oldest commit", zap.Error(err))
	}

	branch.CreatedDate = oldestCommit
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

	//graphqlClient := githubv4.NewClient(ghs.client)

	// This has been moved forward with the refactor due to new logic for confirming
	// if the provided org is a user or an organization
	//var provided_org_is_org bool

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

	data, err = getRepoData(ctx, genClient, ghs.cfg.GitHubOrg, ownertype, repoCursor)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
		return ghs.mb.Emit(), err
	}

	// TODO: setting this here for access from the proceeding for statement
	// gathering repo data
	var orgRepos []getOrgRepoDataOrganizationRepositoriesRepositoryConnectionEdgesRepositoryEdge
	var userRepos []getUserRepoDataUserRepositoriesRepositoryConnectionEdgesRepositoryEdge

	if orgData, ok := data.(*getOrgRepoDataResponse); ok {
		ghs.logger.Sugar().Debug("successful response for organization")
		ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(orgData.Organization.Repositories.TotalCount))

		// TODO: this is a temporary fix for the refactor to using genqlient and should
		// be removed once the refactor is complete
		//provided_org_is_org = true
		// END TODO

		pages := getNumPages(float64(orgData.Organization.Repositories.TotalCount))
		ghs.logger.Sugar().Debugf("pages: %v", pages)

		for i := 0; i < pages; i++ {
			orgRepos = append(orgRepos, orgData.Organization.Repositories.Edges...)

			repoCursor = &orgData.Organization.Repositories.PageInfo.EndCursor
			data, err = getRepoData(ctx, genClient, ghs.cfg.GitHubOrg, ownertype, repoCursor)
			if err != nil {
				ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
			}
		}

		ghs.logger.Sugar().Debugf("repos: %v", orgRepos)

	} else if userData, ok := data.(*getUserRepoDataResponse); ok {
		ghs.logger.Sugar().Debug("successful response for organization")
		ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(userData.User.Repositories.TotalCount))

		// TODO: this is a temporary fix for the refactor to using genqlient and should
		// be removed once the refactor is complete
		//provided_org_is_org = false
		// END TODO

		pages := getNumPages(float64(userData.User.Repositories.TotalCount))
		ghs.logger.Sugar().Debugf("pages: %v", pages)

		for i := 0; i < pages; i++ {
			userRepos = append(userRepos, userData.User.Repositories.Edges...)

			repoCursor = &userData.User.Repositories.PageInfo.EndCursor
			data, err = getRepoData(ctx, genClient, ghs.cfg.GitHubOrg, ownertype, repoCursor)
			if err != nil {
				ghs.logger.Sugar().Errorf("Error getting repo data", zap.Error(err))
			}
		}

		ghs.logger.Sugar().Debugf("repos: %v", userRepos)

	}
	// TODO: End of refactor to using genqlient

	// Slightly refactoring this and making it more nested during the refactor
	// to maintain parady with the original code while using genqlient and
	// not having to use the original query login interspection and types
	if _, ok := data.(*getOrgRepoDataResponse); ok {
		for _, repo := range orgRepos {
			repoInfo := &Repo{Name: repo.Node.Name, Owner: ghs.cfg.GitHubOrg, DefaultBranch: repo.Node.DefaultBranchRef.Name}

			ghs.getRepoBranchInformation(repoInfo)

			numOfBranches := len(repoInfo.Branches)

			ghs.logger.Sugar().Debugf("Repo Name: %v", repoInfo.Name)
			ghs.logger.Sugar().Debugf("Num of Branches: %v", numOfBranches)

			ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(numOfBranches), repoInfo.Name)

			for _, branch := range repoInfo.Branches {
				if branch.Name != repoInfo.DefaultBranch {
					branch := branch
					ghs.getOldestBranchCommit(repoInfo, &branch)
					branchAge := int64(time.Since(branch.CreatedDate).Hours())
					ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, branchAge, repoInfo.Name, branch.Name)
				}
			}

			ghs.getRepoPullRequestInformation(repoInfo)

			for _, pr := range repoInfo.PullRequests {
				ghs.logger.Sugar().Debugf("PR Creation Date: %v PR Closed Date %v", pr.CreatedDate.Format(time.RFC3339), pr.ClosedDate.Format(time.RFC3339))
			}
		}

	} else if _, ok := data.(*getUserRepoDataResponse); ok {
		for _, repo := range userRepos {
			repoInfo := &Repo{Name: repo.Node.Name, Owner: ghs.cfg.GitHubOrg, DefaultBranch: repo.Node.DefaultBranchRef.Name}

			ghs.getRepoBranchInformation(repoInfo)

			numOfBranches := len(repoInfo.Branches)

			ghs.logger.Sugar().Debugf("Repo Name: %v", repoInfo.Name)
			ghs.logger.Sugar().Debugf("Num of Branches: %v", numOfBranches)

			ghs.mb.RecordGitRepositoryBranchCountDataPoint(now, int64(numOfBranches), repoInfo.Name)

			for _, branch := range repoInfo.Branches {
				if branch.Name != repoInfo.DefaultBranch {
					branch := branch
					ghs.getOldestBranchCommit(repoInfo, &branch)
					branchAge := int64(time.Since(branch.CreatedDate).Hours())
					ghs.mb.RecordGitRepositoryBranchTimeDataPoint(now, branchAge, repoInfo.Name, branch.Name)
				}
			}

			ghs.getRepoPullRequestInformation(repoInfo)

			for _, pr := range repoInfo.PullRequests {
				ghs.logger.Sugar().Debugf("PR Creation Date: %v PR Closed Date %v", pr.CreatedDate.Format(time.RFC3339), pr.ClosedDate.Format(time.RFC3339))
			}
		}
	}

	//variables := map[string]interface{}{
	//	"login": githubv4.String(ghs.cfg.GitHubOrg),
	//    "repoCursor": (*githubv4.String)(nil),
	//}

	//variables["repoCursor"] = (*githubv4.String)(nil)

	//var user_query struct {
	//	User struct {
	//		Repositories struct {
	//			TotalCount int
	//			PageInfo   struct {
	//				EndCursor   githubv4.String
	//				HasNextPage bool
	//			}
	//			Edges []RepositoryEdge
	//		} `graphql:"repositories(first: 100, affiliations: OWNER, after: $repoCursor, isArchived: false, isFork: false)"`
	//	} `graphql:"user(login: $login)"`
	//}

	//var org_query struct {
	//	Organization struct {
	//		Repositories struct {
	//			TotalCount int
	//			PageInfo   struct {
	//				EndCursor   githubv4.String
	//				HasNextPage bool
	//			}
	//			Edges []RepositoryEdge
	//		} `graphql:"repositories(first: 100, after: $repoCursor, affiliations: OWNER, isArchived: false, isFork: false)"`
	//	} `graphql:"organization(login: $login)"`
	//}

	//var repos []RepositoryEdge
	//	for {
	//
	//		// query must be dynamic to user or organization type for Graphql
	//		if provided_org_is_org {
	//			err := graphqlClient.Query(context.Background(), &org_query, variables)
	//
	//			if err != nil {
	//				ghs.logger.Sugar().Errorf("Error getting all Repositories", zap.Error(err))
	//			}
	//			repos = append(repos, org_query.Organization.Repositories.Edges...)
	//			ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(org_query.Organization.Repositories.TotalCount))
	//
	//			if !org_query.Organization.Repositories.PageInfo.HasNextPage {
	//				break
	//			}
	//			variables["repoCursor"] = githubv4.NewString(org_query.Organization.Repositories.PageInfo.EndCursor)
	//		} else {
	//			err := graphqlClient.Query(context.Background(), &user_query, variables)
	//
	//			if err != nil {
	//				ghs.logger.Sugar().Errorf("Error getting all Repositories", zap.Error(err))
	//			}
	//			repos = append(repos, user_query.User.Repositories.Edges...)
	//			ghs.mb.RecordGitRepositoryCountDataPoint(now, int64(user_query.User.Repositories.TotalCount))
	//
	//			if !user_query.User.Repositories.PageInfo.HasNextPage {
	//				break
	//			}
	//			variables["repoCursor"] = githubv4.NewString(user_query.User.Repositories.PageInfo.EndCursor)
	//		}
	//	}

	return ghs.mb.Emit(), nil
}
