package gitlabprocessor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestNewLogProcessor(t *testing.T) {
	// Create a logger for testing
	logger := zap.NewNop()
	cfg := &Config{
		Token: "test-token",
	}

	// Create processor
	processor := newLogProcessor(context.Background(), logger, cfg)

	// Verify processor was created
	assert.NotNil(t, processor)
	assert.Equal(t, logger, processor.logger)
	assert.Equal(t, cfg, processor.cfg)
}

func TestProcessLogs_EmptyLogs(t *testing.T) {
	// Create a logger for testing
	logger := zap.NewNop()
	cfg := &Config{
		Token: "test-token",
	}

	// Create processor
	processor := newLogProcessor(context.Background(), logger, cfg)

	// Create empty logs
	emptyLogs := plog.NewLogs()

	// Process empty logs
	result, err := processor.processLogs(context.Background(), emptyLogs)

	// Verify result
	assert.NoError(t, err)
	assert.Equal(t, emptyLogs, result)
}

func TestProcessLogs(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create processor with mock getPipeCompAttrs
	proc := &pipelineProcessor{
		cfg: &Config{
			ClientConfig: confighttp.ClientConfig{
				Endpoint: server.URL,
			},
			Token: "test-token",
		},
		logger: zap.NewNop(),
		GetPipeCompAttrsFn: func(ctx context.Context, fullPath string, revision string) (map[string]string, error) {
			return map[string]string{
				"example-org/pipeline-components/components/test": "1.0.0",
			}, nil
		},
	}

	// Test successful enrichment
	inputLogs, err := golden.ReadLogs("testdata/input_logs.yaml")
	require.NoError(t, err)

	result, err := proc.processLogs(context.Background(), inputLogs)
	require.NoError(t, err)

	expected, err := golden.ReadLogs("testdata/expected_logs.yaml")
	require.NoError(t, err)
	require.Equal(t, expected, result, "Logs do not match expected output")

	// Test missing repository name
	inputLogs, err = golden.ReadLogs("testdata/input_logs_missing_repo.yaml")
	require.NoError(t, err)

	result, err = proc.processLogs(context.Background(), inputLogs)
	require.NoError(t, err)

	// Should not modify logs when repository name is missing
	require.Equal(t, inputLogs, result, "Logs should not be modified when missing repository name")
}
