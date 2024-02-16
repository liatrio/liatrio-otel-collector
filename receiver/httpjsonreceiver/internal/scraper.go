// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "httpjson/internal"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"net/http"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/liatrio/liatrio-otel-collector/receiver/httpjsonreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
)

var errClientNotInitErr = errors.New("http client not initialized")

type ScraperFactory struct{}

func (f *ScraperFactory) CreateMetricsScraper(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg ScraperConfig,
) (scraperhelper.Scraper, error) {
	s := &scraper{
		cfg:      &cfg,
		logger:   params.Logger,
		settings: params.TelemetrySettings,
	}

	return scraperhelper.NewScraper(
		"httpjsonScraper",
		s.scrape,
		scraperhelper.WithStart(s.start),
	)
}

type ScraperConfig struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	confighttp.HTTPClientSettings           `mapstructure:",squash"`
	Method                                  string            `mapstructure:",squash"`
	Fields                                  map[string]string `mapstructure:"fields"`
}

type scraper struct {
	client   *http.Client
	cfg      *ScraperConfig
	logger   *zap.Logger
	settings component.TelemetrySettings
	mb       *metadata.MetricsBuilder
}

func parseJSON(data []byte, fields map[string]string) map[string]any {
	metricsMap := make(map[string]any)
	tmp := interface{}(nil)
	json.Unmarshal(data, &tmp)

	for key, value := range fields {
		jv, err := jsonpath.Get(value, tmp)
		if err != nil {
			fmt.Println(err)
		}
		// forcing the value to become a string
		metricsMap[key] = fmt.Sprintf("%v", jv)
	}

	return metricsMap
}

func (s *scraper) start(_ context.Context, host component.Host) (err error) {
	s.logger.Sugar().Info("starting the httpjson scraper")
	s.client, err = s.cfg.ToClient(host, s.settings)
	if err != nil {
		return errClientNotInitErr
	}
	return
}

func (s *scraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	req, err := http.NewRequestWithContext(ctx, s.cfg.Method, s.cfg.Endpoint, http.NoBody)

	if err != nil {
		s.logger.Sugar().Errorln("Unable to create new http request")
		return s.mb.Emit(), nil
	}
	start := time.Now()
	res, err := s.client.Do(req)

	if err != nil {
		s.logger.Sugar().Errorln("Unable to execute http request")
		return s.mb.Emit(), nil
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		s.logger.Sugar().Errorln("Unable to read http response")
		return s.mb.Emit(), nil
	}

	metricsAttributeMap := parseJSON(data, s.cfg.Fields)
	s.mb.RecordHttpjsonDurationDataPoint(pcommon.NewTimestampFromTime(time.Now()), time.Since(start).Milliseconds(), s.cfg.Endpoint, int64(res.StatusCode), s.cfg.Method, metricsAttributeMap)

	return s.mb.Emit(), nil
}
