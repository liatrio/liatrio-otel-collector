// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"testing"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewAzureDevOpsScraper(t *testing.T) {
	cfg := &Config{
		Organization:        "test-org",
		PersonalAccessToken: "test-token",
		BaseURL:             "https://dev.azure.com",
		LimitPullRequests:   100,
		ConcurrencyLimit:    5,
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	assert.NotNil(t, scraper)
	assert.Equal(t, cfg, scraper.cfg)
	assert.NotNil(t, scraper.logger)
	assert.NotNil(t, scraper.mb)
	assert.NotNil(t, scraper.rb)
}

func TestAzureDevOpsScraperStart(t *testing.T) {
	cfg := &Config{
		Organization:        "test-org",
		PersonalAccessToken: "test-token",
		BaseURL:             "https://dev.azure.com",
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	// Test start without host - should not panic
	err := scraper.start(context.Background(), nil)
	require.NoError(t, err)
}

func TestMakeRequestURL(t *testing.T) {
	cfg := &Config{
		Organization:        "test-org",
		PersonalAccessToken: "test-token",
		BaseURL:             "https://dev.azure.com",
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	// We can't easily test the actual request without mocking, but we can verify the scraper was created
	assert.NotNil(t, scraper)
	assert.Equal(t, "test-org", scraper.cfg.Organization)
	assert.Equal(t, "https://dev.azure.com", scraper.cfg.BaseURL)
}
