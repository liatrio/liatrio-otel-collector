// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "sspr/internal"

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"net/http"
	"time"

	"github.com/liatrio/liatrio-otel-collector/receiver/ssprreceiver/internal/metadata"
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
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, params), // Not loading Enabled default value?
	}

	return scraperhelper.NewScraper(
		"ssprScraper",
		s.scrape,
		scraperhelper.WithStart(s.start),
	)
}

type ScraperConfig struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	confighttp.HTTPClientSettings           `mapstructure:",squash"`
	metadata.MetricsBuilderConfig           `mapstructure:",squash"`
	Endpoint                                string `mapstructure:"endpoint"`
}

type scraper struct {
	client   *http.Client
	cfg      *ScraperConfig
	logger   *zap.Logger
	settings component.TelemetrySettings
	mb       *metadata.MetricsBuilder
}

func (s *scraper) parseJSON(data []byte) map[string]any {
	unMarshalledJson := interface{}(nil)
	err := json.Unmarshal(data, &unMarshalledJson)
	if err != nil {
		s.logger.Sugar().Errorln("[ERROR] Unable to unmarshal JSON payload.")
	}

	return unMarshalledJson.(map[string]any)
}

func getValueAtPath(fullJsonMap interface{}, pathToListOfMaps []string, keyName string) (interface{}, error) {
	if len(pathToListOfMaps) == 0 {
		return nil, errors.New("path is empty")
	}

	current, ok := fullJsonMap.(map[string]interface{})
	if !ok {
		return nil, errors.New("expected map[string]interface{} type")
	}

	for _, key := range pathToListOfMaps {
		value, ok := current[key]
		if !ok {
			return nil, errors.New("path not found")
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively call the function with the subMap.
			return getValueAtPath(v, pathToListOfMaps[1:], keyName)
		case []interface{}:
			// Iterate through the list of maps.
			for _, item := range v {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					return nil, errors.New("expected map[string]interface{} type in the list")
				}
				name, ok := itemMap["name"].(string)
				if !ok {
					return nil, errors.New("missing or invalid 'name' field in the list item")
				}
				if name == keyName {
					// Return the value if the name matches.
					return itemMap["value"], nil
				}
			}
			// If the key is not found in the list, return an error.
			return nil, errors.New("key not found in the list")
		default:
			// If the value is neither map nor list, return an error.
			return nil, errors.New("unexpected type")
		}
	}

	return nil, errors.New("key not found in the target map")
}

func (s *scraper) start(_ context.Context, host component.Host) (err error) {
	s.logger.Sugar().Info("starting the sspr scraper")
	s.client, err = s.cfg.ToClient(host, s.settings)
	if err != nil {
		return errClientNotInitErr
	}
	return
}

func (s *scraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.cfg.Endpoint, http.NoBody)
	req.Header.Add("Accept", "application/json")

	if err != nil {
		s.logger.Sugar().Errorln("Unable to create new http request: ", err)
		return s.mb.Emit(), nil
	}

	res, err := s.client.Do(req)
	if err != nil {
		s.logger.Sugar().Errorln("Unable to execute http request: ", err)
		return s.mb.Emit(), nil
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		s.logger.Sugar().Errorln("Unable to read http response: ", err, "response is: ", res.Body)
		return s.mb.Emit(), nil
	}

	fullJsonPayload := s.parseJSON(data)

	// Begin Metric Creation Pattern Example
	value, err := getValueAtPath(fullJsonPayload, []string{"data", "current"}, "DB_UNAVAILABLE_COUNT")
	if err != nil {
		s.logger.Sugar().Errorln("Error collecting value at given path.")
	}
	intValue, err := strconv.ParseInt(value.(string), 10, 64)
	if err == nil {
		s.mb.RecordSsprDbUnavailableCountDataPoint(pcommon.NewTimestampFromTime(time.Now()), intValue)
	} else {
		s.logger.Sugar().Errorln("Error converting value from string to Int.")
	}
	// End Metric Creation Pattern Example

	return s.mb.Emit(), nil
}
