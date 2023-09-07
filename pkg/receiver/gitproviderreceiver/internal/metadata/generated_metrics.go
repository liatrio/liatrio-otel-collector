// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
)

type metricGitRepositoryBranchAdditionCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.addition.count metric with initial data.
func (m *metricGitRepositoryBranchAdditionCount) init() {
	m.data.SetName("git.repository.branch.addition.count")
	m.data.SetDescription("Total additional lines of code in the branch")
	m.data.SetUnit("1")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchAdditionCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
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
func (m *metricGitRepositoryBranchAdditionCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchAdditionCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchAdditionCount(cfg MetricConfig) metricGitRepositoryBranchAdditionCount {
	m := metricGitRepositoryBranchAdditionCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryBranchCommitAheadbyCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.commit.aheadby.count metric with initial data.
func (m *metricGitRepositoryBranchCommitAheadbyCount) init() {
	m.data.SetName("git.repository.branch.commit.aheadby.count")
	m.data.SetDescription("Number of commits the branch is ahead of the default branch")
	m.data.SetUnit("1")
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
	m.data.SetDescription("Number of commits the branch is behind the default branch")
	m.data.SetUnit("1")
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
	m.data.SetDescription("Number of branches that exist in the repository")
	m.data.SetUnit("1")
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

type metricGitRepositoryBranchDeletionCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.branch.deletion.count metric with initial data.
func (m *metricGitRepositoryBranchDeletionCount) init() {
	m.data.SetName("git.repository.branch.deletion.count")
	m.data.SetDescription("Total deleted lines of code in the branch")
	m.data.SetUnit("1")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryBranchDeletionCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
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
func (m *metricGitRepositoryBranchDeletionCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryBranchDeletionCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryBranchDeletionCount(cfg MetricConfig) metricGitRepositoryBranchDeletionCount {
	m := metricGitRepositoryBranchDeletionCount{config: cfg}
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
	m.data.SetUnit("1")
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
	m.data.SetDescription("Total number of unique contributors to this repository")
	m.data.SetUnit("1")
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
	m.data.SetDescription("Number of repositories that exist in an organization")
	m.data.SetUnit("1")
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

type metricGitRepositoryPullRequestApprovalTime struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.approval.time metric with initial data.
func (m *metricGitRepositoryPullRequestApprovalTime) init() {
	m.data.SetName("git.repository.pull_request.approval.time")
	m.data.SetDescription("Time for the PR to be approved")
	m.data.SetUnit("1")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestApprovalTime) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
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
func (m *metricGitRepositoryPullRequestApprovalTime) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestApprovalTime) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestApprovalTime(cfg MetricConfig) metricGitRepositoryPullRequestApprovalTime {
	m := metricGitRepositoryPullRequestApprovalTime{config: cfg}
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
	m.data.SetDescription("The amount of open pull requests")
	m.data.SetUnit("1")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string) {
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

type metricGitRepositoryPullRequestDeploymentTime struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.deployment.time metric with initial data.
func (m *metricGitRepositoryPullRequestDeploymentTime) init() {
	m.data.SetName("git.repository.pull_request.deployment.time")
	m.data.SetDescription("Time for the merged PR to be deployed")
	m.data.SetUnit("1")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestDeploymentTime) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
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
func (m *metricGitRepositoryPullRequestDeploymentTime) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestDeploymentTime) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestDeploymentTime(cfg MetricConfig) metricGitRepositoryPullRequestDeploymentTime {
	m := metricGitRepositoryPullRequestDeploymentTime{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryPullRequestMergeTime struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.merge.time metric with initial data.
func (m *metricGitRepositoryPullRequestMergeTime) init() {
	m.data.SetName("git.repository.pull_request.merge.time")
	m.data.SetDescription("Time the PR has been merged")
	m.data.SetUnit("1")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestMergeTime) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
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
func (m *metricGitRepositoryPullRequestMergeTime) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestMergeTime) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestMergeTime(cfg MetricConfig) metricGitRepositoryPullRequestMergeTime {
	m := metricGitRepositoryPullRequestMergeTime{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricGitRepositoryPullRequestTime struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills git.repository.pull_request.time metric with initial data.
func (m *metricGitRepositoryPullRequestTime) init() {
	m.data.SetName("git.repository.pull_request.time")
	m.data.SetDescription("Time the PR has been open")
	m.data.SetUnit("1")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricGitRepositoryPullRequestTime) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
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
func (m *metricGitRepositoryPullRequestTime) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricGitRepositoryPullRequestTime) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricGitRepositoryPullRequestTime(cfg MetricConfig) metricGitRepositoryPullRequestTime {
	m := metricGitRepositoryPullRequestTime{config: cfg}
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
	metricGitRepositoryBranchAdditionCount       metricGitRepositoryBranchAdditionCount
	metricGitRepositoryBranchCommitAheadbyCount  metricGitRepositoryBranchCommitAheadbyCount
	metricGitRepositoryBranchCommitBehindbyCount metricGitRepositoryBranchCommitBehindbyCount
	metricGitRepositoryBranchCount               metricGitRepositoryBranchCount
	metricGitRepositoryBranchDeletionCount       metricGitRepositoryBranchDeletionCount
	metricGitRepositoryBranchTime                metricGitRepositoryBranchTime
	metricGitRepositoryContributorCount          metricGitRepositoryContributorCount
	metricGitRepositoryCount                     metricGitRepositoryCount
	metricGitRepositoryPullRequestApprovalTime   metricGitRepositoryPullRequestApprovalTime
	metricGitRepositoryPullRequestCount          metricGitRepositoryPullRequestCount
	metricGitRepositoryPullRequestDeploymentTime metricGitRepositoryPullRequestDeploymentTime
	metricGitRepositoryPullRequestMergeTime      metricGitRepositoryPullRequestMergeTime
	metricGitRepositoryPullRequestTime           metricGitRepositoryPullRequestTime
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
		config:                                 mbc,
		startTime:                              pcommon.NewTimestampFromTime(time.Now()),
		metricsBuffer:                          pmetric.NewMetrics(),
		buildInfo:                              settings.BuildInfo,
		metricGitRepositoryBranchAdditionCount: newMetricGitRepositoryBranchAdditionCount(mbc.Metrics.GitRepositoryBranchAdditionCount),
		metricGitRepositoryBranchCommitAheadbyCount:  newMetricGitRepositoryBranchCommitAheadbyCount(mbc.Metrics.GitRepositoryBranchCommitAheadbyCount),
		metricGitRepositoryBranchCommitBehindbyCount: newMetricGitRepositoryBranchCommitBehindbyCount(mbc.Metrics.GitRepositoryBranchCommitBehindbyCount),
		metricGitRepositoryBranchCount:               newMetricGitRepositoryBranchCount(mbc.Metrics.GitRepositoryBranchCount),
		metricGitRepositoryBranchDeletionCount:       newMetricGitRepositoryBranchDeletionCount(mbc.Metrics.GitRepositoryBranchDeletionCount),
		metricGitRepositoryBranchTime:                newMetricGitRepositoryBranchTime(mbc.Metrics.GitRepositoryBranchTime),
		metricGitRepositoryContributorCount:          newMetricGitRepositoryContributorCount(mbc.Metrics.GitRepositoryContributorCount),
		metricGitRepositoryCount:                     newMetricGitRepositoryCount(mbc.Metrics.GitRepositoryCount),
		metricGitRepositoryPullRequestApprovalTime:   newMetricGitRepositoryPullRequestApprovalTime(mbc.Metrics.GitRepositoryPullRequestApprovalTime),
		metricGitRepositoryPullRequestCount:          newMetricGitRepositoryPullRequestCount(mbc.Metrics.GitRepositoryPullRequestCount),
		metricGitRepositoryPullRequestDeploymentTime: newMetricGitRepositoryPullRequestDeploymentTime(mbc.Metrics.GitRepositoryPullRequestDeploymentTime),
		metricGitRepositoryPullRequestMergeTime:      newMetricGitRepositoryPullRequestMergeTime(mbc.Metrics.GitRepositoryPullRequestMergeTime),
		metricGitRepositoryPullRequestTime:           newMetricGitRepositoryPullRequestTime(mbc.Metrics.GitRepositoryPullRequestTime),
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
	ils.Scope().SetName("otelcol/gitproviderreceiver")
	ils.Scope().SetVersion(mb.buildInfo.Version)
	ils.Metrics().EnsureCapacity(mb.metricsCapacity)
	mb.metricGitRepositoryBranchAdditionCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchCommitAheadbyCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchCommitBehindbyCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchDeletionCount.emit(ils.Metrics())
	mb.metricGitRepositoryBranchTime.emit(ils.Metrics())
	mb.metricGitRepositoryContributorCount.emit(ils.Metrics())
	mb.metricGitRepositoryCount.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestApprovalTime.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestCount.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestDeploymentTime.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestMergeTime.emit(ils.Metrics())
	mb.metricGitRepositoryPullRequestTime.emit(ils.Metrics())

	for _, op := range rmo {
		op(rm)
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

// RecordGitRepositoryBranchAdditionCountDataPoint adds a data point to git.repository.branch.addition.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchAdditionCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryBranchAdditionCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
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

// RecordGitRepositoryBranchDeletionCountDataPoint adds a data point to git.repository.branch.deletion.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryBranchDeletionCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryBranchDeletionCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
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

// RecordGitRepositoryPullRequestApprovalTimeDataPoint adds a data point to git.repository.pull_request.approval.time metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestApprovalTimeDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestApprovalTime.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryPullRequestCountDataPoint adds a data point to git.repository.pull_request.count metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestCountDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestCount.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue)
}

// RecordGitRepositoryPullRequestDeploymentTimeDataPoint adds a data point to git.repository.pull_request.deployment.time metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestDeploymentTimeDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestDeploymentTime.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryPullRequestMergeTimeDataPoint adds a data point to git.repository.pull_request.merge.time metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestMergeTimeDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestMergeTime.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// RecordGitRepositoryPullRequestTimeDataPoint adds a data point to git.repository.pull_request.time metric.
func (mb *MetricsBuilder) RecordGitRepositoryPullRequestTimeDataPoint(ts pcommon.Timestamp, val int64, repositoryNameAttributeValue string, branchNameAttributeValue string) {
	mb.metricGitRepositoryPullRequestTime.recordDataPoint(mb.startTime, ts, val, repositoryNameAttributeValue, branchNameAttributeValue)
}

// Reset resets metrics builder to its initial state. It should be used when external metrics source is restarted,
// and metrics builder should update its startTime and reset it's internal state accordingly.
func (mb *MetricsBuilder) Reset(options ...metricBuilderOption) {
	mb.startTime = pcommon.NewTimestampFromTime(time.Now())
	for _, op := range options {
		op(mb)
	}
}
