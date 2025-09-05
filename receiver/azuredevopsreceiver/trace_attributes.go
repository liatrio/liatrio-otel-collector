// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// getPipelineEventAttrs gets resource attributes for Azure DevOps Pipeline Run State Changed events
func (atr *azuredevopsTracesReceiver) getPipelineEventAttrs(resource pcommon.Resource, event *PipelineRunStateChangedEvent) error {
	attrs := resource.Attributes()

	attrs.PutStr("cicd.pipeline.name", event.Resource.Run.Pipeline.Name)
	attrs.PutInt("cicd.pipeline.id", int64(event.Resource.Run.Pipeline.ID))
	attrs.PutStr("cicd.pipeline.run.state", event.Resource.Run.State)
	attrs.PutStr("cicd.pipeline.run.result", event.Resource.Run.Result)
	attrs.PutStr("cicd.pipeline.run.created_date", event.Resource.Run.CreatedDate.Format(time.RFC3339))
	if event.Resource.Run.FinishedDate != nil {
		attrs.PutStr("cicd.pipeline.run.finished_date", event.Resource.Run.FinishedDate.Format(time.RFC3339))
	}
	attrs.PutStr("cicd.pipeline.run.url", transformAzureDevOpsURL(event.Resource.Run.URL))

	attrs.PutStr("vcs.vendor.name", "azuredevops")

	return nil
}

// getStageEventAttrs gets resource attributes for Azure DevOps Pipeline Stage State Changed events
func (atr *azuredevopsTracesReceiver) getStageEventAttrs(resource pcommon.Resource, event *PipelineStageStateChangedEvent) error {
	attrs := resource.Attributes()

	attrs.PutStr("cicd.pipeline.name", event.Resource.Pipeline.Name)
	attrs.PutInt("cicd.pipeline.id", int64(event.Resource.Pipeline.ID))
	attrs.PutStr("cicd.pipeline.stage.name", event.Resource.Stage.Name)
	attrs.PutStr("cicd.pipeline.stage.display_name", event.Resource.Stage.DisplayName)
	attrs.PutStr("cicd.pipeline.stage.state", event.Resource.Stage.State)
	attrs.PutStr("cicd.pipeline.stage.result", event.Resource.Stage.Result)
	attrs.PutStr("cicd.pipeline.run.created_date", event.Resource.Run.CreatedDate.Format(time.RFC3339))

	// Add repository information if available
	if len(event.Resource.Repositories) > 0 {
		repo := event.Resource.Repositories[0]
		attrs.PutStr("vcs.repository.url.full", repo.URL)
		attrs.PutStr("vcs.repository.type", repo.Type)
		if repo.Change.Author.Name != "" {
			attrs.PutStr("vcs.commit.author.name", repo.Change.Author.Name)
			attrs.PutStr("vcs.commit.author.email", repo.Change.Author.Email)
			attrs.PutStr("vcs.commit.message", repo.Change.Message)
		}
	}

	attrs.PutStr("vcs.vendor.name", "azuredevops")
	attrs.PutStr("azuredevops.project.id", event.ResourceContainers.Project.ID)

	return nil
}

// getJobEventAttrs gets resource attributes for Azure DevOps Pipeline Job State Changed events
func (atr *azuredevopsTracesReceiver) getJobEventAttrs(resource pcommon.Resource, event *PipelineJobStateChangedEvent) error {
	attrs := resource.Attributes()

	attrs.PutStr("cicd.pipeline.name", event.Resource.Pipeline.Name)
	attrs.PutInt("cicd.pipeline.id", int64(event.Resource.Pipeline.ID))
	attrs.PutStr("cicd.pipeline.job.name", event.Resource.Job.Name)
	attrs.PutStr("cicd.pipeline.job.state", event.Resource.Job.State)
	attrs.PutStr("cicd.pipeline.job.result", event.Resource.Job.Result)
	if event.Resource.Job.StartTime != nil {
		attrs.PutStr("cicd.pipeline.job.start_time", event.Resource.Job.StartTime.Format(time.RFC3339))
	}
	if event.Resource.Job.FinishTime != nil {
		attrs.PutStr("cicd.pipeline.job.finish_time", event.Resource.Job.FinishTime.Format(time.RFC3339))
	}
	attrs.PutInt("cicd.pipeline.job.attempt", int64(event.Resource.Job.Attempt))

	attrs.PutStr("cicd.pipeline.stage.name", event.Resource.Stage.Name)
	attrs.PutStr("cicd.pipeline.stage.display_name", event.Resource.Stage.DisplayName)

	attrs.PutStr("cicd.pipeline.run.created_date", event.Resource.Run.CreatedDate.Format(time.RFC3339))

	// Add repository information if available
	if len(event.Resource.Repositories) > 0 {
		repo := event.Resource.Repositories[0]
		attrs.PutStr("vcs.repository.url.full", repo.URL)
		attrs.PutStr("vcs.repository.type", repo.Type)
		if repo.Change.Author.Name != "" {
			attrs.PutStr("vcs.commit.author.name", repo.Change.Author.Name)
			attrs.PutStr("vcs.commit.author.email", repo.Change.Author.Email)
			attrs.PutStr("vcs.commit.message", repo.Change.Message)
			attrs.PutStr("vcs.commit.version", repo.Change.Version)
		}
	}

	attrs.PutStr("vcs.vendor.name", "azuredevops")
	attrs.PutStr("azuredevops.project.id", event.ResourceContainers.Project.ID)

	return nil
}
