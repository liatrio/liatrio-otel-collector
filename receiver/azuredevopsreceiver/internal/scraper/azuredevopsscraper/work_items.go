// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// fetchWorkItems fetches all work items from Azure DevOps using WIQL query
func (ados *azuredevopsScraper) fetchWorkItems(ctx context.Context, org, project string, lookbackDays int) ([]WorkItem, error) {
	// Default to 30 days if not specified
	if lookbackDays <= 0 {
		lookbackDays = 30
	}

	// Build WIQL query - fetch all work items modified in the lookback period
	// No type filtering; filtering can be done in downstream processors or backend
	wiql := fmt.Sprintf(`SELECT [System.Id] FROM WorkItems WHERE [System.ChangedDate] >= @StartOfDay('-%dd')`,
		lookbackDays,
	)

	ados.logger.Sugar().Debugf("Fetching work items with query: %s", wiql)

	// Execute WIQL query to get work item IDs
	workItemIDs, err := ados.executeWIQLQuery(ctx, org, project, wiql)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WIQL query: %w", err)
	}

	if len(workItemIDs) == 0 {
		ados.logger.Sugar().Info("No work items found")
		return []WorkItem{}, nil
	}

	ados.logger.Sugar().Infof("Found %d work items, fetching details", len(workItemIDs))

	// Batch fetch work item details (API supports up to 200 IDs per request)
	var allWorkItems []WorkItem
	batchSize := 200

	for i := 0; i < len(workItemIDs); i += batchSize {
		end := i + batchSize
		if end > len(workItemIDs) {
			end = len(workItemIDs)
		}

		batch := workItemIDs[i:end]
		workItems, err := ados.getWorkItemsBatch(ctx, org, project, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to get work items batch: %w", err)
		}

		allWorkItems = append(allWorkItems, workItems...)

		// Small delay between batches to avoid rate limiting
		if end < len(workItemIDs) {
			time.Sleep(200 * time.Millisecond)
		}
	}

	ados.logger.Sugar().Infof("Fetched %d work items", len(allWorkItems))
	return allWorkItems, nil
}

// executeWIQLQuery executes a WIQL query and returns work item IDs
func (ados *azuredevopsScraper) executeWIQLQuery(ctx context.Context, org, project, wiql string) ([]int, error) {
	urlPath := fmt.Sprintf("%s/%s/%s/_apis/wit/wiql", ados.cfg.BaseURL, org, project)

	params := url.Values{}
	params.Set("api-version", apiVersion)

	queryBody := map[string]string{
		"query": wiql,
	}

	body, err := json.Marshal(queryBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urlPath+"?"+params.Encode(), strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ados.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WIQL query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("WIQL query failed with status %d (unable to read response body: %w)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("WIQL query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkItemQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode WIQL response: %w", err)
	}

	workItemIDs := make([]int, len(result.WorkItems))
	for i, wi := range result.WorkItems {
		workItemIDs[i] = wi.ID
	}

	return workItemIDs, nil
}

// getWorkItemsBatch fetches a batch of work items by their IDs
func (ados *azuredevopsScraper) getWorkItemsBatch(ctx context.Context, org, project string, ids []int) ([]WorkItem, error) {
	if len(ids) == 0 {
		return []WorkItem{}, nil
	}

	// Convert IDs to comma-separated string
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = fmt.Sprintf("%d", id)
	}
	idsParam := strings.Join(idStrings, ",")

	urlPath := fmt.Sprintf("%s/%s/%s/_apis/wit/workitems", ados.cfg.BaseURL, org, project)

	params := url.Values{}
	params.Set("ids", idsParam)
	params.Set("api-version", apiVersion)
	params.Set("fields", "System.Id,System.WorkItemType,System.State,System.CreatedDate,Microsoft.VSTS.Common.ClosedDate,System.Title")

	req, err := http.NewRequestWithContext(ctx, "GET", urlPath+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ados.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get work items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("get work items failed with status %d (unable to read response body: %w)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("get work items failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkItemBatchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode work items response: %w", err)
	}

	return result.Value, nil
}

// getWorkItemField safely extracts a field value from a work item
func getWorkItemField(wi WorkItem, fieldName string) (interface{}, bool) {
	val, ok := wi.Fields[fieldName]
	return val, ok
}

// getWorkItemStringField safely extracts a string field from a work item
func getWorkItemStringField(wi WorkItem, fieldName string) string {
	val, ok := getWorkItemField(wi, fieldName)
	if !ok {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// getWorkItemTimeField safely extracts a time field from a work item
func getWorkItemTimeField(wi WorkItem, fieldName string) (time.Time, bool) {
	val, ok := getWorkItemField(wi, fieldName)
	if !ok {
		return time.Time{}, false
	}
	if str, ok := val.(string); ok {
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	return time.Time{}, false
}

// recordWorkItemMetrics processes work items and records metrics
func (ados *azuredevopsScraper) recordWorkItemMetrics(nowTimestamp pcommon.Timestamp, workItems []WorkItem, project string) {
	now := nowTimestamp.AsTime()
	// Track counts by type and state
	counts := make(map[string]map[string]int) // type -> state -> count

	for _, wi := range workItems {
		workItemType := getWorkItemStringField(wi, "System.WorkItemType")
		state := getWorkItemStringField(wi, "System.State")

		if workItemType == "" || state == "" {
			continue
		}

		// Initialize map if needed
		if counts[workItemType] == nil {
			counts[workItemType] = make(map[string]int)
		}
		counts[workItemType][state]++

		// Get created date
		createdDate, hasCreated := getWorkItemTimeField(wi, "System.CreatedDate")
		if !hasCreated {
			continue
		}

		// Check if work item is closed
		closedDate, isClosed := getWorkItemTimeField(wi, "Microsoft.VSTS.Common.ClosedDate")

		if isClosed {
			// Record cycle time for closed items
			cycleTime := closedDate.Sub(createdDate).Seconds()
			ados.mb.RecordWorkItemCycleTimeDataPoint(
				nowTimestamp,
				int64(cycleTime),
				workItemType,
				project,
			)
		} else {
			// Record age for open items
			age := now.Sub(createdDate).Seconds()
			ados.mb.RecordWorkItemAgeDataPoint(
				nowTimestamp,
				int64(age),
				workItemType,
				state,
				project,
			)
		}
	}

	// Record counts
	for workItemType, states := range counts {
		for state, count := range states {
			ados.mb.RecordWorkItemCountDataPoint(
				nowTimestamp,
				int64(count),
				workItemType,
				state,
				project,
			)
		}
	}

	ados.logger.Sugar().Infof("Recorded work item metrics: %d items across %d types", len(workItems), len(counts))
}
