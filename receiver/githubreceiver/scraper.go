package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/shurcooL/githubv4"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

var (
	clientNotInitErr = errors.New("http client not initialized")
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
	CreatedDate string
	ClosedDate  string
}

type Repo struct {
	Name          string
	Owner         string
	DefaultBranch string
	Branches      []Branch
	PullRequests  []PullRequest
}

type ghScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
}

func (ghs *ghScraper) start(ctx context.Context, host component.Host) (err error) {
	ghs.logger.Sugar().Info("Starting the scraper inside scraper.go")
	ghs.client, err = ghs.cfg.ToClient(host, ghs.settings)
	return
}

func newScraper(cfg *Config, settings receiver.CreateSettings) *ghScraper {
	return &ghScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}
}

func (ghs *ghScraper) getRepoBranchInformation(repo *Repo) {
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

func (ghs *ghScraper) getOldestBranchCommit(repo *Repo, branch *Branch) {
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

func (ghs *ghScraper) getRepoPullRequestInformation(repo *Repo) {
	graphqlClient := githubv4.NewClient(ghs.client)

	type PullRequestNode struct {
		Node struct {
			Title     string
			CreatedAt string
			ClosedAt  string
		}
	}

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

		pullRequest := &PullRequest{
			Title:       prNode.Node.Title,
			CreatedDate: prNode.Node.CreatedAt,
			ClosedDate:  prNode.Node.ClosedAt,
		}
		repo.PullRequests = append(repo.PullRequests, *pullRequest)
	}
}

// scrape and return metrics
func (ghs *ghScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	ghs.logger.Sugar().Debug("checking if client is initialized")
	if ghs.client == nil {
		return pmetric.NewMetrics(), clientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	ghs.logger.Sugar().Debugf("current time: %v", now)

	currentDate := time.Now().Day()
	ghs.logger.Sugar().Debugf("current date: %v", currentDate)

	ghs.logger.Sugar().Debug("creating a new github client")
	client := github.NewClient(ghs.client)

	repos, _, err := client.Repositories.List(ctx, ghs.cfg.GitHubOrg, nil)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting repos", zap.Error(err))
	}

	ghs.mb.RecordGhRepoCountDataPoint(now, int64(len(repos)), ghs.cfg.GitHubOrg)

	for _, repo := range repos {
		repoInfo := &Repo{Name: *repo.Name, Owner: ghs.cfg.GitHubOrg, DefaultBranch: *repo.DefaultBranch}

		ghs.getRepoBranchInformation(repoInfo)

		numOfBranches := len(repoInfo.Branches)

		ghs.logger.Sugar().Debugf("Repo Name: %v", repoInfo.Name)
		ghs.logger.Sugar().Debugf("Num of Branches: %v", numOfBranches)

		ghs.mb.RecordGhRepoBranchesCountDataPoint(now, int64(numOfBranches), repoInfo.Name, repoInfo.Owner)

		var totalBranchLife float64
		for _, branch := range repoInfo.Branches {
			if branch.Name != repoInfo.DefaultBranch {
				ghs.logger.Sugar().Debugf("Commit Count: %v", branch.CommitCount)

				ghs.getOldestBranchCommit(repoInfo, &branch)
				branchAge := time.Now().Sub(branch.CreatedDate).Hours()

				ghs.logger.Sugar().Debugf("Branch Age: %v", branchAge)

				totalBranchLife += branchAge
			}
		}
		ghs.logger.Sugar().Debugf("Mean Branch Age: %v", totalBranchLife/float64(numOfBranches))

		ghs.getRepoPullRequestInformation(repoInfo)

		ghs.logger.Sugar().Debugf("PRs: %v", repoInfo.PullRequests)
	}

	ghs.logger.Sugar().Debugf("metrics: %v", ghs.cfg.Metrics.GhRepoCount)
	return ghs.mb.Emit(), nil
}
