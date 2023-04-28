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
		// branch list
		ghs.logger.Sugar().Debugf("Repo Name: %v", *repo.Name)
		defaultBranch := *repo.DefaultBranch

		graphqlClient := githubv4.NewClient(ghs.client)

		var branchQuery struct {
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

		queryVars := map[string]interface{}{
			"repoName": githubv4.String(*repo.Name),
			"owner":    githubv4.String(ghs.cfg.GitHubOrg),
		}

		queryErr := graphqlClient.Query(context.Background(), &branchQuery, queryVars)

		if queryErr != nil {
			ghs.logger.Sugar().Errorf("Error getting branch details", zap.Error(err))
		}

		numOfBranches := branchQuery.Repository.Refs.TotalCount
		ghs.logger.Sugar().Debugf("Num of Branches: %v", numOfBranches)

		//ghs.mb.RecordGhRepoBranchesCountDataPoint(now, int64(numOfBranches), *repo.Name, ghs.cfg.GitHubOrg)

		var totalBranchLife float64
		for _, branch := range branchQuery.Repository.Refs.Nodes {
			if branch.Name != defaultBranch {
				ghs.logger.Sugar().Debugf("Commit Count: %v", branch.Target.Commit.History.TotalCount)

				cursor := branch.Target.Commit.History.PageInfo.EndCursor

				var oldestCommitQuery struct {
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

				commitQueryVars := map[string]interface{}{
					"repoName":   githubv4.String(*repo.Name),
					"owner":      githubv4.String(ghs.cfg.GitHubOrg),
					"branchName": githubv4.String(branch.Name),
					"endCursor":  githubv4.String(cursor),
				}

				ghs.logger.Sugar().Debugf("Repo Name: %v", *repo.Name)
				ghs.logger.Sugar().Debugf("Owner: %v", ghs.cfg.GitHubOrg)
				ghs.logger.Sugar().Debugf("Branch Name: %v", branch.Name)

				queryErr := graphqlClient.Query(context.Background(), &oldestCommitQuery, commitQueryVars)

				if queryErr != nil {
					ghs.logger.Sugar().Errorf("Error getting oldest commit", zap.Error(err))
				}

				oldestCommit, err := time.Parse(time.RFC3339, oldestCommitQuery.Repository.Ref.Target.Commit.History.Edges[0].Node.CommittedDate)

				if err != nil {
					ghs.logger.Sugar().Errorf("Error converting timestamp for oldest commit", zap.Error(err))
				}

				branchAge := time.Now().Sub(oldestCommit).Hours()
				ghs.logger.Sugar().Debugf("Branch Age: %v", branchAge)

				totalBranchLife += branchAge
			}
		}
		ghs.logger.Sugar().Debugf("Mean Branch Age: %v", totalBranchLife/float64(numOfBranches))

		type PullRequest struct {
			Node struct {
				CreatedAt string
				ClosedAt  string
			}
		}

		var prQuery struct {
			Repository struct {
				PullRequests struct {
					Edges    []PullRequest
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage bool
					}
				} `graphql:"pullRequests(first: 100, after: $prCursor)"`
			} `graphql:"repository(name: $repoName, owner: $owner)"`
		}

		queryVars = map[string]interface{}{
			"repoName": githubv4.String(*repo.Name),
			"owner":    githubv4.String(ghs.cfg.GitHubOrg),
			"prCursor": (*githubv4.String)(nil),
		}

		var pullRequests []PullRequest
		for {
			queryErr := graphqlClient.Query(context.Background(), &prQuery, queryVars)

			if queryErr != nil {
				ghs.logger.Sugar().Errorf("Error getting branch details", zap.Error(err))
			}


			pullRequests = append(pullRequests, prQuery.Repository.PullRequests.Edges...)
			if !prQuery.Repository.PullRequests.PageInfo.HasNextPage {
				break
			}

			queryVars["prCursor"] = githubv4.NewString(prQuery.Repository.PullRequests.PageInfo.EndCursor)
		}

	}

	ghs.logger.Sugar().Debugf("metrics: %v", ghs.cfg.Metrics.GhRepoCount)
	return ghs.mb.Emit(), nil
}
