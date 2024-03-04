// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "sspr/internal"

import (
	"context"
	"crypto/tls"
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

type HTTPResponse struct {
	Status           string
	StatusCode       int
	Proto            string
	ProtoMajor       int
	ProtoMinor       int
	Header           http.Header
	Body             io.ReadCloser
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Uncompressed     bool
	Trailer          http.Header
	Request          *http.Request
	TLS              *tls.ConnectionState
}

type SsprBody struct {
	Error          bool   `json:"error"`
	ErrorCode      int    `json:"errorCode,omitempty"`
	SuccessMessage string `json:"successMessage,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	ErrorDetail    string `json:"errorDetail,omitempty"`
	Data           Data   `json:"data,omitempty"`
}

type Data interface {
	GetData() interface{}
}

func (data *RecordsList) GetData() interface{} {
	return data
}

func (data *CurrentList) GetData() interface{} {
	return data
}

type RecordsList struct {
	RecordData []*RecordData
}

type CurrentList struct {
	CurrentData []*CurrentData
}

type RecordData struct {
	Status string `json:"status"`
	Topic  string `json:"topic"`
	Detail string `json:"detail"`
}

type CurrentData struct {
	Description string `json:"description"`
	Labels      []struct {
		Description string `json:"description"`
		Name        string `json:"name"`
		Label       string `json:"label"`
		Type        string `json:"type"`
	} `json:"labels"`
	EventRates []struct {
		Description string  `json:"description"`
		Name        string  `json:"name"`
		Value       float64 `json:"value"`
	} `json:"eventRates"`
	Current []struct {
		Description string  `json:"description"`
		Name        string  `json:"name"`
		Value       float64 `json:"value"`
	} `json:"current"`
	Cumulative []struct {
		Description string  `json:"description"`
		Name        string  `json:"name"`
		Value       float64 `json:"value"`
	} `json:"cumulative"`
	History []struct {
		Description string `json:"description"`
		Name        string `json:"name"`
		Date        string `json:"date"`
		Year        int    `json:"year"`
		Month       int    `json:"month"`
		Day         int    `json:"day"`
		DaysAgo     int    `json:"daysAgo"`
		Data        []struct {
			Description string  `json:"description"`
			Name        string  `json:"name"`
			Value       float64 `json:"value"`
		} `json:"data"`
	} `json:"history"`
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
	res      *HTTPResponse
	payload  *SsprBody
}

func (s *scraper) LoadHTTPResponse(resp *http.Response) {
	s.res = &HTTPResponse{
		Status:           resp.Status,
		StatusCode:       resp.StatusCode,
		Proto:            resp.Proto,
		ProtoMajor:       resp.ProtoMajor,
		ProtoMinor:       resp.ProtoMinor,
		Header:           resp.Header,
		Body:             resp.Body,
		ContentLength:    resp.ContentLength,
		TransferEncoding: resp.TransferEncoding,
		Close:            resp.Close,
		Uncompressed:     resp.Uncompressed,
		Trailer:          resp.Trailer,
		Request:          resp.Request,
		TLS:              resp.TLS,
	}
}

func (s *scraper) parseJSON(data []byte) error {
	err := json.Unmarshal(data, &s.payload.Data)
	if err != nil {
		s.logger.Sugar().Errorln("[ERROR] Unable to unmarshal JSON payload.")
		return err
	}

	return nil
}

// func (res *SsprBody) parseError() (int, float64, string) {
// 	if res.Error {
// 		return 1, float64(res.ErrorCode), res.ErrorMessage
// 	}

// 	return 0, 0, "no error"
// }

// func (res *SsprBody) parseHealth(searchString string) (int, error) {
// 	records := res.Data.GetPayload().([]RecordData)
// 	if len(records) == 0 {
// 		return 0, errors.New("records list is empty")
// 	}

// 	// Iterate through the records list.
// 	for _, record := range records {
// 		if strings.Contains(record.Detail, searchString) {
// 			return 1, nil
// 		}
// 	}
// 	// If the searchString is not found in any record detail, return false.
// 	return 0, nil
// }

func (res *SsprBody) parseStatistics(keyName string) (interface{}, error) {
	payload, ok := res.Data.GetData().(map[string]interface{})
	if !ok {
		return nil, errors.New("unexpected type for payload")
	}

	for _, dataMap := range payload {
		// Check if the 'current' key exists in the dataMap
		dataMap, ok := dataMap.(map[string]interface{})
		if !ok {
			return nil, errors.New("unexpected type for dataMap")
		}

		if currentData, ok := dataMap["current"]; ok {
			currentDataSlice, ok := currentData.([]*CurrentData)
			if !ok {
				return nil, errors.New("unexpected type for 'current' key")
			}
			// Iterate over the 'current' data
			for _, dataCurrent := range currentDataSlice {
				// Iterate over the 'current' entries
				for _, currentEntry := range dataCurrent.Current {
					// Check if the 'Name' matches the keyName
					if currentEntry.Name == keyName {
						return currentEntry.Value, nil
					}
				}
			}
		} else if recordsData, ok := dataMap["records"]; ok {
			recordsDataSlice, ok := recordsData.([]*RecordData)
			if !ok {
				return nil, errors.New("unexpected type for 'records' key")
			}
			// Iterate over the 'records' data
			for _, recordData := range recordsDataSlice {
				// Check if the 'Topic' matches the keyName
				if recordData.Topic == keyName {
					return recordData.Status, nil
				}
			}
		}
	}

	// If the key is not found in the payload, return an error
	return nil, errors.New("key not found in the list")
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
	// defer res.Body.Close()

	s.LoadHTTPResponse(res)

	data, err := io.ReadAll(s.res.Body)
	if err != nil {
		s.logger.Sugar().Errorln("Unable to read http response: ", err, "response is: ", res.Body)
		return s.mb.Emit(), nil
	}

	s.parseJSON(data)
	// Begin Error Metric Creation Pattern Example

	if s.payload.ErrorCode == 1 {
		s.logger.Sugar().Errorln("Error: ", s.payload.ErrorCode, "Error Message: ", s.payload.ErrorMessage)
		// Place different codes here and build metrics accordingly
		if s.payload.ErrorCode == 5053 {
			s.mb.RecordSsprConfigurationUnlockedDataPoint(pcommon.NewTimestampFromTime(time.Now()), 1)
		}
	} else {
		// If no error build the metric stating there was no error.
		s.mb.RecordSsprConfigurationUnlockedDataPoint(pcommon.NewTimestampFromTime(time.Now()), 0)
	}

	// End Error Metric Creation Pattern Example
	// Begin API Metric Creation Pattern Example

	// New Metric
	// value, err := s.getValueAtPath([]string{"data", "current"}, "DB_UNAVAILABLE_COUNT", "")
	// value, err := s.payload.parseStatistics("DB_UNAVAILABLE_COUNT")
	// if err != nil {
	// 	s.logger.Sugar().Infoln("Value was not present at the given path. Continuing to next given key.")
	// } else {
	value, err := s.payload.parseStatistics("DB_UNAVAILABLE_COUNT")
	if err != nil {
		s.logger.Sugar().Errorln("Unable to parse statistics: ", err)
	}
	intValue, err := strconv.ParseInt(value.(string), 10, 64)
	if err == nil {
		s.mb.RecordSsprDbUnavailableCountDataPoint(pcommon.NewTimestampFromTime(time.Now()), intValue)
	}

	// End Metric Creation Pattern Example

	return s.mb.Emit(), nil
}
