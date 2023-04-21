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
	// here is a problem maybe
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

	ghs.logger.Sugar().Debug("creating a new github client")
	client := github.NewClient(ghs.client)

	repos, _, err := client.Repositories.List(ctx, "adrielp", nil)
	if err != nil {
		ghs.logger.Sugar().Errorf("Error getting repos", zap.Error(err))
		//fmt.Print("failure in client repo list")
		//ghs.settings.Logger.Sugar().Error("Error getting repos", zap.Error(err))
	}

	//ghs.mb.RecordGhReposDataPoint(now, int64(10), "adrielp")
	ghs.mb.RecordGhRepoCountDataPoint(now, int64(len(repos)), "adrielp")

	ghs.logger.Sugar().Debugf("repos: %v", repos)

	ghs.logger.Sugar().Debugf("metrics: %v", ghs.cfg.Metrics.GhRepoCount)
	return ghs.mb.Emit(), nil
}
