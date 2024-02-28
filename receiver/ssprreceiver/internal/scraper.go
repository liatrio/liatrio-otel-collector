// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "sspr/internal"

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

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

type SsprJsonResponse struct {
	Error          bool   `json:"error"`
	ErrorCode      int    `json:"errorCode"`
	SuccessMessage string `json:"successMessage"`
	ErrorMessage   string `json:"errorMessage"`
	ErrorDetail    string `json:"errorDetail"`
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

func parseError(input map[string]interface{}) (int, float64, string) {
	errorFlag, ok := input["error"].(bool)
	if !ok {
		return 0, 0, "error key not found or not a boolean"
	}

	if errorFlag {
		errorCode := input["errorCode"].(float64)
		errorMessage, _ := input["errorMessage"].(string)
		return 1, errorCode, errorMessage
	}

	return 0, 0, "no error"
}

func parseStatistics(current interface{}, keyName string) (interface{}, error) {
	// Check if the current value is a []interface{}
	currentList, ok := current.([]interface{})
	if !ok {
		return nil, errors.New("expected []interface{} type for current")
	}

	// Iterate through the list of maps.
	for _, item := range currentList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, errors.New("expected map[string]interface{} type in the list")
		}
		name, ok := itemMap["name"].(string)
		if !ok {
			return nil, errors.New("missing or invalid 'name' field in the list item")
		}
		if name == keyName {
			return itemMap["value"], nil
		}
	}
	return nil, errors.New("key not found in the list")
}

func parseHealth(current interface{}, desiredKey string, searchString string) (int64, error) {
	// Check if the current value is a []interface{}
	currentList, ok := current.([]interface{})
	if !ok {
		return 0, errors.New("expected []interface{} type for current")
	}

	// Iterate through the list of maps.
	for _, item := range currentList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return 0, errors.New("expected map[string]interface{} type in the list")
		}

		// Extract the value associated with the desired key.
		keyValue, ok := itemMap[desiredKey]
		if !ok {
			return 0, errors.New("desired key not found in the list item")
		}

		// Convert the value to a string if possible.
		detail, ok := keyValue.(string)
		if !ok {
			return 0, errors.New("desired key value is not a string")
		}

		// Check if the detail contains the given searchString.
		if strings.Contains(detail, searchString) {
			return 1, nil
		} else {
			return 0, errors.New("searchString not found")
		}
	}

	// If the searchString is not found in any detail, return 0.
	return 0, nil
}

func (s *scraper) getValueAtPath(fullJsonMap interface{}, pathToListOfMaps []string, desiredKey string, searchString string) (interface{}, error) {
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

		switch key {
		case "current":
			// Handle the "current" case using parseStatistics function
			v, ok := value.([]interface{})
			if !ok {
				return nil, errors.New("unexpected type for 'current' switch case")
			}
			return parseStatistics(v, desiredKey)
		case "records":
			// Handle the "records" case using parseHealth function
			v, ok := value.([]interface{})
			if !ok {
				return nil, errors.New("unexpected type for 'records' switch case")
			}
			return parseHealth(v, desiredKey, searchString)
		default:
			// For other keys, handle as before
			switch v := value.(type) {
			case map[string]interface{}:
				// Recursively call the function with the subMap.
				return s.getValueAtPath(v, pathToListOfMaps[1:], desiredKey, searchString)
			default:
				// If the value is neither map nor list, return an error.
				return nil, errors.New("unexpected type")
			}
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
	// Begin Error Metric Creation Pattern Example
	errorState, errorCode, errorMessage := parseError(fullJsonPayload)
	if errorState == 1 {
		s.logger.Sugar().Errorln("Error: ", errorCode, "Error Message: ", errorMessage)
		// Place different codes here and build metrics accordingly
		if errorCode == 5053 {
			s.mb.RecordSsprConfigurationUnlockedDataPoint(pcommon.NewTimestampFromTime(time.Now()), 1)
		}
	} else {
		// If no error build the metric stating there was no error.
		s.mb.RecordSsprConfigurationUnlockedDataPoint(pcommon.NewTimestampFromTime(time.Now()), 0)
	}

	// End Error Metric Creation Pattern Example
	// Begin API Metric Creation Pattern Example

	// New Metric
	value, err := s.getValueAtPath(fullJsonPayload, []string{"data", "current"}, "DB_UNAVAILABLE_COUNT", "")
	if err != nil {
		s.logger.Sugar().Infoln("Value was not present at the given path. Continuing to next given key.")
	} else {
		intValue, err := strconv.ParseInt(value.(string), 10, 64)
		if err == nil {
			s.mb.RecordSsprDbUnavailableCountDataPoint(pcommon.NewTimestampFromTime(time.Now()), intValue)
		}
	}

	// New Metric
	configLockedMessage := "The application is unavailable or is restarting.  If this error occurs repeatedly please contact your help desk."
	value, err = s.getValueAtPath(fullJsonPayload, []string{"data", "records"}, "detail", configLockedMessage)
	if err != nil {
		s.logger.Sugar().Infoln("Value was not present at the given path. Continuing to next given key.")
	}
	if err == nil {
		s.mb.RecordSsprConfigurationUnlockedDataPoint(pcommon.NewTimestampFromTime(time.Now()), value.(int64))
	}

	// End Metric Creation Pattern Example

	return s.mb.Emit(), nil
}
