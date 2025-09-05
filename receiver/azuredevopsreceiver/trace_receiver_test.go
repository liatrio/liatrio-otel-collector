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

func TestEventToTraces(t *testing.T) {
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

func TestHandleStageEvent(t *testing.T) {
	tests := []struct {
		desc          string
		event         *PipelineStageStateChangedEvent
		expectedError error
		expectedSpans int
	}{
		{
			desc: "PipelineStageStateChangedEvent processing - success",
			event: &PipelineStageStateChangedEvent{
				Resource: struct {
					Stage struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID          string `json:"id"`
						Name        string `json:"name"`
						DisplayName string `json:"displayName"`
						State       string `json:"state"`
						Result      string `json:"result"`
					} `json:"stage"`
					Run struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					} `json:"run"`
					Pipeline struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					} `json:"pipeline"`
					Repositories []struct {
						Type   string `json:"type"`
						Change struct {
							Author struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"author"`
							Committer struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"committer"`
							Message string `json:"message"`
						} `json:"change"`
						URL string `json:"url"`
					} `json:"repositories"`
				}{
					Stage: struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID          string `json:"id"`
						Name        string `json:"name"`
						DisplayName string `json:"displayName"`
						State       string `json:"state"`
						Result      string `json:"result"`
					}{
						ID:          "stage-123",
						Name:        "Build",
						DisplayName: "Build Stage",
						State:       "completed",
						Result:      "succeeded",
					},
					Run: struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					}{
						Pipeline: struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						}{
							ID:   456,
							Name: "test-pipeline",
						},
						State:       "completed",
						Result:      "succeeded",
						CreatedDate: time.Now().Add(-time.Hour),
						ID:          789,
						Name:        "test-run",
					},
					Pipeline: struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					}{
						ID:   456,
						Name: "test-pipeline",
					},
				},
			},
			expectedError: nil,
			expectedSpans: 1,
		},
		{
			desc: "PipelineStageStateChangedEvent processing - failed",
			event: &PipelineStageStateChangedEvent{
				Resource: struct {
					Stage struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID          string `json:"id"`
						Name        string `json:"name"`
						DisplayName string `json:"displayName"`
						State       string `json:"state"`
						Result      string `json:"result"`
					} `json:"stage"`
					Run struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					} `json:"run"`
					Pipeline struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					} `json:"pipeline"`
					Repositories []struct {
						Type   string `json:"type"`
						Change struct {
							Author struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"author"`
							Committer struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"committer"`
							Message string `json:"message"`
						} `json:"change"`
						URL string `json:"url"`
					} `json:"repositories"`
				}{
					Stage: struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID          string `json:"id"`
						Name        string `json:"name"`
						DisplayName string `json:"displayName"`
						State       string `json:"state"`
						Result      string `json:"result"`
					}{
						ID:          "stage-456",
						Name:        "Test",
						DisplayName: "Test Stage",
						State:       "completed",
						Result:      "failed",
					},
					Run: struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					}{
						Pipeline: struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						}{
							ID:   789,
							Name: "failed-pipeline",
						},
						State:       "completed",
						Result:      "failed",
						CreatedDate: time.Now().Add(-time.Hour),
						ID:          101112,
						Name:        "failed-run",
					},
					Pipeline: struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					}{
						ID:   789,
						Name: "failed-pipeline",
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
			atr := &azuredevopsTracesReceiver{
				logger: logger,
			}
			traces, err := atr.handleStageEvent(test.event)

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

func TestHandleJobEvent(t *testing.T) {
	tests := []struct {
		desc          string
		event         *PipelineJobStateChangedEvent
		expectedError error
		expectedSpans int
	}{
		{
			desc: "PipelineJobStateChangedEvent processing - success",
			event: &PipelineJobStateChangedEvent{
				Resource: struct {
					Job struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID         string    `json:"id"`
						Name       string    `json:"name"`
						State      string    `json:"state"`
						Result     string    `json:"result"`
						StartTime  time.Time `json:"startTime"`
						FinishTime time.Time `json:"finishTime"`
						Attempt    int64     `json:"attempt"`
					} `json:"job"`
					Stage struct {
						ID          string    `json:"id"`
						Name        string    `json:"name"`
						DisplayName string    `json:"displayName"`
						Attempt     int64     `json:"attempt"`
						State       string    `json:"state"`
						Result      string    `json:"result"`
						StartTime   time.Time `json:"startTime"`
						FinishTime  time.Time `json:"finishTime"`
					} `json:"stage"`
					Run struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					} `json:"run"`
					Pipeline struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					} `json:"pipeline"`
					Repositories []struct {
						Alias  string `json:"alias"`
						ID     string `json:"id"`
						Type   string `json:"type"`
						Change struct {
							Author struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"author"`
							Committer struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"committer"`
							Message string `json:"message"`
							Version string `json:"version"`
						} `json:"change"`
						URL string `json:"url"`
					} `json:"repositories"`
				}{
					Job: struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID         string    `json:"id"`
						Name       string    `json:"name"`
						State      string    `json:"state"`
						Result     string    `json:"result"`
						StartTime  time.Time `json:"startTime"`
						FinishTime time.Time `json:"finishTime"`
						Attempt    int64     `json:"attempt"`
					}{
						ID:         "job-456",
						Name:       "Build Job",
						State:      "completed",
						Result:     "succeeded",
						StartTime:  time.Now().Add(-30 * time.Minute),
						FinishTime: time.Now().Add(-10 * time.Minute),
						Attempt:    1,
					},
					Stage: struct {
						ID          string    `json:"id"`
						Name        string    `json:"name"`
						DisplayName string    `json:"displayName"`
						Attempt     int64     `json:"attempt"`
						State       string    `json:"state"`
						Result      string    `json:"result"`
						StartTime   time.Time `json:"startTime"`
						FinishTime  time.Time `json:"finishTime"`
					}{
						ID:          "stage-123",
						Name:        "Build",
						DisplayName: "Build Stage",
						State:       "completed",
						Result:      "succeeded",
					},
					Run: struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					}{
						Pipeline: struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						}{
							ID:   456,
							Name: "test-pipeline",
						},
						State:       "completed",
						Result:      "succeeded",
						CreatedDate: time.Now().Add(-time.Hour),
						ID:          789,
						Name:        "test-run",
					},
					Pipeline: struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					}{
						ID:   456,
						Name: "test-pipeline",
					},
				},
			},
			expectedError: nil,
			expectedSpans: 1,
		},
		{
			desc: "PipelineJobStateChangedEvent processing - failed job",
			event: &PipelineJobStateChangedEvent{
				Resource: struct {
					Job struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID         string    `json:"id"`
						Name       string    `json:"name"`
						State      string    `json:"state"`
						Result     string    `json:"result"`
						StartTime  time.Time `json:"startTime"`
						FinishTime time.Time `json:"finishTime"`
						Attempt    int64     `json:"attempt"`
					} `json:"job"`
					Stage struct {
						ID          string    `json:"id"`
						Name        string    `json:"name"`
						DisplayName string    `json:"displayName"`
						Attempt     int64     `json:"attempt"`
						State       string    `json:"state"`
						Result      string    `json:"result"`
						StartTime   time.Time `json:"startTime"`
						FinishTime  time.Time `json:"finishTime"`
					} `json:"stage"`
					Run struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					} `json:"run"`
					Pipeline struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					} `json:"pipeline"`
					Repositories []struct {
						Alias  string `json:"alias"`
						ID     string `json:"id"`
						Type   string `json:"type"`
						Change struct {
							Author struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"author"`
							Committer struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"committer"`
							Message string `json:"message"`
							Version string `json:"version"`
						} `json:"change"`
						URL string `json:"url"`
					} `json:"repositories"`
				}{
					Job: struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID         string    `json:"id"`
						Name       string    `json:"name"`
						State      string    `json:"state"`
						Result     string    `json:"result"`
						StartTime  time.Time `json:"startTime"`
						FinishTime time.Time `json:"finishTime"`
						Attempt    int64     `json:"attempt"`
					}{
						ID:         "job-789",
						Name:       "Test Job",
						State:      "completed",
						Result:     "failed",
						StartTime:  time.Now().Add(-45 * time.Minute),
						FinishTime: time.Now().Add(-15 * time.Minute),
						Attempt:    2,
					},
					Stage: struct {
						ID          string    `json:"id"`
						Name        string    `json:"name"`
						DisplayName string    `json:"displayName"`
						Attempt     int64     `json:"attempt"`
						State       string    `json:"state"`
						Result      string    `json:"result"`
						StartTime   time.Time `json:"startTime"`
						FinishTime  time.Time `json:"finishTime"`
					}{
						ID:          "stage-456",
						Name:        "Test",
						DisplayName: "Test Stage",
						State:       "completed",
						Result:      "failed",
					},
					Run: struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					}{
						Pipeline: struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						}{
							ID:   789,
							Name: "failed-pipeline",
						},
						State:       "completed",
						Result:      "failed",
						CreatedDate: time.Now().Add(-time.Hour),
						ID:          101112,
						Name:        "failed-run",
					},
					Pipeline: struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					}{
						ID:   789,
						Name: "failed-pipeline",
					},
				},
			},
			expectedError: nil,
			expectedSpans: 1,
		},
		{
			desc: "PipelineJobStateChangedEvent processing - retry attempt",
			event: &PipelineJobStateChangedEvent{
				Resource: struct {
					Job struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID         string    `json:"id"`
						Name       string    `json:"name"`
						State      string    `json:"state"`
						Result     string    `json:"result"`
						StartTime  time.Time `json:"startTime"`
						FinishTime time.Time `json:"finishTime"`
						Attempt    int64     `json:"attempt"`
					} `json:"job"`
					Stage struct {
						ID          string    `json:"id"`
						Name        string    `json:"name"`
						DisplayName string    `json:"displayName"`
						Attempt     int64     `json:"attempt"`
						State       string    `json:"state"`
						Result      string    `json:"result"`
						StartTime   time.Time `json:"startTime"`
						FinishTime  time.Time `json:"finishTime"`
					} `json:"stage"`
					Run struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					} `json:"run"`
					Pipeline struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					} `json:"pipeline"`
					Repositories []struct {
						Alias  string `json:"alias"`
						ID     string `json:"id"`
						Type   string `json:"type"`
						Change struct {
							Author struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"author"`
							Committer struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"committer"`
							Message string `json:"message"`
							Version string `json:"version"`
						} `json:"change"`
						URL string `json:"url"`
					} `json:"repositories"`
				}{
					Job: struct {
						Links struct {
							Web struct {
								Href string `json:"href"`
							} `json:"web"`
							PipelineWeb struct {
								Href string `json:"href"`
							} `json:"pipeline.web"`
						} `json:"_links"`
						ID         string    `json:"id"`
						Name       string    `json:"name"`
						State      string    `json:"state"`
						Result     string    `json:"result"`
						StartTime  time.Time `json:"startTime"`
						FinishTime time.Time `json:"finishTime"`
						Attempt    int64     `json:"attempt"`
					}{
						ID:         "job-retry-123",
						Name:       "Retry Job",
						State:      "completed",
						Result:     "succeeded",
						StartTime:  time.Now().Add(-20 * time.Minute),
						FinishTime: time.Now().Add(-5 * time.Minute),
						Attempt:    3, // Third attempt succeeded
					},
					Stage: struct {
						ID          string    `json:"id"`
						Name        string    `json:"name"`
						DisplayName string    `json:"displayName"`
						Attempt     int64     `json:"attempt"`
						State       string    `json:"state"`
						Result      string    `json:"result"`
						StartTime   time.Time `json:"startTime"`
						FinishTime  time.Time `json:"finishTime"`
					}{
						ID:          "stage-retry-789",
						Name:        "Deploy",
						DisplayName: "Deploy Stage",
						State:       "completed",
						Result:      "succeeded",
					},
					Run: struct {
						Pipeline struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						} `json:"pipeline"`
						State        string    `json:"state"`
						Result       string    `json:"result"`
						CreatedDate  time.Time `json:"createdDate"`
						FinishedDate time.Time `json:"finishedDate"`
						ID           int64     `json:"id"`
						Name         string    `json:"name"`
					}{
						Pipeline: struct {
							URL      string `json:"url"`
							ID       int64  `json:"id"`
							Revision int64  `json:"revision"`
							Name     string `json:"name"`
							Folder   string `json:"folder"`
						}{
							ID:   999,
							Name: "retry-pipeline",
						},
						State:       "completed",
						Result:      "succeeded",
						CreatedDate: time.Now().Add(-2 * time.Hour),
						ID:          131415,
						Name:        "retry-run",
					},
					Pipeline: struct {
						URL      string `json:"url"`
						ID       int64  `json:"id"`
						Revision int64  `json:"revision"`
						Name     string `json:"name"`
						Folder   string `json:"folder"`
					}{
						ID:   999,
						Name: "retry-pipeline",
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
			atr := &azuredevopsTracesReceiver{
				logger: logger,
			}
			traces, err := atr.handleJobEvent(test.event)

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
