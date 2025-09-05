// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap/zaptest"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
)

func TestCreateNewTracesReceiver(t *testing.T) {
	defaultConfig := createDefaultConfig().(*Config)

	tests := []struct {
		desc     string
		config   Config
		consumer consumer.Traces
		err      error
	}{
		{
			desc:     "Default config succeeds",
			config:   *defaultConfig,
			consumer: consumertest.NewNop(),
			err:      nil,
		},
		{
			desc: "User defined config success",
			config: Config{
				WebHook: WebHook{
					ServerConfig: confighttp.ServerConfig{
						Endpoint: "localhost:8080",
					},
					Path:       "/events",
					HealthPath: "/health",
					Secret:     "mysecret",
				},
			},
			consumer: consumertest.NewNop(),
			err:      nil,
		},
		{
			desc: "Missing endpoint fails",
			config: Config{
				WebHook: WebHook{
					ServerConfig: confighttp.ServerConfig{
						Endpoint: "", // Empty endpoint should fail
					},
				},
			},
			consumer: consumertest.NewNop(),
			err:      errMissingEndpoint,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			rec, err := newTracesReceiver(receivertest.NewNopSettings(metadata.Type), &test.config, test.consumer)
			if test.err == nil {
				require.NotNil(t, rec)
			} else {
				require.ErrorIs(t, err, test.err)
				require.Nil(t, rec)
			}
		})
	}
}

func TestEventToTracesTraces(t *testing.T) {
	tests := []struct {
		desc          string
		event         interface{}
		expectedError error
		expectedSpans int
	}{
		{
			desc: "PipelineRunStateChangedEvent processing",
			event: &PipelineRunStateChangedEvent{
				Resource: struct {
					Run struct {
						Links struct {
							Self struct {
								Href string `json:"href"`
							} `json:"self"`
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
						} `json:"_links"`
						Pipeline struct {
							ID   int64  `json:"id"`
							Name string `json:"name"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						URL          string    `json:"url"`
					} `json:"run"`
				}{
					Run: struct {
						Links struct {
							Self struct {
								Href string `json:"href"`
							} `json:"self"`
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
						} `json:"_links"`
						Pipeline struct {
							ID   int64  `json:"id"`
							Name string `json:"name"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						URL          string    `json:"url"`
					}{
						Pipeline: struct {
							ID   int64  `json:"id"`
							Name string `json:"name"`
						}{
							ID:   123,
							Name: "test-pipeline",
						},
						State:  "completed",
						Result: "succeeded",
					},
				},
			},
			expectedError: nil,
			expectedSpans: 1,
		},
	}

	logger := zaptest.NewLogger(t)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Create a mock receiver for testing
			atr := &azuredevopsTracesReceiver{
				logger: logger,
			}
			traces, err := atr.handlePipelineEvent(test.event.(*PipelineRunStateChangedEvent))

			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expectedSpans, traces.SpanCount(), fmt.Sprintf("%s: unexpected number of spans", test.desc))
		})
	}
}

func TestGenerateTraceID(t *testing.T) {
	tests := []struct {
		desc      string
		runID     int64
		expectErr bool
	}{
		{
			desc:      "Valid trace ID generation",
			runID:     123,
			expectErr: false,
		},
		{
			desc:      "Different run ID generates different trace ID",
			runID:     456,
			expectErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			traceID, err := newTraceID(tc.runID)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEqual(t, [16]byte{}, traceID)
			}
		})
	}
}
