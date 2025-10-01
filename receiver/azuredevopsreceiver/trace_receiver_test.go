// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
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

func TestEventToTraces(t *testing.T) {
	tests := []struct {
		desc          string
		eventLoader   func(t *testing.T) interface{}
		expectedError error
		expectedSpans int
	}{
		{
			desc: "PipelineRunStateChangedEvent processing",
			eventLoader: func(t *testing.T) interface{} {
				return loadPipelineRunEvent(t)
			},
			expectedError: nil,
			expectedSpans: 1,
		},
	}

	logger := zaptest.NewLogger(t)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Create a properly initialized receiver for testing
			cfg := &Config{
				WebHook: WebHook{
					ServerConfig: confighttp.ServerConfig{
						Endpoint: "localhost:8080",
					},
				},
			}
			atr := &azuredevopsTracesReceiver{
				logger: logger,
				cfg:    cfg,
			}
			event := test.eventLoader(t)
			traces, err := atr.handlePipelineEvent(event.(*PipelineRunStateChangedEvent))

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

func TestHandleStageEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &Config{
		WebHook: WebHook{
			ServerConfig: confighttp.ServerConfig{
				Endpoint: "localhost:8080",
			},
		},
	}

	event := loadStageEvent(t)
	atr := &azuredevopsTracesReceiver{
		logger: logger,
		cfg:    cfg,
	}

	traces, err := atr.handleStageEvent(event)
	require.NoError(t, err)
	require.Equal(t, 1, traces.SpanCount())
}

func TestHandleJobEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := &Config{
		WebHook: WebHook{
			ServerConfig: confighttp.ServerConfig{
				Endpoint: "localhost:8080",
			},
		},
	}

	event := loadJobEvent(t)
	atr := &azuredevopsTracesReceiver{
		logger: logger,
		cfg:    cfg,
	}

	traces, err := atr.handleJobEvent(event)
	require.NoError(t, err)
	require.Equal(t, 1, traces.SpanCount())
}

func TestEventHandlingWithRealData(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		desc          string
		eventLoader   func(t *testing.T) interface{}
		expectedSpans int
	}{
		{
			desc: "Pipeline event creates span",
			eventLoader: func(t *testing.T) interface{} {
				return loadPipelineRunEvent(t)
			},
			expectedSpans: 1,
		},
		{
			desc: "Stage event creates span",
			eventLoader: func(t *testing.T) interface{} {
				return loadStageEvent(t)
			},
			expectedSpans: 1,
		},
		{
			desc: "Job event creates span",
			eventLoader: func(t *testing.T) interface{} {
				return loadJobEvent(t)
			},
			expectedSpans: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			cfg := &Config{
				WebHook: WebHook{
					ServerConfig: confighttp.ServerConfig{
						Endpoint: "localhost:8080",
					},
				},
			}

			atr := &azuredevopsTracesReceiver{
				logger: logger,
				cfg:    cfg,
			}

			event := test.eventLoader(t)
			var traces ptrace.Traces
			var err error

			switch e := event.(type) {
			case *PipelineRunStateChangedEvent:
				traces, err = atr.handlePipelineEvent(e)
			case *PipelineStageStateChangedEvent:
				traces, err = atr.handleStageEvent(e)
			case *PipelineJobStateChangedEvent:
				traces, err = atr.handleJobEvent(e)
			default:
				t.Fatalf("Unknown event type: %T", event)
			}

			require.NoError(t, err)
			require.Equal(t, test.expectedSpans, traces.SpanCount())
		})
	}
}
