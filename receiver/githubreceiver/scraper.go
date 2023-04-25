package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/go-github/v50/github"
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

func ghClient(ghs *ghScraper) (client *github.Client) {
	ghs.logger.Sugar().Info("Creating the github client")
	client = github.NewClient(nil)
	return
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

	for _, repo := range repos {
		// branch list
		branches, _, err := client.Repositories.ListBranches(ctx, ghs.cfg.GitHubOrg, *repo.Name, nil)
		if err != nil {
			ghs.logger.Sugar().Errorf("Error getting branches", zap.Error(err))
		}
		ghs.mb.RecordGhRepoBranchesCountDataPoint(now, int64(len(branches)), *repo.Name, ghs.cfg.GitHubOrg)

		defaultBranch := *repo.DefaultBranch

		// For each branch, get some additional metrics and associate them with the branch
		for _, branch := range branches {
			if *branch.Name != defaultBranch {
				cmts, _, err := client.Repositories.ListCommits(ctx, ghs.cfg.GitHubOrg, *repo.Name, &github.CommitsListOptions{SHA: *branch.Name})
				if err != nil {
					ghs.logger.Sugar().Errorf("Error getting commits", zap.Error(err))
				}

				numCommits := int64(len(cmts))
				ghs.mb.RecordGhRepoBranchCommitsCountDataPoint(now, numCommits, *repo.Name, ghs.cfg.GitHubOrg, *branch.Name)

				// TODO: this should be a goroutine in order to maximize efficiency
				// TODO: there's a could potential for leveraging the graphql API for github
				// Commenting this out for now given the above, leverages the diverged func
				// instead of the REST API
				//for _, cmt := range cmts {
				//    sha := cmt.SHA
				//    diverged, err := diverges(ctx, client, ghs.cfg.GitHubOrg, *repo.Name, defaultBranch, *sha)
				//    if err != nil {
				//        ghs.logger.Sugar().Errorf("error when checking if the commit has diverged from the default branch: %v", zap.Error(err))
				//    }

				//    if len(cmt.Parents) == 1 && !diverged {
				//        createdDate := cmt.Commit.Author.Date.Time
				//        ageDelta := now.AsTime().Sub(createdDate)
				//        ghs.mb.RecordGhRepoBranchTotalAgeDataPoint(now, int64(ageDelta.Hours()), *repo.Name, ghs.cfg.GitHubOrg, *branch.Name)
				//    }
				//}
			}
		}

		// contributor list
		contribs, _, err := client.Repositories.ListContributors(ctx, ghs.cfg.GitHubOrg, *repo.Name, nil)
		if err != nil {
			ghs.logger.Sugar().Errorf("Error getting contributors", zap.Error(err))
		}
		ghs.mb.RecordGhRepoContributorsCountDataPoint(now, int64(len(contribs)), *repo.Name, ghs.cfg.GitHubOrg)
		//ghs.mb.RecordGhRepoBranchesCountDataPoint(now, int64(repo.Bran))
	}
	ghs.logger.Sugar().Debugf("repos: %v", repos)

	ghs.logger.Sugar().Debugf("metrics: %v", ghs.cfg.Metrics.GhRepoCount)
	return ghs.mb.Emit(), nil
}
