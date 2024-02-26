// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"testing"

	"github.com/alecthomas/assert/v2"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

var creationSet = receivertest.NewNopCreateSettings()

type statisticsResponse struct {
	status_code int    `json:"status_code"`
	version     string `json:"version"`
}

func MockServer() *http.ServeMux {
	var mux http.ServeMux
	mux.HandleFunc("/statistics", func(w http.ResponseWriter, r *http.Request) {
		res := statisticsResponse{status_code: 200, version: "1.0.0"}
		resByte, err := json.Marshal(res)

		if err != nil {
			log.Fatal(err.Error())
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resByte)
	})
	return &mux
}

func TestParseJSON(t *testing.T) {
	fields := map[string]string{
		"status_code": "$.status_code",
		"version":     "$.version",
	}
	testData := []byte(`{
		"status_code": 200,
		"version": "1.0.0"
	}`)

	parsedJson := parseJSON(testData, fields)

	statusCode, ok := parsedJson["status_code"]
	if ok {
		assert.Equal(t, "200", statusCode)
	} else {
		t.Error(`Key "status_code" does not exist in map.`)
	}

	version, ok := parsedJson["version"]
	if ok {
		assert.Equal(t, "1.0.0", version)
	} else {
		t.Error(`Key "version" does not exist in map.`)
	}
}

func TestScraperFactory_CreateMetricsScraper(t *testing.T) {
	factory := ScraperFactory{}
	ctx := context.Background()
	cfg := &ScraperConfig{}

	scraper, err := factory.CreateMetricsScraper(ctx, creationSet, *cfg)
	assert.NoError(t, err)
	assert.NoError(t, scraper.Start(ctx, componenttest.NewNopHost()))
}
