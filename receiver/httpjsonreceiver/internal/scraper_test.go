// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
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
	testData := statisticsResponse{status_code: 200, version: "1.0.0"}
	marshaledData, err := json.Marshal(testData)
	if err != nil {
		t.Error("Unable to marshal testData")
	}

	parsedJson, err := parseJSON(marshaledData)

	// name := "Gladys"
	// want := regexp.MustCompile(`\b`+name+`\b`)
	// msg, err := Hello("Gladys")
	// if !want.MatchString(msg) || err != nil {
	//     t.Fatalf(`Hello("Gladys") = %q, %v, want match for %#q, nil`, msg, err, want)
	// }
}

func TestScraperFactory_CreateMetricsScraper(t *testing.T) {
	factory := ScraperFactory{}
	ctx := context.Background()
	cfg := &ScraperConfig{}

	scraper, err := factory.CreateMetricsScraper(ctx, creationSet, *cfg)
	assert.NoError(t, err)

	assert.NoError(t, scraper.Start(ctx, componenttest.NewNopHost()))

	_, err = scraper.Scrape(ctx)
	assert.NoError(t, err)
}

func TestScaperScrape(t *testing.T) {
	testCases := []struct {
		desc              string
		expectedMetricGen func(t *testing.T) pmetric.Metrics
		endpoint          string
		compareOptions    []pmetrictest.CompareMetricsOption
	}{
		{
			desc:             "Successful Collection",
			expectedResponse: 200,
			expectedMetricGen: func(t *testing.T) pmetric.Metrics {
				goldenPath := filepath.Join("testdata", "golden.yaml")
				expectedMetrics, err := golden.ReadMetrics(goldenPath)
				require.NoError(t, err)
				return expectedMetrics
			},
			expectedErr: nil,
			compareOptions: []pmetrictest.CompareMetricsOption{
				pmetrictest.IgnoreMetricAttributeValue("http.url"),
				pmetrictest.IgnoreMetricValues("httpjson.duration"),
				pmetrictest.IgnoreMetricDataPointsOrder(),
				pmetrictest.IgnoreStartTimestamp(),
				pmetrictest.IgnoreTimestamp(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			factory := ScraperFactory{}
			ctx := context.Background()
			cfg := &ScraperConfig{}
			scraper, err := factory.CreateMetricsScraper(ctx, creationSet, *cfg)
			// cfg := createDefaultConfig().(*Config)
			// if len(tc.endpoint) > 0 {
			// 	cfg.Targets = []*targetConfig{{
			// 		ClientConfig: confighttp.ClientConfig{
			// 			Endpoint: tc.endpoint,
			// 		}},
			// 	}
			// } else {
			// 	ms := newMockServer(t, tc.expectedResponse)
			// 	defer ms.Close()
			// 	cfg.Targets = []*targetConfig{{
			// 		ClientConfig: confighttp.ClientConfig{
			// 			Endpoint: ms.URL,
			// 		}},
			// 	}
			// }
			// scraper := newScraper(cfg, receivertest.NewNopCreateSettings())
			// require.NoError(t, scraper.start(context.Background(), componenttest.NewNopHost()))
			ms :=
				scraper.Scrape(ctx)

			actualMetrics, err := scraper.scrape(context.Background())
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedErr.Error())
			}

			expectedMetrics := tc.expectedMetricGen(t)

			require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, tc.compareOptions...))
		})
	}
}
