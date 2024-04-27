// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/filter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
)

// AttributePullRequestState specifies the a value pull_request.state attribute.
type AttributePullRequestState int

const (
	_ AttributePullRequestState = iota
	AttributePullRequestStateOpen
	AttributePullRequestStateMerged
)

// String returns the string representation of the AttributePullRequestState.
func (av AttributePullRequestState) String() string {
	switch av {
	case AttributePullRequestStateOpen:
		return "open"
	case AttributePullRequestStateMerged:
		return "merged"
	}
	return ""
}

// MapAttributePullRequestState is a helper map of string to AttributePullRequestState attribute value.
var MapAttributePullRequestState = map[string]AttributePullRequestState{
	"open":   AttributePullRequestStateOpen,
	"merged": AttributePullRequestStateMerged,
}

type metricGitRepositoryBranchCommitAheadbyCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.commit.aheadby.count metric with initial data.
func (m *metricGitRepositoryBranchCommitAheadbyCount) init() {
	m.data.SetName("git.repository.branch.commit.aheadby.count")
	m.data.SetDescription("Number of commits a branch is ahead of the default branch")
	m.data.SetUnit("{branch}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchCommitAheadbyCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryBranchCommitAheadbyCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchCommitAheadbyCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchCommitAheadbyCount(cfg MetricConfig) metricGitRepositoryBranchCommitAheadbyCount {
	m := metricGitRepositoryBranchCommitAheadbyCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryBranchCommitBehindbyCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.commit.behindby.count metric with initial data.
func (m *metricGitRepositoryBranchCommitBehindbyCount) init() {
	m.data.SetName("git.repository.branch.commit.behindby.count")
	m.data.SetDescription("Number of commits a branch is behind the default branch")
	m.data.SetUnit("{branch}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchCommitBehindbyCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryBranchCommitBehindbyCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchCommitBehindbyCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchCommitBehindbyCount(cfg MetricConfig) metricGitRepositoryBranchCommitBehindbyCount {
	m := metricGitRepositoryBranchCommitBehindbyCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryBranchCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.count metric with initial data.
func (m *metricGitRepositoryBranchCount) init() {
	m.data.SetName("git.repository.branch.count")
	m.data.SetDescription("Number of branches in a repository")
	m.data.SetUnit("{branch}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryBranchCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchCount(cfg MetricConfig) metricGitRepositoryBranchCount {
	m := metricGitRepositoryBranchCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryBranchLineAdditionCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.line.addition.count metric with initial data.
func (m *metricGitRepositoryBranchLineAdditionCount) init() {
	m.data.SetName("git.repository.branch.line.addition.count")
	m.data.SetDescription("Count of lines added to code in a branch")
	m.data.SetUnit("{branch}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchLineAdditionCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryBranchLineAdditionCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchLineAdditionCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchLineAdditionCount(cfg MetricConfig) metricGitRepositoryBranchLineAdditionCount {
	m := metricGitRepositoryBranchLineAdditionCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryBranchLineDeletionCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.line.deletion.count metric with initial data.
func (m *metricGitRepositoryBranchLineDeletionCount) init() {
	m.data.SetName("git.repository.branch.line.deletion.count")
	m.data.SetDescription("Count of lines deleted from code in a branch")
	m.data.SetUnit("{branch}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchLineDeletionCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryBranchLineDeletionCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchLineDeletionCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchLineDeletionCount(cfg MetricConfig) metricGitRepositoryBranchLineDeletionCount {
	m := metricGitRepositoryBranchLineDeletionCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryBranchTime struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.time metric with initial data.
func (m *metricGitRepositoryBranchTime) init() {
	m.data.SetName("git.repository.branch.time")
	m.data.SetDescription("Time the branch has existed")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchTime) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryBranchTime) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchTime) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchTime(cfg MetricConfig) metricGitRepositoryBranchTime {
	m := metricGitRepositoryBranchTime{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryContributorCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.contributor.count metric with initial data.
func (m *metricGitRepositoryContributorCount) init() {
	m.data.SetName("git.repository.contributor.count")
	m.data.SetDescription("Total number of unique contributors to a repository")
	m.data.SetUnit("{contributor}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryContributorCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryContributorCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryContributorCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryContributorCount(cfg MetricConfig) metricGitRepositoryContributorCount {
	m := metricGitRepositoryContributorCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.count metric with initial data.
func (m *metricGitRepositoryCount) init() {
	m.data.SetName("git.repository.count")
	m.data.SetDescription("Number of repositories in an organization")
	m.data.SetUnit("{repository}")
	m.data.SetEmptyGauge()
}

func (m *metricGitRepositoryCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryCount(cfg MetricConfig) metricGitRepositoryCount {
	m := metricGitRepositoryCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryPullRequestCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.count metric with initial data.
func (m *metricGitRepositoryPullRequestCount) init() {
	m.data.SetName("git.repository.pull_request.count")
	m.data.SetDescription("The number of pull requests in a repository, categorized by their state (either open or merged)")
	m.data.SetUnit("{pull_request}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, pullRequestStateAttributeValue string, repositoryNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("pull_request.state", pullRequestStateAttributeValue)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryPullRequestCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestCount(cfg MetricConfig) metricGitRepositoryPullRequestCount {
	m := metricGitRepositoryPullRequestCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryPullRequestTimeOpen struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.time_open metric with initial data.
func (m *metricGitRepositoryPullRequestTimeOpen) init() {
	m.data.SetName("git.repository.pull_request.time_open")
	m.data.SetDescription("The amount of time a pull request has been open")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestTimeOpen) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryPullRequestTimeOpen) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestTimeOpen) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestTimeOpen(cfg MetricConfig) metricGitRepositoryPullRequestTimeOpen {
	m := metricGitRepositoryPullRequestTimeOpen{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryPullRequestTimeToApproval struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.time_to_approval metric with initial data.
func (m *metricGitRepositoryPullRequestTimeToApproval) init() {
	m.data.SetName("git.repository.pull_request.time_to_approval")
	m.data.SetDescription("The amount of time it took a pull request to go from open to approved")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestTimeToApproval) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryPullRequestTimeToApproval) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestTimeToApproval) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestTimeToApproval(cfg MetricConfig) metricGitRepositoryPullRequestTimeToApproval {
	m := metricGitRepositoryPullRequestTimeToApproval{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryPullRequestTimeToMerge struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.time_to_merge metric with initial data.
func (m *metricGitRepositoryPullRequestTimeToMerge) init() {
	m.data.SetName("git.repository.pull_request.time_to_merge")
	m.data.SetDescription("The amount of time it took a pull request to go from open to merged")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestTimeToMerge) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("repository.name", repositoryNameAttributeValue)
	dp.Attributes().PutStr("branch.name", branchNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricGitRepositoryPullRequestTimeToMerge) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestTimeToMerge) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestTimeToMerge(cfg MetricConfig) metricGitRepositoryPullRequestTimeToMerge {
	m := metricGitRepositoryPullRequestTimeToMerge{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

// MetricsBuilder provides an interface for scrapers to report metrics while taking care of all the transformations
// required to produce metric representation defined in metadata and user config.
type MetricsBuilder struct {
	config                                       MetricsBuilderConfig // config of the metrics builder.
	startTime                                    pcommon.Timestamp    // start time that will be applied to all recorded data points.
	metricsCapacity                              int                  // maximum observed number of metrics per resource.
	metricsBuffer                                pmetric.Metrics      // accumulates metrics data before emitting.
	buildInfo                                    component.BuildInfo  // contains version information.
	resourceAttributeIncludeFilter               map[string]filter.Filter
	resourceAttributeExcludeFilter               map[string]filter.Filter
	metricGitRepositoryBranchCommitAheadbyCount  metricGitRepositoryBranchCommitAheadbyCount
	metricGitRepositoryBranchCommitBehindbyCount metricGitRepositoryBranchCommitBehindbyCount
	metricGitRepositoryBranchCount               metricGitRepositoryBranchCount
	metricGitRepositoryBranchLineAdditionCount   metricGitRepositoryBranchLineAdditionCount
	metricGitRepositoryBranchLineDeletionCount   metricGitRepositoryBranchLineDeletionCount
	metricGitRepositoryBranchTime                metricGitRepositoryBranchTime
	metricGitRepositoryContributorCount          metricGitRepositoryContributorCount
	metricGitRepositoryCount                     metricGitRepositoryCount
	metricGitRepositoryPullRequestCount          metricGitRepositoryPullRequestCount
	metricGitRepositoryPullRequestTimeOpen       metricGitRepositoryPullRequestTimeOpen
	metricGitRepositoryPullRequestTimeToApproval metricGitRepositoryPullRequestTimeToApproval
	metricGitRepositoryPullRequestTimeToMerge    metricGitRepositoryPullRequestTimeToMerge
}

// metricBuilderOption applies changes to default metrics builder.
type metricBuilderOption func(*MetricsBuilder)

// WithStartTime sets startTime on the metrics builder.
func WithStartTime(startTime pcommon.Timestamp) metricBuilderOption {
	return func(mb *MetricsBuilder) {
		mb.startTime = startTime
	}
}

func NewMetricsBuilder(mbc MetricsBuilderConfig, settings receiver.CreateSettings, options ...metricBuilderOption) *MetricsBuilder {
	mb := &MetricsBuilder{
		config:        mbc,
		startTime:     pcommon.NewTimestampFromTime(time.Now()),
		metricsBuffer: pmetric.NewMetrics(),
		buildInfo:     settings.BuildInfo,
		metricGitRepositoryBranchCommitAheadbyCount:  newMetricGitRepositoryBranchCommitAheadbyCount(mbc.Metrics.GitRepositoryBranchCommitAheadbyCount),
		metricGitRepositoryBranchCommitBehindbyCount: newMetricGitRepositoryBranchCommitBehindbyCount(mbc.Metrics.GitRepositoryBranchCommitBehindbyCount),
		metricGitRepositoryBranchCount:               newMetricGitRepositoryBranchCount(mbc.Metrics.GitRepositoryBranchCount),
		metricGitRepositoryBranchLineAdditionCount:   newMetricGitRepositoryBranchLineAdditionCount(mbc.Metrics.GitRepositoryBranchLineAdditionCount),
		metricGitRepositoryBranchLineDeletionCount:   newMetricGitRepositoryBranchLineDeletionCount(mbc.Metrics.GitRepositoryBranchLineDeletionCount),
		metricGitRepositoryBranchTime:                newMetricGitRepositoryBranchTime(mbc.Metrics.GitRepositoryBranchTime),
		metricGitRepositoryContributorCount:          newMetricGitRepositoryContributorCount(mbc.Metrics.GitRepositoryContributorCount),
		metricGitRepositoryCount:                     newMetricGitRepositoryCount(mbc.Metrics.GitRepositoryCount),
		metricGitRepositoryPullRequestCount:          newMetricGitRepositoryPullRequestCount(mbc.Metrics.GitRepositoryPullRequestCount),
		metricGitRepositoryPullRequestTimeOpen:       newMetricGitRepositoryPullRequestTimeOpen(mbc.Metrics.GitRepositoryPullRequestTimeOpen),
		metricGitRepositoryPullRequestTimeToApproval: newMetricGitRepositoryPullRequestTimeToApproval(mbc.Metrics.GitRepositoryPullRequestTimeToApproval),
		metricGitRepositoryPullRequestTimeToMerge:    newMetricGitRepositoryPullRequestTimeToMerge(mbc.Metrics.GitRepositoryPullRequestTimeToMerge),
		resourceAttributeIncludeFilter:               make(map[string]filter.Filter),
		resourceAttributeExcludeFilter:               make(map[string]filter.Filter),
	}
	if mbc.ResourceAttributes.GitVendorName.MetricsInclude != nil {
		mb.resourceAttributeIncludeFilter["git.vendor.name"] = filter.CreateFilter(mbc.ResourceAttributes.GitVendorName.MetricsInclude)
	}
	if mbc.ResourceAttributes.GitVendorName.MetricsExclude != nil {
		mb.resourceAttributeExcludeFilter["git.vendor.name"] = filter.CreateFilter(mbc.ResourceAttributes.GitVendorName.MetricsExclude)
	}
	if mbc.ResourceAttributes.OrganizationName.MetricsInclude != nil {
		mb.resourceAttributeIncludeFilter["organization.name"] = filter.CreateFilter(mbc.ResourceAttributes.OrganizationName.MetricsInclude)
	}
	if mbc.ResourceAttributes.OrganizationName.MetricsExclude != nil {
		mb.resourceAttributeExcludeFilter["organization.name"] = filter.CreateFilter(mbc.ResourceAttributes.OrganizationName.MetricsExclude)
	}

	for _, op := range options {
		op(mb)
	}
	return mb
}

// NewResourceBuilder returns a new resource builder that should be used to build a resource associated with for the emitted metrics.
func (mb *MetricsBuilder) NewResourceBuilder() *ResourceBuilder {
	return NewResourceBuilder(mb.config.ResourceAttributes)
}

// updateCapacity updates max length of metrics and resource attributes that will be used for the slice capacity.
func (mb *MetricsBuilder) updateCapacity(rm pmetric.ResourceMetrics) {
	if mb.metricsCapacity < rm.ScopeMetrics().At(0).Metrics().Len() {
		mb.metricsCapacity = rm.ScopeMetrics().At(0).Metrics().Len()
	}
}

// ResourceMetricsOption applies changes to provided resource metrics.
type ResourceMetricsOption func(pmetric.ResourceMetrics)

// WithResource sets the provided resource on the emitted ResourceMetrics.
// It's recommended to use ResourceBuilder to create the resource.
func WithResource(res pcommon.Resource) ResourceMetricsOption {
	return func(rm pmetric.ResourceMetrics) {
		res.CopyTo(rm.Resource())
	}
}

// WithStartTimeOverride overrides start time for all the resource metrics data points.
// This option should be only used if different start time has to be set on metrics coming from different resources.
func WithStartTimeOverride(start pcommon.Timestamp) ResourceMetricsOption {
	return func(rm pmetric.ResourceMetrics) {
		var dps pmetric.NumberDataPointSlice
		metrics := rm.ScopeMetrics().At(0).Metrics()
		for i := 0; i < metrics.Len(); i++ {
			switch metrics.At(i).Type() {
			case pmetric.MetricTypeGauge:
				dps = metrics.At(i).Gauge().DataPoints()
			case pmetric.MetricTypeSum:
				dps = metrics.At(i).Sum().DataPoints()
			}
			for j := 0; j < dps.Len(); j++ {
				dps.At(j).SetStartTimestamp(start)
			}
		}
	}
}

// EmitForResource saves all the generated metrics under a new resource and updates the internal state to be ready for
// recording another set of data points as part of another resource. This function can be helpful when one scraper
// needs to emit metrics from several resources. Otherwise calling this function is not required,
// just `Emit` function can be called instead.
// Resource attributes should be provided as ResourceMetricsOption arguments.
func (mb *MetricsBuilder) EmitForResource(rmo ...ResourceMetricsOption) {
	rm := pmetric.NewResourceMetrics()
	rm.SetSchemaUrl(conventions.SchemaURL)
	ils := rm.ScopeMetrics().AppendEmpty()
	ils.Scope().SetName("github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver")
	ils.Scope().SetVersion(mb.buildInfo.Version)
	ils.Metrics().EnsureCapacity(mb.metricsCapacity)
	mb.metricGitRepositoryBranchCommitAheadbyCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchCommitBehindbyCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchLineAdditionCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchLineDeletionCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchTime.emit(ils.Metrics())
	mb.metricGitRepositoryContributorCount.emit(ils.Metrics())
	mb.metricGitRepositoryCount.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestCount.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestTimeOpen.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestTimeToApproval.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestTimeToMerge.emit(ils.Metrics())

	for _, op := range rmo {
		op(rm)
	}
	for attr, filter := range mb.resourceAttributeIncludeFilter {
		if val, ok := rm.Resource().Attributes().Get(attr); ok && !filter.Matches(val.AsString()) {
			return
		}
	}
	for attr, filter := range mb.resourceAttributeExcludeFilter {
		if val, ok := rm.Resource().Attributes().Get(attr); ok && filter.Matches(val.AsString()) {
			return
		}
	}

	if ils.Metrics().Len() > 0 {
		mb.updateCapacity(rm)
		rm.MoveTo(mb.metricsBuffer.ResourceMetrics().AppendEmpty())
	}
}

// Emit returns all the metrics accumulated by the metrics builder and updates the internal state to be ready for
// recording another set of metrics. This function will be responsible for applying all the transformations required to
// produce metric representation defined in metadata and user config, e.g. delta or cumulative.
func (mb *MetricsBuilder) Emit(rmo ...ResourceMetricsOption) pmetric.Metrics {
	mb.EmitForResource(rmo...)
	metrics := mb.metricsBuffer
	mb.metricsBuffer = pmetric.NewMetrics()
	return metrics
}

// RecordGitRepositoryBranchCommitAheadbyCountDataPoint adds a data point to git.repository.branch.commit.aheadby.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchCommitAheadbyCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryBranchCommitAheadbyCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryBranchCommitBehindbyCountDataPoint adds a data point to git.repository.branch.commit.behindby.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchCommitBehindbyCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryBranchCommitBehindbyCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryBranchCountDataPoint adds a data point to git.repository.branch.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string) {
	mb.metricGitRepositoryBranchCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue)
}

// RecordGitRepositoryBranchLineAdditionCountDataPoint adds a data point to git.repository.branch.line.addition.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchLineAdditionCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryBranchLineAdditionCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryBranchLineDeletionCountDataPoint adds a data point to git.repository.branch.line.deletion.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchLineDeletionCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryBranchLineDeletionCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryBranchTimeDataPoint adds a data point to git.repository.branch.time metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchTimeDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryBranchTime.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryContributorCountDataPoint adds a data point to git.repository.contributor.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryContributorCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string) {
	mb.metricGitRepositoryContributorCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue)
}

// RecordGitRepositoryCountDataPoint adds a data point to git.repository.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryCountDataPoint(ts pcommon.Timestamp, val int64) {
	mb.metricGitRepositoryCount.recordDataPoint(mb.startTime, ts, val)
}

// RecordGitRepositoryPullRequestCountDataPoint adds a data point to git.repository.pull_request.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestCountDataPoint(ts pcommon.Timestamp, val int64, pullRequestStateAttributeValue AttributePullRequestState, repositoryNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestCount.recordDataPoint(mb.startTime, ts, val, pullRequestStateAttributeValue.String(), repositoryNameAttributeValue)
}

// RecordGitRepositoryPullRequestTimeOpenDataPoint adds a data point to git.repository.pull_request.time_open metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestTimeOpenDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestTimeOpen.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryPullRequestTimeToApprovalDataPoint adds a data point to git.repository.pull_request.time_to_approval metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestTimeToApprovalDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestTimeToApproval.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryPullRequestTimeToMergeDataPoint adds a data point to git.repository.pull_request.time_to_merge metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestTimeToMergeDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestTimeToMerge.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// Reset resets metrics builder to its initial state. It should be used when external metrics source is restarted,
// and metrics builder should update its startTime and reset it's internal state accordingly.
func (mb *MetricsBuilder) Reset(options ...metricBuilderOption) {
	mb.startTime = pcommon.NewTimestampFromTime(time.Now())
	for _, op := range options {
		op(mb)
	}
}
