// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type testConfigCollection int

const (
	testSetDefault testConfigCollection = iota
	testSetAll
	testSetNone
)

func TestMetricsBuilder(t *testing.T) {
	tests := []struct {
		name      string
		configSet testConfigCollection
	}{
		{
			name:      "default",
			configSet: testSetDefault,
		},
		{
			name:      "all_set",
			configSet: testSetAll,
		},
		{
			name:      "none_set",
			configSet: testSetNone,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := pcommon.Timestamp(1_000_000_000)
			ts := pcommon.Timestamp(1_000_001_000)
			observedZapCore, observedLogs := observer.New(zap.WarnLevel)
			settings := receivertest.NewNopCreateSettings()
			settings.Logger = zap.New(observedZapCore)
			mb := NewMetricsBuilder(loadMetricsBuilderConfig(t, test.name), settings, WithStartTime(start))

			expectedWarnings := 0

			assert.Equal(t, expectedWarnings, observedLogs.Len())

			defaultMetricsCount := 0
			allMetricsCount := 0

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryBranchCommitAheadbyCountDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryBranchCommitBehindbyCountDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryBranchCountDataPoint(ts, 1, "repository.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryBranchLineAdditionCountDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryBranchLineDeletionCountDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryBranchTimeDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			allMetricsCount++
			mb.RecordGitRepositoryContributorCountDataPoint(ts, 1, "repository.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryCountDataPoint(ts, 1)

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryPullRequestCountDataPoint(ts, 1, AttributePullRequestStateOpen, "repository.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryPullRequestTimeOpenDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryPullRequestTimeToApprovalDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			defaultMetricsCount++
			allMetricsCount++
			mb.RecordGitRepositoryPullRequestTimeToMergeDataPoint(ts, 1, "repository.name-val", "branch.name-val")

			allMetricsCount++
			mb.RecordGitRepositoryVulnerabilitiesDataPoint(ts, 1, "repository.name-val", "severity-val")

			rb := mb.NewResourceBuilder()
			rb.SetGitVendorName("git.vendor.name-val")
			rb.SetOrganizationName("organization.name-val")
			res := rb.Emit()
			metrics := mb.Emit(WithResource(res))

			if test.configSet == testSetNone {
				assert.Equal(t, 0, metrics.ResourceMetrics().Len())
				return
			}

			assert.Equal(t, 1, metrics.ResourceMetrics().Len())
			rm := metrics.ResourceMetrics().At(0)
			assert.Equal(t, res, rm.Resource())
			assert.Equal(t, 1, rm.ScopeMetrics().Len())
			ms := rm.ScopeMetrics().At(0).Metrics()
			if test.configSet == testSetDefault {
				assert.Equal(t, defaultMetricsCount, ms.Len())
			}
			if test.configSet == testSetAll {
				assert.Equal(t, allMetricsCount, ms.Len())
			}
			validatedMetrics := make(map[string]bool)
			for i := 0; i < ms.Len(); i++ {
				switch ms.At(i).Name() {
				case "git.repository.branch.commit.aheadby.count":
					assert.False(t, validatedMetrics["git.repository.branch.commit.aheadby.count"], "Found a duplicate in the metrics slice: git.repository.branch.commit.aheadby.count")
					validatedMetrics["git.repository.branch.commit.aheadby.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Number of commits a branch is ahead of the default branch", ms.At(i).Description())
					assert.Equal(t, "{branch}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.branch.commit.behindby.count":
					assert.False(t, validatedMetrics["git.repository.branch.commit.behindby.count"], "Found a duplicate in the metrics slice: git.repository.branch.commit.behindby.count")
					validatedMetrics["git.repository.branch.commit.behindby.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Number of commits a branch is behind the default branch", ms.At(i).Description())
					assert.Equal(t, "{branch}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.branch.count":
					assert.False(t, validatedMetrics["git.repository.branch.count"], "Found a duplicate in the metrics slice: git.repository.branch.count")
					validatedMetrics["git.repository.branch.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Number of branches in a repository", ms.At(i).Description())
					assert.Equal(t, "{branch}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
				case "git.repository.branch.line.addition.count":
					assert.False(t, validatedMetrics["git.repository.branch.line.addition.count"], "Found a duplicate in the metrics slice: git.repository.branch.line.addition.count")
					validatedMetrics["git.repository.branch.line.addition.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Count of lines added to code in a branch", ms.At(i).Description())
					assert.Equal(t, "{branch}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.branch.line.deletion.count":
					assert.False(t, validatedMetrics["git.repository.branch.line.deletion.count"], "Found a duplicate in the metrics slice: git.repository.branch.line.deletion.count")
					validatedMetrics["git.repository.branch.line.deletion.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Count of lines deleted from code in a branch", ms.At(i).Description())
					assert.Equal(t, "{branch}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.branch.time":
					assert.False(t, validatedMetrics["git.repository.branch.time"], "Found a duplicate in the metrics slice: git.repository.branch.time")
					validatedMetrics["git.repository.branch.time"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Time the branch has existed", ms.At(i).Description())
					assert.Equal(t, "s", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.contributor.count":
					assert.False(t, validatedMetrics["git.repository.contributor.count"], "Found a duplicate in the metrics slice: git.repository.contributor.count")
					validatedMetrics["git.repository.contributor.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Total number of unique contributors to a repository", ms.At(i).Description())
					assert.Equal(t, "{contributor}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
				case "git.repository.count":
					assert.False(t, validatedMetrics["git.repository.count"], "Found a duplicate in the metrics slice: git.repository.count")
					validatedMetrics["git.repository.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "Number of repositories in an organization", ms.At(i).Description())
					assert.Equal(t, "{repository}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
				case "git.repository.pull_request.count":
					assert.False(t, validatedMetrics["git.repository.pull_request.count"], "Found a duplicate in the metrics slice: git.repository.pull_request.count")
					validatedMetrics["git.repository.pull_request.count"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "The number of pull requests in a repository, categorized by their state (either open or merged)", ms.At(i).Description())
					assert.Equal(t, "{pull_request}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("pull_request.state")
					assert.True(t, ok)
					assert.EqualValues(t, "open", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
				case "git.repository.pull_request.time_open":
					assert.False(t, validatedMetrics["git.repository.pull_request.time_open"], "Found a duplicate in the metrics slice: git.repository.pull_request.time_open")
					validatedMetrics["git.repository.pull_request.time_open"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "The amount of time a pull request has been open", ms.At(i).Description())
					assert.Equal(t, "s", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.pull_request.time_to_approval":
					assert.False(t, validatedMetrics["git.repository.pull_request.time_to_approval"], "Found a duplicate in the metrics slice: git.repository.pull_request.time_to_approval")
					validatedMetrics["git.repository.pull_request.time_to_approval"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "The amount of time it took a pull request to go from open to approved", ms.At(i).Description())
					assert.Equal(t, "s", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.pull_request.time_to_merge":
					assert.False(t, validatedMetrics["git.repository.pull_request.time_to_merge"], "Found a duplicate in the metrics slice: git.repository.pull_request.time_to_merge")
					validatedMetrics["git.repository.pull_request.time_to_merge"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "The amount of time it took a pull request to go from open to merged", ms.At(i).Description())
					assert.Equal(t, "s", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("branch.name")
					assert.True(t, ok)
					assert.EqualValues(t, "branch.name-val", attrVal.Str())
				case "git.repository.vulnerabilities":
					assert.False(t, validatedMetrics["git.repository.vulnerabilities"], "Found a duplicate in the metrics slice: git.repository.vulnerabilities")
					validatedMetrics["git.repository.vulnerabilities"] = true
					assert.Equal(t, pmetric.MetricTypeGauge, ms.At(i).Type())
					assert.Equal(t, 1, ms.At(i).Gauge().DataPoints().Len())
					assert.Equal(t, "The number of vulnerabilities in the repository", ms.At(i).Description())
					assert.Equal(t, "{severity}", ms.At(i).Unit())
					dp := ms.At(i).Gauge().DataPoints().At(0)
					assert.Equal(t, start, dp.StartTimestamp())
					assert.Equal(t, ts, dp.Timestamp())
					assert.Equal(t, pmetric.NumberDataPointValueTypeInt, dp.ValueType())
					assert.Equal(t, int64(1), dp.IntValue())
					attrVal, ok := dp.Attributes().Get("repository.name")
					assert.True(t, ok)
					assert.EqualValues(t, "repository.name-val", attrVal.Str())
					attrVal, ok = dp.Attributes().Get("severity")
					assert.True(t, ok)
					assert.EqualValues(t, "severity-val", attrVal.Str())
				}
			}
		})
	}
}
