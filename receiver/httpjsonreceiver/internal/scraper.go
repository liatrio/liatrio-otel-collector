// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "httpjson/internal"

import (
	"context"
	"errors"
	"io"

	"encoding/json"
	"net/http"
	"time"

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
}

type scraper struct {
	client        *http.Client
	cfg           *ScraperConfig
	logger        *zap.Logger
	settings      component.TelemetrySettings
	metricsBuffer pmetric.Metrics
}

func (s *scraper) start(_ context.Context, host component.Host) error {
	s.logger.Sugar().Info("starting the httpjson scraper")
	if s.client == nil {
		return errClientNotInitErr
	}
	return nil
}

func (s *scraper) scrape(_ context.Context) (pmetric.Metrics, error) {

	if s.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	s.logger.Sugar().Info("running the httpjson scrape function")

	now := pcommon.NewTimestampFromTime(time.Now())
	s.logger.Sugar().Debugf("current time: %v", now)

	currentDate := time.Now().Day()
	s.logger.Sugar().Debugf("current date: %v", currentDate)

	// Create a request using the HTTP GET method
	res, err := s.client.Get(s.cfg.HTTPClientSettings.Endpoint)
	if err != nil {
		s.logger.Sugar().DPanicln("[ERROR]: Unable to make GET request.")
	}

	// TODO: Handle response body here
	// Two cases: Targeted Fields or Catch-All funnel the rest as Metrics.
	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.logger.Sugar().DPanicln("[ERROR]: Unable to read request from body")
		// Handle error reading response body
	}

	var jsonData map[string]interface{} // Or define a custom struct based on your JSON schema
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		s.logger.Sugar().DPanicln("[ERROR]: Unable to unmarshall JSON data.")
	}

	for key, value := range jsonData["data"].(map[string]interface{}) {
		switch key {
		case "labels":
			// Extract information from labels array
			for _, label := range value.([]interface{}) {
				labelMap := label.(map[string]interface{})
				metricName := labelMap["name"].(string)
				metricType := labelMap["type"].(string)
				description := labelMap["description"].(string)

				// Create metric based on type (assuming known types)
				switch metricType {
				case "INCREMENTER":
					metric := s.metricsBuffer.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
					metric.SetName(metricName)
					metric.SetDescription(description)
					metric.SetEmptySum()

				default:
					// Log warning or handle unknown types
					s.logger.Sugar().Warnf("Unknown metric type: %s", metricType)
				}
			}
		// case "eventRates":
		// 	// Extract information from eventRates array
		// 	for _, eventRate := range value.([]interface{}) {
		// 		eventRateMap := eventRate.(map[string]interface{})
		// 		metricName := eventRateMap["name"].(string)
		// 		value := eventRateMap["value"].(string)
		// 		// Assuming value is a float64, convert and create gauge metric
		// 		metricValue, err := strconv.ParseFloat(value, 64)
		// 		if err != nil {
		// 			s.logger.Sugar().Warnf("Error parsing eventRate value: %s", err)
		// 			continue
		// 		}
		// 		metric := pmetric.NewMetric()
		// 		metric.SetName(metricName)
		// 		metric.Set
		// 		metrics.AddMetric(metric)
		// 	}
		// case "history":
		// 	// Loop through history entries and extract data points
		// 	for _, historyEntry := range value.([]interface{}) {
		// 		historyEntryMap := historyEntry.(map[string]interface{})
		// 		dateStr := historyEntryMap["date"].(string)
		// 		// Assuming date is in YYYY-MM-DD format, parse it
		// 		date, err := time.Parse("2006-01-02", dateStr)
		// 		if err != nil {
		// 			s.logger.Sugar().Warnf("Error parsing history entry date: %s", err)
		// 			continue
		// 		}
		// 		for _, dataPoint := range historyEntryMap["data"].([]interface{}) {
		// 			dataPointMap := dataPoint.(map[string]interface{})
		// 			metricName := dataPointMap["name"].(string)
		// 			valueStr := dataPointMap["value"].(string)
		// 			// Assuming value is an int64, convert and create sum metric
		// 			metricValue, err := strconv.ParseInt(valueStr, 10, 64)
		// 			if err != nil {
		// 				s.logger.Sugar().Warnf("Error parsing history entry value: %s", err)
		// 				continue
		// 			}
		// 			metric := pmetric.NewMetric(
		// 				pmetric.MetricID(metricName),
		// 				pmetric.MetricDataTypeInt64,
		// 				pcommon.NewTimestampFromTime(date),
		// 				metricValue,
		// 			)
		// 			metric.SetName(metricName)
		// 			metrics.AddMetric(metric)
		// 		}
		// 	}
		default:
			// Handle other keys in "data" if needed
		}
	}

	return s.metricsBuffer, nil
}
