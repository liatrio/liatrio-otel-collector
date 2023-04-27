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

// TODO: This is pretty bad because it's making an API call per commit
// TODO: This needs to be unit tested & might not be the best way to do this
// TODO: Commenting this out for now given the slowness and rate limiting of the GitHub API
// Checks if the commit paassed is in the parent branch to determine if the
// branch at given point of commit is divergent.
//func diverges(ctx context.Context, client *github.Client, org string, repo string, parentBranch string, commit string) (bool, error) {
//    comparison, resp, err := client.Repositories.CompareCommits(ctx, org, repo, parentBranch, commit, nil)
//    if err != nil {
//        // 404 is returned when the commit doesn't exist in the parent branch
//        if resp.StatusCode == 404 {
//            return true , nil
//        }
//        return false, err
//    }
//    return comparison.GetStatus() != "identical", nil
//}

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

	graphqlClient := githubv4.NewClient(ghs.client)

	type commit struct {
		Node struct {
			CommittedDate string
		}
	}

	type ref struct {
		Name   string
		Target struct {
			Commit struct {
				History struct {
					TotalCount int
					Edges      []commit
					PageInfo   struct {
						EndCursor string
					}
				}
			} `graphql:"... on Commit"`
		}
	}

	var branchQuery struct {
		Repository struct {
			Refs struct {
				TotalCount int
				Nodes      []ref
			} `graphql:"refs(refPrefix: \"refs/heads/\", first: 100)"`
		} `graphql:"repository(name: $repoName, owner: $owner)"`
	}

	for _, repo := range repos {
		// branch list
		ghs.logger.Sugar().Debugf("Repo Name: %v", *repo.Name)
		defaultBranch := *repo.DefaultBranch

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

		ghs.mb.RecordGhRepoBranchesCountDataPoint(now, int64(numOfBranches), *repo.Name, ghs.cfg.GitHubOrg)

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
										Edges []commit
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
					ghs.logger.Sugar().Errorf("Branch Name: %v", branch.Name)
					ghs.logger.Sugar().Errorf("Owner: %v", ghs.cfg.GitHubOrg)
					ghs.logger.Sugar().Errorf("Repo Name: %v", *repo.Name)
				}

				oldestCommit, err := time.Parse(time.RFC3339, oldestCommitQuery.Repository.Ref.Target.Commit.History.Edges[0].Node.CommittedDate)

				if err != nil {
					ghs.logger.Sugar().Errorf("Error converting timestamp for oldest commit", zap.Error(err))
				}

				ghs.logger.Sugar().Debugf("Branch Age: %v", time.Now().Sub(oldestCommit).Hours())

			}

		}
	}

	ghs.logger.Sugar().Debugf("metrics: %v", ghs.cfg.Metrics.GhRepoCount)
	return ghs.mb.Emit(), nil
}
