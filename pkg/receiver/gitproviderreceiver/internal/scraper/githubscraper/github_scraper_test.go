// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/scraper/githubscraper/mocks"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver"
)

func TestNewGitHubScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitHubScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

func TestGetPullRequests(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitHubScraper(context.Background(), receiver.CreateSettings{}, defaultConfig.(*Config))

	client := mocks.NewClient(t)
	client.On("MakeRequest", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("*graphql.Request"), mock.AnythingOfType("*graphql.Response")).
		Return(func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
			var data getPullRequestCountResponse
			var err error

			resp = &graphql.Response{Data: &data}

			return err
		})

	ctx := context.Background()
	repos := make([]SearchNode, 5)
	now := pcommon.NewTimestampFromTime(time.Now())

	var wg sync.WaitGroup
	pullRequests := make(chan []PullRequestNode)

	wg.Add(2)
	s.getPullRequests(
		ctx,
		client,
		repos,
		now,
		pullRequests,
		&wg,
	)
	wg.Wait()
	close(pullRequests)

	assert.NotNil(t, s)

}
