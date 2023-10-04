// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver/internal/scraper/githubscraper/mocks"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/receivertest"
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

	settings := receivertest.NewNopCreateSettings()

	s := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))

	repos := []SearchNode{}
	mockrepo := mocks.NewMySearchNode(t)

	mockrepo.Name = "foo"
	mockrepo.Typename = "Repository"
	mockrepo.Id = "foo"

	mockrepo.On("GetTypename").
		Return(func() string {
			return "foo"
		})

	repos = append(repos, mockrepo)

	client := mocks.NewClient(t)
	client.On("MakeRequest", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("*graphql.Request"), mock.AnythingOfType("*graphql.Response")).
		Return(func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
			var data getPullRequestCountResponse
			var err error

			data.Repository.PullRequests.TotalCount = 10

			resp.Data = data

			return err
		})

	ctx := context.Background()
	now := pcommon.NewTimestampFromTime(time.Now())

	var wg sync.WaitGroup
	pullRequests := make(chan []PullRequestNode)

	wg.Add(1)
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

}

//func TestGetPullRequests2(t *testing.T) {
//	factory := Factory{}
//	defaultConfig := factory.CreateDefaultConfig()
//
//	settings := receivertest.NewNopCreateSettings()
//
//	s := newGitHubScraper(context.Background(), settings, defaultConfig.(*Config))
//
//	repos := []SearchNode{}
//	mockrepo := mocks.NewMySearchNode(t)
//
//	mockrepo.Name = "foo"
//	mockrepo.Typename = "Repository"
//	mockrepo.Id = "foo"
//
//	mockrepo.On("GetTypename").
//		Return(func() string {
//			return "foo"
//		})
//
//	repos = append(repos, mockrepo)
//	var client graphql.Client
//	old := zgetPullRequestCount
//	defer func() { zgetPullRequestCount = old }()
//
//	zgetPullRequestCount = func(
//		ctx context.Context,
//		client graphql.Client,
//		name string,
//		owner string,
//		states []PullRequestState,
//	) (*getPullRequestCountResponse, error) {
//		// This will be called, do whatever you want to,
//		// return whatever you want to
//
//		var data getPullRequestCountResponse
//		return &data, nil
//	}
//
//	ctx := context.Background()
//	now := pcommon.NewTimestampFromTime(time.Now())
//
//	var wg sync.WaitGroup
//	pullRequests := make(chan []PullRequestNode)
//
//	wg.Add(2)
//	s.getPullRequests(
//		ctx,
//		client,
//		repos,
//		now,
//		pullRequests,
//		&wg,
//	)
//	wg.Wait()
//	close(pullRequests)
//
//}
