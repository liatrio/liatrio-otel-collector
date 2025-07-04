// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/filter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	conventions "go.opentelemetry.io/collector/semconv/v1.27.0"
)

// AttributeVcsChangeState specifies the value vcs.change.state attribute.
type AttributeVcsChangeState int

const (
	_ AttributeVcsChangeState = iota
	AttributeVcsChangeStateOpen
	AttributeVcsChangeStateMerged
)

// String returns the string representation of the AttributeVcsChangeState.
func (av AttributeVcsChangeState) String() string {
	switch av {
	case AttributeVcsChangeStateOpen:
		return "open"
	case AttributeVcsChangeStateMerged:
		return "merged"
	}
	return ""
}

// MapAttributeVcsChangeState is a helper map of string to AttributeVcsChangeState attribute value.
var MapAttributeVcsChangeState = map[string]AttributeVcsChangeState{
	"open":   AttributeVcsChangeStateOpen,
	"merged": AttributeVcsChangeStateMerged,
}

// AttributeVcsLineChangeType specifies the value vcs.line_change.type attribute.
type AttributeVcsLineChangeType int

const (
	_ AttributeVcsLineChangeType = iota
	AttributeVcsLineChangeTypeAdded
	AttributeVcsLineChangeTypeRemoved
)

// String returns the string representation of the AttributeVcsLineChangeType.
func (av AttributeVcsLineChangeType) String() string {
	switch av {
	case AttributeVcsLineChangeTypeAdded:
		return "added"
	case AttributeVcsLineChangeTypeRemoved:
		return "removed"
	}
	return ""
}

// MapAttributeVcsLineChangeType is a helper map of string to AttributeVcsLineChangeType attribute value.
var MapAttributeVcsLineChangeType = map[string]AttributeVcsLineChangeType{
	"added":   AttributeVcsLineChangeTypeAdded,
	"removed": AttributeVcsLineChangeTypeRemoved,
}

// AttributeVcsRefHeadType specifies the value vcs.ref.head.type attribute.
type AttributeVcsRefHeadType int

const (
	_ AttributeVcsRefHeadType = iota
	AttributeVcsRefHeadTypeBranch
	AttributeVcsRefHeadTypeTag
)

// String returns the string representation of the AttributeVcsRefHeadType.
func (av AttributeVcsRefHeadType) String() string {
	switch av {
	case AttributeVcsRefHeadTypeBranch:
		return "branch"
	case AttributeVcsRefHeadTypeTag:
		return "tag"
	}
	return ""
}

// MapAttributeVcsRefHeadType is a helper map of string to AttributeVcsRefHeadType attribute value.
var MapAttributeVcsRefHeadType = map[string]AttributeVcsRefHeadType{
	"branch": AttributeVcsRefHeadTypeBranch,
	"tag":    AttributeVcsRefHeadTypeTag,
}

// AttributeVcsRevisionDeltaDirection specifies the value vcs.revision_delta.direction attribute.
type AttributeVcsRevisionDeltaDirection int

const (
	_ AttributeVcsRevisionDeltaDirection = iota
	AttributeVcsRevisionDeltaDirectionAhead
	AttributeVcsRevisionDeltaDirectionBehind
)

// String returns the string representation of the AttributeVcsRevisionDeltaDirection.
func (av AttributeVcsRevisionDeltaDirection) String() string {
	switch av {
	case AttributeVcsRevisionDeltaDirectionAhead:
		return "ahead"
	case AttributeVcsRevisionDeltaDirectionBehind:
		return "behind"
	}
	return ""
}

// MapAttributeVcsRevisionDeltaDirection is a helper map of string to AttributeVcsRevisionDeltaDirection attribute value.
var MapAttributeVcsRevisionDeltaDirection = map[string]AttributeVcsRevisionDeltaDirection{
	"ahead":  AttributeVcsRevisionDeltaDirectionAhead,
	"behind": AttributeVcsRevisionDeltaDirectionBehind,
}

type metricVcsChangeCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.change.count metric with initial data.
func (m *metricVcsChangeCount) init() {
	m.data.SetName("vcs.change.count")
	m.data.SetDescription("The number of changes (pull requests) in a repository, categorized by their state (either open or merged).")
	m.data.SetUnit("{change}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsChangeCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsChangeStateAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.change.state", vcsChangeStateAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsChangeCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsChangeCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsChangeCount(cfg MetricConfig) metricVcsChangeCount {
	m := metricVcsChangeCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsChangeDuration struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.change.duration metric with initial data.
func (m *metricVcsChangeDuration) init() {
	m.data.SetName("vcs.change.duration")
	m.data.SetDescription("The time duration a change (pull request/merge request/changelist) has been in an open state.")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsChangeDuration) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsChangeStateAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.name", vcsRefHeadNameAttributeValue)
	dp.Attributes().PutStr("vcs.change.state", vcsChangeStateAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsChangeDuration) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsChangeDuration) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsChangeDuration(cfg MetricConfig) metricVcsChangeDuration {
	m := metricVcsChangeDuration{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsChangeTimeToApproval struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.change.time_to_approval metric with initial data.
func (m *metricVcsChangeTimeToApproval) init() {
	m.data.SetName("vcs.change.time_to_approval")
	m.data.SetDescription("The amount of time it took a change (pull request) to go from open to approved.")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsChangeTimeToApproval) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.name", vcsRefHeadNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsChangeTimeToApproval) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsChangeTimeToApproval) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsChangeTimeToApproval(cfg MetricConfig) metricVcsChangeTimeToApproval {
	m := metricVcsChangeTimeToApproval{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsChangeTimeToMerge struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.change.time_to_merge metric with initial data.
func (m *metricVcsChangeTimeToMerge) init() {
	m.data.SetName("vcs.change.time_to_merge")
	m.data.SetDescription("The amount of time it took a change (pull request) to go from open to merged.")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsChangeTimeToMerge) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.name", vcsRefHeadNameAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsChangeTimeToMerge) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsChangeTimeToMerge) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsChangeTimeToMerge(cfg MetricConfig) metricVcsChangeTimeToMerge {
	m := metricVcsChangeTimeToMerge{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsContributorCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.contributor.count metric with initial data.
func (m *metricVcsContributorCount) init() {
	m.data.SetName("vcs.contributor.count")
	m.data.SetDescription("The number of unique contributors to a repository.")
	m.data.SetUnit("{contributor}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsContributorCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsContributorCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsContributorCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsContributorCount(cfg MetricConfig) metricVcsContributorCount {
	m := metricVcsContributorCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsRefCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.ref.count metric with initial data.
func (m *metricVcsRefCount) init() {
	m.data.SetName("vcs.ref.count")
	m.data.SetDescription("The number of refs of type branch in a repository.")
	m.data.SetUnit("{ref}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsRefCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadTypeAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.type", vcsRefHeadTypeAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsRefCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsRefCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsRefCount(cfg MetricConfig) metricVcsRefCount {
	m := metricVcsRefCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsRefLinesDelta struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.ref.lines_delta metric with initial data.
func (m *metricVcsRefLinesDelta) init() {
	m.data.SetName("vcs.ref.lines_delta")
	m.data.SetDescription("The number of lines added/removed in a ref (branch) relative to the default branch (trunk).")
	m.data.SetUnit("{line}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsRefLinesDelta) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsRefHeadTypeAttributeValue string, vcsLineChangeTypeAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.name", vcsRefHeadNameAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.type", vcsRefHeadTypeAttributeValue)
	dp.Attributes().PutStr("vcs.line_change.type", vcsLineChangeTypeAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsRefLinesDelta) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsRefLinesDelta) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsRefLinesDelta(cfg MetricConfig) metricVcsRefLinesDelta {
	m := metricVcsRefLinesDelta{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsRefRevisionsDelta struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.ref.revisions_delta metric with initial data.
func (m *metricVcsRefRevisionsDelta) init() {
	m.data.SetName("vcs.ref.revisions_delta")
	m.data.SetDescription("The number of revisions (commits) a ref (branch) is ahead/behind the branch from trunk (default).")
	m.data.SetUnit("{revision}")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsRefRevisionsDelta) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsRefHeadTypeAttributeValue string, vcsRevisionDeltaDirectionAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.name", vcsRefHeadNameAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.type", vcsRefHeadTypeAttributeValue)
	dp.Attributes().PutStr("vcs.revision_delta.direction", vcsRevisionDeltaDirectionAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsRefRevisionsDelta) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsRefRevisionsDelta) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsRefRevisionsDelta(cfg MetricConfig) metricVcsRefRevisionsDelta {
	m := metricVcsRefRevisionsDelta{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsRefTime struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.ref.time metric with initial data.
func (m *metricVcsRefTime) init() {
	m.data.SetName("vcs.ref.time")
	m.data.SetDescription("Time a ref (branch) created from the default branch (trunk) has existed. The `vcs.ref.head.type` attribute will always be `branch`.")
	m.data.SetUnit("s")
	m.data.SetEmptyGauge()
	m.data.Gauge().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricVcsRefTime) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsRefHeadTypeAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.url.full", vcsRepositoryURLFullAttributeValue)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("vcs.repository.id", vcsRepositoryIDAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.name", vcsRefHeadNameAttributeValue)
	dp.Attributes().PutStr("vcs.ref.head.type", vcsRefHeadTypeAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsRefTime) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsRefTime) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsRefTime(cfg MetricConfig) metricVcsRefTime {
	m := metricVcsRefTime{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricVcsRepositoryCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills vcs.repository.count metric with initial data.
func (m *metricVcsRepositoryCount) init() {
	m.data.SetName("vcs.repository.count")
	m.data.SetDescription("The number of repositories in an organization.")
	m.data.SetUnit("{repository}")
	m.data.SetEmptyGauge()
}

func (m *metricVcsRepositoryCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Gauge().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricVcsRepositoryCount) updateCapacity() {
	if m.data.Gauge().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Gauge().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricVcsRepositoryCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Gauge().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricVcsRepositoryCount(cfg MetricConfig) metricVcsRepositoryCount {
	m := metricVcsRepositoryCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

// MetricsBuilder provides an interface for scrapers to report metrics while taking care of all the transformations
// required to produce metric representation defined in metadata and user config.
type MetricsBuilder struct {
	config                         MetricsBuilderConfig // config of the metrics builder.
	startTime                      pcommon.Timestamp    // start time that will be applied to all recorded data points.
	metricsCapacity                int                  // maximum observed number of metrics per resource.
	metricsBuffer                  pmetric.Metrics      // accumulates metrics data before emitting.
	buildInfo                      component.BuildInfo  // contains version information.
	resourceAttributeIncludeFilter map[string]filter.Filter
	resourceAttributeExcludeFilter map[string]filter.Filter
	metricVcsChangeCount           metricVcsChangeCount
	metricVcsChangeDuration        metricVcsChangeDuration
	metricVcsChangeTimeToApproval  metricVcsChangeTimeToApproval
	metricVcsChangeTimeToMerge     metricVcsChangeTimeToMerge
	metricVcsContributorCount      metricVcsContributorCount
	metricVcsRefCount              metricVcsRefCount
	metricVcsRefLinesDelta         metricVcsRefLinesDelta
	metricVcsRefRevisionsDelta     metricVcsRefRevisionsDelta
	metricVcsRefTime               metricVcsRefTime
	metricVcsRepositoryCount       metricVcsRepositoryCount
}

// MetricBuilderOption applies changes to default metrics builder.
type MetricBuilderOption interface {
	apply(*MetricsBuilder)
}

type metricBuilderOptionFunc func(mb *MetricsBuilder)

func (mbof metricBuilderOptionFunc) apply(mb *MetricsBuilder) {
	mbof(mb)
}

// WithStartTime sets startTime on the metrics builder.
func WithStartTime(startTime pcommon.Timestamp) MetricBuilderOption {
	return metricBuilderOptionFunc(func(mb *MetricsBuilder) {
		mb.startTime = startTime
	})
}
func NewMetricsBuilder(mbc MetricsBuilderConfig, settings receiver.Settings, options ...MetricBuilderOption) *MetricsBuilder {
	mb := &MetricsBuilder{
		config:                         mbc,
		startTime:                      pcommon.NewTimestampFromTime(time.Now()),
		metricsBuffer:                  pmetric.NewMetrics(),
		buildInfo:                      settings.BuildInfo,
		metricVcsChangeCount:           newMetricVcsChangeCount(mbc.Metrics.VcsChangeCount),
		metricVcsChangeDuration:        newMetricVcsChangeDuration(mbc.Metrics.VcsChangeDuration),
		metricVcsChangeTimeToApproval:  newMetricVcsChangeTimeToApproval(mbc.Metrics.VcsChangeTimeToApproval),
		metricVcsChangeTimeToMerge:     newMetricVcsChangeTimeToMerge(mbc.Metrics.VcsChangeTimeToMerge),
		metricVcsContributorCount:      newMetricVcsContributorCount(mbc.Metrics.VcsContributorCount),
		metricVcsRefCount:              newMetricVcsRefCount(mbc.Metrics.VcsRefCount),
		metricVcsRefLinesDelta:         newMetricVcsRefLinesDelta(mbc.Metrics.VcsRefLinesDelta),
		metricVcsRefRevisionsDelta:     newMetricVcsRefRevisionsDelta(mbc.Metrics.VcsRefRevisionsDelta),
		metricVcsRefTime:               newMetricVcsRefTime(mbc.Metrics.VcsRefTime),
		metricVcsRepositoryCount:       newMetricVcsRepositoryCount(mbc.Metrics.VcsRepositoryCount),
		resourceAttributeIncludeFilter: make(map[string]filter.Filter),
		resourceAttributeExcludeFilter: make(map[string]filter.Filter),
	}
	if mbc.ResourceAttributes.OrganizationName.MetricsInclude != nil {
		mb.resourceAttributeIncludeFilter["organization.name"] = filter.CreateFilter(mbc.ResourceAttributes.OrganizationName.MetricsInclude)
	}
	if mbc.ResourceAttributes.OrganizationName.MetricsExclude != nil {
		mb.resourceAttributeExcludeFilter["organization.name"] = filter.CreateFilter(mbc.ResourceAttributes.OrganizationName.MetricsExclude)
	}
	if mbc.ResourceAttributes.VcsVendorName.MetricsInclude != nil {
		mb.resourceAttributeIncludeFilter["vcs.vendor.name"] = filter.CreateFilter(mbc.ResourceAttributes.VcsVendorName.MetricsInclude)
	}
	if mbc.ResourceAttributes.VcsVendorName.MetricsExclude != nil {
		mb.resourceAttributeExcludeFilter["vcs.vendor.name"] = filter.CreateFilter(mbc.ResourceAttributes.VcsVendorName.MetricsExclude)
	}

	for _, op := range options {
		op.apply(mb)
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
type ResourceMetricsOption interface {
	apply(pmetric.ResourceMetrics)
}

type resourceMetricsOptionFunc func(pmetric.ResourceMetrics)

func (rmof resourceMetricsOptionFunc) apply(rm pmetric.ResourceMetrics) {
	rmof(rm)
}

// WithResource sets the provided resource on the emitted ResourceMetrics.
// It's recommended to use ResourceBuilder to create the resource.
func WithResource(res pcommon.Resource) ResourceMetricsOption {
	return resourceMetricsOptionFunc(func(rm pmetric.ResourceMetrics) {
		res.CopyTo(rm.Resource())
	})
}

// WithStartTimeOverride overrides start time for all the resource metrics data points.
// This option should be only used if different start time has to be set on metrics coming from different resources.
func WithStartTimeOverride(start pcommon.Timestamp) ResourceMetricsOption {
	return resourceMetricsOptionFunc(func(rm pmetric.ResourceMetrics) {
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
	})
}

// EmitForResource saves all the generated metrics under a new resource and updates the internal state to be ready for
// recording another set of data points as part of another resource. This function can be helpful when one scraper
// needs to emit metrics from several resources. Otherwise calling this function is not required,
// just `Emit` function can be called instead.
// Resource attributes should be provided as ResourceMetricsOption arguments.
func (mb *MetricsBuilder) EmitForResource(options ...ResourceMetricsOption) {
	rm := pmetric.NewResourceMetrics()
	rm.SetSchemaUrl(conventions.SchemaURL)
	ils := rm.ScopeMetrics().AppendEmpty()
	ils.Scope().SetName(ScopeName)
	ils.Scope().SetVersion(mb.buildInfo.Version)
	ils.Metrics().EnsureCapacity(mb.metricsCapacity)
	mb.metricVcsChangeCount.emit(ils.Metrics())
	mb.metricVcsChangeDuration.emit(ils.Metrics())
	mb.metricVcsChangeTimeToApproval.emit(ils.Metrics())
	mb.metricVcsChangeTimeToMerge.emit(ils.Metrics())
	mb.metricVcsContributorCount.emit(ils.Metrics())
	mb.metricVcsRefCount.emit(ils.Metrics())
	mb.metricVcsRefLinesDelta.emit(ils.Metrics())
	mb.metricVcsRefRevisionsDelta.emit(ils.Metrics())
	mb.metricVcsRefTime.emit(ils.Metrics())
	mb.metricVcsRepositoryCount.emit(ils.Metrics())

	for _, op := range options {
		op.apply(rm)
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
func (mb *MetricsBuilder) Emit(options ...ResourceMetricsOption) pmetric.Metrics {
	mb.EmitForResource(options...)
	metrics := mb.metricsBuffer
	mb.metricsBuffer = pmetric.NewMetrics()
	return metrics
}

// RecordVcsChangeCountDataPoint adds a data point to vcs.change.count metric.
func (mb *MetricsBuilder) RecordVcsChangeCountDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsChangeStateAttributeValue AttributeVcsChangeState, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string) {
	mb.metricVcsChangeCount.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsChangeStateAttributeValue.String(), vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue)
}

// RecordVcsChangeDurationDataPoint adds a data point to vcs.change.duration metric.
func (mb *MetricsBuilder) RecordVcsChangeDurationDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsChangeStateAttributeValue AttributeVcsChangeState) {
	mb.metricVcsChangeDuration.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue, vcsRefHeadNameAttributeValue, vcsChangeStateAttributeValue.String())
}

// RecordVcsChangeTimeToApprovalDataPoint adds a data point to vcs.change.time_to_approval metric.
func (mb *MetricsBuilder) RecordVcsChangeTimeToApprovalDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string) {
	mb.metricVcsChangeTimeToApproval.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue, vcsRefHeadNameAttributeValue)
}

// RecordVcsChangeTimeToMergeDataPoint adds a data point to vcs.change.time_to_merge metric.
func (mb *MetricsBuilder) RecordVcsChangeTimeToMergeDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string) {
	mb.metricVcsChangeTimeToMerge.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue, vcsRefHeadNameAttributeValue)
}

// RecordVcsContributorCountDataPoint adds a data point to vcs.contributor.count metric.
func (mb *MetricsBuilder) RecordVcsContributorCountDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string) {
	mb.metricVcsContributorCount.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue)
}

// RecordVcsRefCountDataPoint adds a data point to vcs.ref.count metric.
func (mb *MetricsBuilder) RecordVcsRefCountDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadTypeAttributeValue AttributeVcsRefHeadType) {
	mb.metricVcsRefCount.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue, vcsRefHeadTypeAttributeValue.String())
}

// RecordVcsRefLinesDeltaDataPoint adds a data point to vcs.ref.lines_delta metric.
func (mb *MetricsBuilder) RecordVcsRefLinesDeltaDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsRefHeadTypeAttributeValue AttributeVcsRefHeadType, vcsLineChangeTypeAttributeValue AttributeVcsLineChangeType) {
	mb.metricVcsRefLinesDelta.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue, vcsRefHeadNameAttributeValue, vcsRefHeadTypeAttributeValue.String(), vcsLineChangeTypeAttributeValue.String())
}

// RecordVcsRefRevisionsDeltaDataPoint adds a data point to vcs.ref.revisions_delta metric.
func (mb *MetricsBuilder) RecordVcsRefRevisionsDeltaDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsRefHeadTypeAttributeValue AttributeVcsRefHeadType, vcsRevisionDeltaDirectionAttributeValue AttributeVcsRevisionDeltaDirection) {
	mb.metricVcsRefRevisionsDelta.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue, vcsRefHeadNameAttributeValue, vcsRefHeadTypeAttributeValue.String(), vcsRevisionDeltaDirectionAttributeValue.String())
}

// RecordVcsRefTimeDataPoint adds a data point to vcs.ref.time metric.
func (mb *MetricsBuilder) RecordVcsRefTimeDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryURLFullAttributeValue string, vcsRepositoryNameAttributeValue string, vcsRepositoryIDAttributeValue string, vcsRefHeadNameAttributeValue string, vcsRefHeadTypeAttributeValue AttributeVcsRefHeadType) {
	mb.metricVcsRefTime.recordDataPoint(mb.startTime, ts, val, vcsRepositoryURLFullAttributeValue, vcsRepositoryNameAttributeValue, vcsRepositoryIDAttributeValue, vcsRefHeadNameAttributeValue, vcsRefHeadTypeAttributeValue.String())
}

// RecordVcsRepositoryCountDataPoint adds a data point to vcs.repository.count metric.
func (mb *MetricsBuilder) RecordVcsRepositoryCountDataPoint(ts pcommon.Timestamp, val int64) {
	mb.metricVcsRepositoryCount.recordDataPoint(mb.startTime, ts, val)
}

// Reset resets metrics builder to its initial state. It should be used when external metrics source is restarted,
// and metrics builder should update its startTime and reset it's internal state accordingly.
func (mb *MetricsBuilder) Reset(options ...MetricBuilderOption) {
	mb.startTime = pcommon.NewTimestampFromTime(time.Now())
	for _, op := range options {
		op.apply(mb)
	}
}
