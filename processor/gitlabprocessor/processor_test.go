package gitlabprocessor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.uber.org/zap"
)

func TestGetPipeCompAttrs(t *testing.T) {
	// Load test data
	testData, err := os.ReadFile("testdata/blobContent.json")
	require.NoError(t, err, "Failed to read test data file")

	tests := []struct {
		name          string
		responseData  string
		expectedComps map[string]string
		expectedError bool
		emptyResponse bool
	}{
		{
			name:         "Valid blob content with components",
			responseData: string(testData),
			expectedComps: map[string]string{
				"gitlab.com/liatrio2/components/general-util/lint-markdown":       "v0.0.2",
				"gitlab.com/liatrio2/components/general-util/SAST-snyk":           "v0.0.2",
				"gitlab.com/liatrio2/components/maven-build/unit-test":            "v0.0.2",
				"gitlab.com/liatrio2/components/maven-build/docker-build-publish": "v0.0.2",
			},
			expectedError: false,
		},
		{
			name:          "Empty blob content",
			responseData:  `{"data":{"project":{"repository":{"blobs":{"nodes":[{"rawBlob":""}]}}}}}`,
			expectedComps: map[string]string{},
			expectedError: false,
		},
		{
			name:          "No components in blob content",
			responseData:  `{"data":{"project":{"repository":{"blobs":{"nodes":[{"rawBlob":"stages:\n  - test\n  - build"}]}}}}}`,
			expectedComps: map[string]string{},
			expectedError: false,
		},
		{
			name:          "Empty nodes array",
			responseData:  `{"data":{"project":{"repository":{"blobs":{"nodes":[]}}}}}`,
			expectedComps: nil,
			expectedError: true,
			emptyResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(tt.responseData))
				require.NoError(t, err)
			}))
			defer server.Close()

			// Create processor with test configuration
			logger := zap.NewNop()
			cfg := Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: server.URL,
				},
			}
			processor := &logProcessor{
				logger: logger,
				cfg:    &cfg,
				client: server.Client(),
			}

			// Call the function
			comps, err := processor.getPipeCompAttrs(context.Background(), "test/repo", "main")

			// Check results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, comps)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedComps, comps)
			}

			// Additional checks for empty response
			if tt.emptyResponse {
				assert.Nil(t, comps)
			}
		})
	}
}

// Helper function to validate JSON response structure
func validateJSONResponse(t *testing.T, data []byte) {
	var response struct {
		Data struct {
			Project struct {
				Repository struct {
					Blobs struct {
						Nodes []struct {
							RawBlob string `json:"rawBlob"`
						} `json:"nodes"`
					} `json:"blobs"`
				} `json:"repository"`
			} `json:"project"`
		} `json:"data"`
	}

	err := json.Unmarshal(data, &response)
	require.NoError(t, err, "Failed to parse JSON response")
}
