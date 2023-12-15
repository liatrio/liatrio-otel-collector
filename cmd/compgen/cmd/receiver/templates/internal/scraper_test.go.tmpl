// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

var creationSet = receivertest.NewNopCreateSettings()

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
