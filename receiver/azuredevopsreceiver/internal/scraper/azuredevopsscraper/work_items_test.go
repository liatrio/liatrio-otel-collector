// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"testing"
	"time"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestParseWorkItemTags(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]interface{}
		expected []string
	}{
		{
			name:     "no tags field",
			fields:   map[string]interface{}{},
			expected: nil,
		},
		{
			name:     "single tag",
			fields:   map[string]interface{}{"System.Tags": "P1-Urgent"},
			expected: []string{"P1-Urgent"},
		},
		{
			name:     "multiple tags with spaces",
			fields:   map[string]interface{}{"System.Tags": "P1-Urgent; Bug; Blocked"},
			expected: []string{"P1-Urgent", "Bug", "Blocked"},
		},
		{
			name:     "extra semicolons",
			fields:   map[string]interface{}{"System.Tags": ";P1-Urgent;;Bug;"},
			expected: []string{"P1-Urgent", "Bug"},
		},
		{
			name:     "whitespace only entries",
			fields:   map[string]interface{}{"System.Tags": "  ; P1-Urgent ;  ;  Bug  ; "},
			expected: []string{"P1-Urgent", "Bug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wi := WorkItem{Fields: tt.fields}
			result := parseWorkItemTags(wi)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecordWorkItemMetrics_EmptyAllowlistSkipsTags(t *testing.T) {
	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  "https://dev.azure.com",
		WorkItemsEnabled:         true,
		WorkItemTagAllowlist:     []string{},
		MetricsBuilderConfig:     metadata.NewDefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	now := time.Now()
	ts := pcommon.NewTimestampFromTime(now)

	workItems := []WorkItem{
		{
			ID: 1,
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
				"System.State":        "Active",
				"System.CreatedDate":  now.Add(-24 * time.Hour).Format(time.RFC3339),
				"System.Tags":         "P1-Urgent; Feature; Blocked",
			},
		},
	}

	scraper.recordWorkItemMetrics(ts, workItems, "test-project")
	metrics := scraper.mb.Emit()

	foundCount := false
	foundTagCount := false

	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Name() == "work_item.count" {
					foundCount = true
				}
				if m.Name() == "work_item.tag.count" {
					foundTagCount = true
				}
			}
		}
	}

	assert.True(t, foundCount, "work_item.count should still be emitted")
	assert.False(t, foundTagCount, "work_item.tag.count should NOT be emitted when allowlist is empty")
}

func TestRecordWorkItemMetrics_AllowlistFiltersToOnlyAllowedTags(t *testing.T) {
	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  "https://dev.azure.com",
		WorkItemsEnabled:         true,
		WorkItemTagAllowlist:     []string{"P1-Urgent", "Blocked"},
		MetricsBuilderConfig:     metadata.NewDefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	now := time.Now()
	ts := pcommon.NewTimestampFromTime(now)

	workItems := []WorkItem{
		{
			ID: 1,
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
				"System.State":        "Active",
				"System.CreatedDate":  now.Add(-24 * time.Hour).Format(time.RFC3339),
				"System.Tags":         "P1-Urgent; Feature; Blocked",
			},
		},
		{
			ID: 2,
			Fields: map[string]interface{}{
				"System.WorkItemType": "Task",
				"System.State":        "New",
				"System.CreatedDate":  now.Add(-48 * time.Hour).Format(time.RFC3339),
				"System.Tags":         "P2-High; Tech-Debt",
			},
		},
	}

	scraper.recordWorkItemMetrics(ts, workItems, "test-project")
	metrics := scraper.mb.Emit()

	emittedTags := map[string]bool{}

	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Name() == "work_item.tag.count" {
					dp := m.Gauge().DataPoints()
					for l := 0; l < dp.Len(); l++ {
						tagVal, exists := dp.At(l).Attributes().Get("work_item.tag")
						if exists {
							emittedTags[tagVal.Str()] = true
						}
					}
				}
			}
		}
	}

	assert.True(t, emittedTags["P1-Urgent"], "P1-Urgent should be emitted (in allowlist)")
	assert.True(t, emittedTags["Blocked"], "Blocked should be emitted (in allowlist)")
	assert.False(t, emittedTags["Feature"], "Feature should NOT be emitted (not in allowlist)")
	assert.False(t, emittedTags["P2-High"], "P2-High should NOT be emitted (not in allowlist)")
	assert.False(t, emittedTags["Tech-Debt"], "Tech-Debt should NOT be emitted (not in allowlist)")
}

// TestRecordWorkItemMetrics_DistinctIDsDecollideAgeDatapoints verifies that two
// open work items sharing the same type/state/project produce two distinct
// work_item.age datapoints rather than collapsing into one. Under otel
// v0.156.0 the generated metrics builder merges datapoints with an identical
// attribute set + timestamp (gauges avg-merge), so without work_item.id these
// two items would be averaged into a single lossy datapoint.
func TestRecordWorkItemMetrics_DistinctIDsDecollideAgeDatapoints(t *testing.T) {
	cfg := &Config{
		Organization:             "test-org",
		Project:                  "test-project",
		BaseURL:                  "https://dev.azure.com",
		WorkItemsEnabled:         true,
		MetricsBuilderConfig:     metadata.NewDefaultMetricsBuilderConfig(),
		ResourceAttributesConfig: metadata.DefaultResourceAttributesConfig(),
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	scraper := newAzureDevOpsScraper(context.Background(), settings, cfg)

	now := time.Now()
	ts := pcommon.NewTimestampFromTime(now)

	// Two open Bugs in the same state and project — identical on every
	// attribute except work_item.id.
	workItems := []WorkItem{
		{
			ID: 10,
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
				"System.State":        "Active",
				"System.CreatedDate":  now.Add(-24 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			ID: 11,
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
				"System.State":        "Active",
				"System.CreatedDate":  now.Add(-48 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	scraper.recordWorkItemMetrics(ts, workItems, "test-project")
	metrics := scraper.mb.Emit()

	ids := map[string]bool{}
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				if m.Name() != "work_item.age" {
					continue
				}
				dp := m.Gauge().DataPoints()
				for l := 0; l < dp.Len(); l++ {
					if idVal, ok := dp.At(l).Attributes().Get("work_item.id"); ok {
						ids[idVal.Str()] = true
					}
				}
			}
		}
	}

	assert.Len(t, ids, 2, "work_item.age should emit a distinct datapoint per work item (keyed by work_item.id)")
	assert.True(t, ids["10"] && ids["11"], "both work item ids should be present as distinct datapoints")
}
