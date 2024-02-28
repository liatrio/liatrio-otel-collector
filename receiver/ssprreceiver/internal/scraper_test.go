// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"testing"

	"github.com/alecthomas/assert/v2"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

var creationSet = receivertest.NewNopCreateSettings()

type statisticsResponse struct {
	Status_code int    `json:"status_code"`
	Version     string `json:"version"`
}

func MockServer() *http.ServeMux {
	var mux http.ServeMux
	mux.HandleFunc("/statistics", func(w http.ResponseWriter, r *http.Request) {
		res := statisticsResponse{Status_code: 200, Version: "1.0.0"}
		resByte, err := json.Marshal(res)

		if err != nil {
			log.Fatal(err.Error())
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resByte)
	})
	return &mux
}

func (s *scraper) TestParseJSON(t *testing.T) {
	testData := []byte(`{
		"status_code": 200,
		"version": "1.0.0"
	}`)

	parsedJson := s.parseJSON(testData)

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

func TestParseError(t *testing.T) {
	testCases := []struct {
		input        map[string]interface{} // Input data for the test case
		expectedFlag int                    // Expected error flag
		expectedCode float64                // Expected error code (remains float64)
		expectedMsg  string                 // Expected error message
	}{
		// Test case 1: No error
		{
			input: map[string]interface{}{
				"error": false,
			},
			expectedFlag: 0,
			expectedCode: 0,
			expectedMsg:  "no error",
		},
		// Test case 2: Error with error code and message
		{
			input: map[string]interface{}{
				"error":        true,
				"errorCode":    404.1,
				"errorMessage": "Not Found",
			},
			expectedFlag: 1,
			expectedCode: 404.1,
			expectedMsg:  "Not Found",
		},
		// Test case 3: Error Config is Locked
		{
			input: map[string]interface{}{
				"error":        true,
				"errorCode":    5053.0,
				"errorMessage": "The application is unavailable or is restarting.  If this error occurs repeatedly please contact your help desk.",
			},
			expectedFlag: 1,
			expectedCode: 5053.0,
			expectedMsg:  "The application is unavailable or is restarting.  If this error occurs repeatedly please contact your help desk.",
		},
	}

	for _, tc := range testCases {
		flag, code, msg := parseError(tc.input)

		assert.Equal(t, tc.expectedFlag, flag, "Error flag mismatch")
		assert.Equal(t, tc.expectedCode, code, "Error code mismatch")
		assert.Equal(t, tc.expectedMsg, msg, "Error message mismatch")
	}
}

func TestParseStatistics(t *testing.T) {
	// Test data
	testData := []struct {
		input    interface{} // Input data for the test case
		keyName  string      // Name of the key to search for
		expected interface{} // Expected value associated with the key
		isError  bool        // Indicates whether the test case should result in an error
	}{
		// Test case 1: Valid input, key exists
		{
			input: []interface{}{
				map[string]interface{}{"name": "key1", "value": 100},
				map[string]interface{}{"name": "key2", "value": 200},
			},
			keyName:  "key1",
			expected: 100,
			isError:  false,
		},
		// Test case 2: Valid input, key does not exist
		{
			input:    []interface{}{},
			keyName:  "key3",
			expected: nil,
			isError:  true,
		},
		// Test case 3: Invalid input (not a list)
		{
			input:    map[string]interface{}{},
			keyName:  "key1",
			expected: nil,
			isError:  true,
		},
	}

	// Run test cases
	for index, tc := range testData {
		result, err := parseStatistics(tc.input, tc.keyName)

		// Check for error if expected
		if tc.isError {
			assert.Error(t, err, "Test case %d: Expected error but got none", index+1)
			assert.Equal(t, nil, result, "Test case %d: Result should be nil", index+1)
		} else {
			// Check for equality of results if no error expected
			assert.NoError(t, err, "Test case %d: Unexpected error", index+1)
			assert.Equal(t, tc.expected, result, "Test case %d: Result mismatch", index+1)
		}
	}
}

func TestParseHealth(t *testing.T) {
	// Test data
	testCases := []struct {
		input        interface{} // Input data for the test case
		desiredKey   string      // Desired key to search for
		searchString string      // Search string to match
		expected     int64       // Expected result
		expectedErr  error       // Expected error
	}{
		// Test case 1: Valid input, searchString found
		{
			input: []interface{}{
				map[string]interface{}{"status": "healthy"},
				map[string]interface{}{"status": "unhealthy"},
			},
			desiredKey:   "status",
			searchString: "healthy",
			expected:     1,
			expectedErr:  nil,
		},
		// Test case 2: Valid input, searchString not found
		{
			input: []interface{}{
				map[string]interface{}{"status": "unhealthy"},
				map[string]interface{}{"status": "error"},
			},
			desiredKey:   "status",
			searchString: "healthy",
			expected:     1,
			expectedErr:  nil,
		},
		// Test case 3: Invalid input (not a list)
		{
			input:        map[string]interface{}{},
			desiredKey:   "status",
			searchString: "healthy",
			expected:     0,
			expectedErr:  errors.New("expected []interface{} type for current"),
		},
		// Test case 4: Desired key not found
		{
			input: []interface{}{
				map[string]interface{}{"name": "node1"},
				map[string]interface{}{"name": "node2"},
			},
			desiredKey:   "status",
			searchString: "healthy",
			expected:     0,
			expectedErr:  errors.New("desired key not found in the list item"),
		},
		// Test case 5: Desired key value is not a string
		{
			input: []interface{}{
				map[string]interface{}{"status": 1},
				map[string]interface{}{"status": true},
			},
			desiredKey:   "status",
			searchString: "healthy",
			expected:     0,
			expectedErr:  errors.New("desired key value is not a string"),
		},
	}

	// Run test cases
	for idx, tc := range testCases {
		result, err := parseHealth(tc.input, tc.desiredKey, tc.searchString)

		// Check for error if expected
		if tc.expectedErr != nil {
			assert.EqualError(t, err, tc.expectedErr.Error(), "Test case %d: Error mismatch", idx+1)
			assert.Equal(t, tc.expected, result, "Test case %d: Result mismatch", idx+1)
		} else {
			// Check for equality of results if no error expected
			assert.NoError(t, err, "Test case %d: Unexpected error", idx+1)
			assert.Equal(t, tc.expected, result, "Test case %d: Result mismatch", idx+1)
		}
	}
}
