// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubscraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v89/github"
	"github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewGitHubScraper(t *testing.T) {
	factory := Factory{}
	defaultConfig := factory.CreateDefaultConfig()

	s := newGitHubScraper(receiver.Settings{}, defaultConfig.(*Config))

	assert.NotNil(t, s)
}

func TestScrape(t *testing.T) {
	testCases := []struct {
		desc     string
		server   *http.ServeMux
		testFile string
	}{
		{
			desc: "TestNoRepos",
			server: MockServer(&responses{
				scrape: true,
				checkLoginResponse: loginResponse{
					checkLogin: checkLoginResponse{
						Organization: checkLoginOrganization{
							Login: "liatrio",
						},
					},
					responseCode: http.StatusOK,
				},
				searchRepoResponse: searchRepoResponse{
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 0,
							Nodes:           []SearchNode{},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			testFile: "expected_no_repos.yaml",
		},
		{
			desc: "TestHappyPath",
			server: MockServer(&responses{
				scrape: true,
				checkLoginResponse: loginResponse{
					checkLogin: checkLoginResponse{
						Organization: checkLoginOrganization{
							Login: "liatrio",
						},
					},
					responseCode: http.StatusOK,
				},
				searchRepoResponse: searchRepoResponse{
					limit: rateVals{
						Limit:     5000,
						Remaining: 4999,
						Cost:      1,
						ResetAt:   time.Now().Add(1 * time.Hour),
					},
					repos: []getRepoDataBySearchSearchSearchResultItemConnection{
						{
							RepositoryCount: 1,
							Nodes: []SearchNode{
								&SearchNodeRepository{
									Repo: Repo{
										Name:             "repo1",
										DefaultBranchRef: RepoDefaultBranchRef{Name: "main"},
									},
								},
							},
							PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
				prResponse: prResponse{
					prs: []getPullRequestDataRepositoryPullRequestsPullRequestConnection{
						{
							PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
								HasNextPage: false,
							},
							Nodes: []PullRequestNode{
								{
									Merged: false,
								},
								{
									Merged: true,
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
				branchResponse: branchResponse{
					branches: []getBranchDataRepositoryRefsRefConnection{
						{
							TotalCount: 1,
							Nodes: []BranchNode{
								{
									Name: "dev",
									Compare: BranchNodeCompareComparison{
										AheadBy:  0,
										BehindBy: 1,
									},
								},
							},
							PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
				commitResponse: commitResponse{
					commits: []BranchHistoryTargetCommit{
						{
							History: BranchHistoryTargetCommitHistoryCommitHistoryConnection{
								Nodes: []CommitNode{
									{

										CommittedDate: time.Now().AddDate(0, 0, -1),
										Additions:     10,
										Deletions:     9,
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
				contribResponse: contribResponse{
					contribs: [][]*github.Contributor{
						{
							{
								ID: github.Ptr(int64(1)),
							},
						},
					},
					responseCode: http.StatusOK,
				},
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{
						{
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "HIGH",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "MODERATE",
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			testFile: "expected_happy_path.yaml",
		},
		{
			desc: "TestHappyPathWithTeam",
			server: MockServer(&responses{
				scrape: true,
				checkLoginResponse: loginResponse{
					checkLogin: checkLoginResponse{
						Organization: checkLoginOrganization{
							Login: "liatrio",
						},
					},
					responseCode: http.StatusOK,
				},
				teamRepoResponse: teamRepoResponse{
					repos: []getRepoDataByTeamOrganizationTeamRepositoriesTeamRepositoryConnection{
						{
							TotalCount: 1,
							Nodes: []TeamNode{
								{
									Repo: Repo{
										Name:             "repo1",
										DefaultBranchRef: RepoDefaultBranchRef{Name: "main"},
									},
								},
							},
							PageInfo: getRepoDataByTeamOrganizationTeamRepositoriesTeamRepositoryConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
				prResponse: prResponse{
					prs: []getPullRequestDataRepositoryPullRequestsPullRequestConnection{
						{
							PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
								HasNextPage: false,
							},
							Nodes: []PullRequestNode{
								{
									Merged: false,
								},
								{
									Merged: true,
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
				branchResponse: branchResponse{
					branches: []getBranchDataRepositoryRefsRefConnection{
						{
							TotalCount: 1,
							Nodes: []BranchNode{
								{
									Name: "dev",
									Compare: BranchNodeCompareComparison{
										AheadBy:  0,
										BehindBy: 1,
									},
								},
							},
							PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
								HasNextPage: false,
							},
						},
					},
					responseCode: http.StatusOK,
				},
				commitResponse: commitResponse{
					commits: []BranchHistoryTargetCommit{
						{
							History: BranchHistoryTargetCommitHistoryCommitHistoryConnection{
								Nodes: []CommitNode{
									{
										//Because the date was static, the test would fail as the branch age would change as time passed
										//Made it dynamically generated for yesterdays date, keeping the age at 24 hours
										CommittedDate: time.Now().AddDate(0, 0, -1),
										Additions:     10,
										Deletions:     9,
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
				contribResponse: contribResponse{
					contribs: [][]*github.Contributor{
						{
							{
								ID: github.Ptr(int64(1)),
							},
						},
					},
					responseCode: http.StatusOK,
				},
				depBotAlertResponse: depBotAlertResponse{
					depBotsAlerts: []VulnerabilityAlerts{
						{
							Nodes: []CVENode{
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "HIGH",
									},
								},
								{
									SecurityVulnerability: CVENodeSecurityVulnerability{
										Severity: "MODERATE",
									},
								},
							},
						},
					},
					responseCode: http.StatusOK,
				},
			}),
			testFile: "expected_happy_path_with_team.yaml",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			server := httptest.NewServer(tc.server)
			defer server.Close()

			cfg := &Config{MetricsBuilderConfig: metadata.NewDefaultMetricsBuilderConfig()}

			cfg.Metrics.VcsCveCount.Enabled = true

			ghs := newGitHubScraper(receivertest.NewNopSettings(metadata.Type), cfg)
			ghs.cfg.GitHubOrg = "liatrio"
			ghs.cfg.Endpoint = server.URL
			ghs.cfg.ConcurrencyLimit = 1000

			// TestHappyPathWithTeam is a special case where we need to set the team name
			if tc.desc == "TestHappyPathWithTeam" {
				cfg.ResourceAttributes.TeamName.Enabled = true
				ghs.cfg.GitHubTeam = "tag-o11y"
				err := ghs.start(ctx, componenttest.NewNopHost())
				require.NoError(t, err)

				actualMetrics, err := ghs.scrape(ctx)
				require.NoError(t, err)

				expectedFile := filepath.Join("testdata", "scraper", tc.testFile)

				// Due to the generative nature of the code we're using through genqlient. The tests happy path changes,
				// and needs to be rebuilt to satisfy the unit tests. When the metadata.yaml changes, and code is
				// introduced, or removed. We'll need to update the metrics by uncommenting the below and running
				// `make test` to generate it. Then we're safe to comment this out again and see happy tests.
				// golden.WriteMetrics(t, expectedFile, actualMetrics) // This line is temporary! TODO remove this!!

				expectedMetrics, err := golden.ReadMetrics(expectedFile)
				require.NoError(t, err)
				require.NoError(t, pmetrictest.CompareMetrics(
					expectedMetrics,
					actualMetrics,
					pmetrictest.IgnoreMetricDataPointsOrder(),
					pmetrictest.IgnoreTimestamp(),
					pmetrictest.IgnoreStartTimestamp(),
				))
			} else {
				err := ghs.start(ctx, componenttest.NewNopHost())
				require.NoError(t, err)

				actualMetrics, err := ghs.scrape(ctx)
				require.NoError(t, err)

				expectedFile := filepath.Join("testdata", "scraper", tc.testFile)

				// Due to the generative nature of the code we're using through genqlient. The tests happy path changes,
				// and needs to be rebuilt to satisfy the unit tests. When the metadata.yaml changes, and code is
				// introduced, or removed. We'll need to update the metrics by uncommenting the below and running
				// `make test` to generate it. Then we're safe to comment this out again and see happy tests.
				// golden.WriteMetrics(t, expectedFile, actualMetrics) // This line is temporary! TODO remove this!!

				expectedMetrics, err := golden.ReadMetrics(expectedFile)
				require.NoError(t, err)
				require.NoError(t, pmetrictest.CompareMetrics(
					expectedMetrics,
					actualMetrics,
					pmetrictest.IgnoreMetricDataPointsOrder(),
					pmetrictest.IgnoreTimestamp(),
					pmetrictest.IgnoreStartTimestamp(),
				))
			}
		})
	}
}

// singleRepoResponses builds a happy-path mock-server response set for a
// single repository. The panic-recovery tests below pair it with
// panicRoundTripper to inject a synthetic panic from inside the per-repo
// scrape goroutine, decoupling the recovery-path coverage from any specific
// production nil-deref bug.
func singleRepoResponses(repoName string) *responses {
	return &responses{
		scrape: true,
		checkLoginResponse: loginResponse{
			checkLogin: checkLoginResponse{
				Organization: checkLoginOrganization{Login: "liatrio"},
			},
			responseCode: http.StatusOK,
		},
		searchRepoResponse: searchRepoResponse{
			repos: []getRepoDataBySearchSearchSearchResultItemConnection{
				{
					RepositoryCount: 1,
					Nodes: []SearchNode{
						&SearchNodeRepository{Repo: Repo{Name: repoName}},
					},
					PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
						HasNextPage: false,
					},
				},
			},
			responseCode: http.StatusOK,
		},
		branchResponse: branchResponse{
			branches: []getBranchDataRepositoryRefsRefConnection{
				{
					TotalCount: 0,
					Nodes:      []BranchNode{},
					PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
						HasNextPage: false,
					},
				},
			},
			responseCode: http.StatusOK,
		},
		prResponse: prResponse{
			prs: []getPullRequestDataRepositoryPullRequestsPullRequestConnection{
				{
					PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
						HasNextPage: false,
					},
					Nodes: []PullRequestNode{},
				},
			},
			responseCode: http.StatusOK,
		},
		contribResponse: contribResponse{
			contribs:     [][]*github.Contributor{{}},
			responseCode: http.StatusOK,
		},
		depBotAlertResponse: depBotAlertResponse{
			depBotsAlerts: []VulnerabilityAlerts{{Nodes: []CVENode{}}},
			responseCode:  http.StatusOK,
		},
	}
}

// panicRoundTripper wraps an inner http.RoundTripper and panics on any
// request whose URL path contains trigger. It is the panic injector for the
// recover()-path tests: the panic fires inside the per-repo scrape
// goroutine's call to the GitHub client, which is exactly where the
// production recover() must catch it.
type panicRoundTripper struct {
	inner   http.RoundTripper
	trigger string
}

func (p *panicRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Path, p.trigger) {
		panic("synthetic test panic from panicRoundTripper: " + req.URL.Path)
	}
	inner := p.inner
	if inner == nil {
		inner = http.DefaultTransport
	}
	return inner.RoundTrip(req)
}

// TestScrapeRecoversFromPanic asserts that a runtime panic inside a per-repo
// scrape goroutine does not bring down the collector: scrape returns to its
// caller and metrics recorded prior to the failing goroutine are still emitted.
// Without recover() in place the panic propagates out of the goroutine and
// crashes the test binary.
func TestScrapeRecoversFromPanic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server := httptest.NewServer(MockServer(singleRepoResponses("repo1")))
	defer server.Close()

	cfg := &Config{MetricsBuilderConfig: metadata.NewDefaultMetricsBuilderConfig()}
	cfg.Metrics.VcsCveCount.Enabled = true

	// Attach an observed logger so the test can confirm a panic was actually
	// recovered. Without this guard, removing the panic injector below would
	// silently turn this test into a happy-path no-op that still passes.
	core, recorded := observer.New(zap.ErrorLevel)
	settings := receivertest.NewNopSettings(metadata.Type)
	settings.Logger = zap.New(core)

	ghs := newGitHubScraper(settings, cfg)
	ghs.cfg.GitHubOrg = "liatrio"
	ghs.cfg.Endpoint = server.URL
	ghs.cfg.ConcurrencyLimit = 1000

	require.NoError(t, ghs.start(ctx, componenttest.NewNopHost()))

	// Inject a synthetic panic from inside the per-repo scrape goroutine by
	// wrapping the HTTP transport. The code-scanning REST call is the last
	// per-repo HTTP request and runs after mux.Lock(), so the panic exercises
	// both recover() and defer mux.Unlock() in github_scraper.go.
	ghs.client.Transport = &panicRoundTripper{
		inner:   ghs.client.Transport,
		trigger: "/code-scanning/alerts",
	}

	metrics, err := ghs.scrape(ctx)
	require.NoError(t, err)

	require.NotEmpty(t, recorded.FilterMessageSnippet("panic").All(),
		"test no longer triggers a panic — recovery path is not being exercised")

	// The repository count metric is recorded before the goroutine fan-out,
	// so it must still be present after a panic is recovered.
	require.Equal(t, 1, metrics.ResourceMetrics().Len())
}

// TestScrapeLogsRecoveredPanic asserts that a panic in a per-repo scrape
// goroutine is logged as an error with the repo name and the recovered value
// so operators can identify which repo failed without re-deriving it from a
// stack trace.
func TestScrapeLogsRecoveredPanic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// The mock server hard-codes "/api/v3/repos/liatrio/repo1/..." for the
	// REST endpoints, so the failing repo must be "repo1" so all per-repo
	// HTTP calls are served (and so panicRoundTripper has a code-scanning
	// request to match against).
	server := httptest.NewServer(MockServer(singleRepoResponses("repo1")))
	defer server.Close()

	cfg := &Config{MetricsBuilderConfig: metadata.NewDefaultMetricsBuilderConfig()}
	cfg.Metrics.VcsCveCount.Enabled = true

	core, recorded := observer.New(zap.ErrorLevel)
	settings := receivertest.NewNopSettings(metadata.Type)
	settings.Logger = zap.New(core)

	ghs := newGitHubScraper(settings, cfg)
	ghs.cfg.GitHubOrg = "liatrio"
	ghs.cfg.Endpoint = server.URL
	ghs.cfg.ConcurrencyLimit = 1000

	require.NoError(t, ghs.start(ctx, componenttest.NewNopHost()))

	ghs.client.Transport = &panicRoundTripper{
		inner:   ghs.client.Transport,
		trigger: "/code-scanning/alerts",
	}

	_, err := ghs.scrape(ctx)
	require.NoError(t, err)

	entries := recorded.FilterMessageSnippet("panic").All()
	require.Len(t, entries, 1, "expected exactly one recovery log entry")

	fields := entries[0].ContextMap()
	require.Equal(t, "repo1", fields["repo"], "log must identify the failing repo")
	require.NotNil(t, fields["panic"], "log must carry the recovered value")
}

// TestScrapeDoesNotDeadlockAfterRecoveredPanic asserts that the per-repo
// mutex is released when a goroutine panics. The original code took the lock
// with mux.Lock() and released it manually near the end of the function, so a
// panic between those points (now caught by recover()) leaves the mutex held
// and any sibling goroutine deadlocks on its own mux.Lock(), which then hangs
// wg.Wait() and the entire scrape call. The fix is to defer mux.Unlock().
func TestScrapeDoesNotDeadlockAfterRecoveredPanic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Two repos under ConcurrencyLimit=1 so they run sequentially. Both
	// goroutines panic on their code-scanning HTTP call (see the
	// panicRoundTripper installed below); the assertion is that repo2's
	// mux.Lock() does not block on a lock leaked from repo1's panic.
	r := singleRepoResponses("repo1")
	r.searchRepoResponse.repos = []getRepoDataBySearchSearchSearchResultItemConnection{
		{
			RepositoryCount: 2,
			Nodes: []SearchNode{
				&SearchNodeRepository{Repo: Repo{Name: "repo1"}},
				&SearchNodeRepository{Repo: Repo{Name: "repo2"}},
			},
			PageInfo: getRepoDataBySearchSearchSearchResultItemConnectionPageInfo{
				HasNextPage: false,
			},
		},
	}
	// Per-repo GraphQL responses are indexed by a page counter that
	// advances on every request, so we need one entry per repo.
	emptyBranches := getBranchDataRepositoryRefsRefConnection{
		TotalCount: 0,
		Nodes:      []BranchNode{},
		PageInfo: getBranchDataRepositoryRefsRefConnectionPageInfo{
			HasNextPage: false,
		},
	}
	r.branchResponse.branches = []getBranchDataRepositoryRefsRefConnection{emptyBranches, emptyBranches}
	emptyPRs := getPullRequestDataRepositoryPullRequestsPullRequestConnection{
		PageInfo: getPullRequestDataRepositoryPullRequestsPullRequestConnectionPageInfo{
			HasNextPage: false,
		},
		Nodes: []PullRequestNode{},
	}
	r.prResponse.prs = []getPullRequestDataRepositoryPullRequestsPullRequestConnection{emptyPRs, emptyPRs}
	r.depBotAlertResponse.depBotsAlerts = []VulnerabilityAlerts{
		{Nodes: []CVENode{}},
		{Nodes: []CVENode{}},
	}

	server := httptest.NewServer(MockServer(r))
	defer server.Close()

	cfg := &Config{MetricsBuilderConfig: metadata.NewDefaultMetricsBuilderConfig()}
	cfg.Metrics.VcsCveCount.Enabled = true

	// Same guard as TestScrapeRecoversFromPanic: if the underlying panic
	// trigger ever stops firing, the deadlock scenario no longer exists and
	// this test would silently pass without exercising the deferred unlock.
	core, recorded := observer.New(zap.ErrorLevel)
	settings := receivertest.NewNopSettings(metadata.Type)
	settings.Logger = zap.New(core)

	ghs := newGitHubScraper(settings, cfg)
	ghs.cfg.GitHubOrg = "liatrio"
	ghs.cfg.Endpoint = server.URL
	// Force sequential execution so the second goroutine only starts after
	// the first has panicked-and-recovered, removing any timing ambiguity.
	ghs.cfg.ConcurrencyLimit = 1

	require.NoError(t, ghs.start(ctx, componenttest.NewNopHost()))

	ghs.client.Transport = &panicRoundTripper{
		inner:   ghs.client.Transport,
		trigger: "/code-scanning/alerts",
	}

	done := make(chan struct{})
	var scrapeErr error
	go func() {
		_, scrapeErr = ghs.scrape(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scrape deadlocked: recovered panic left the per-repo mutex locked")
	}
	require.NoError(t, scrapeErr)

	require.NotEmpty(t, recorded.FilterMessageSnippet("panic").All(),
		"test no longer triggers a panic — deferred-unlock path is not being exercised")
}
