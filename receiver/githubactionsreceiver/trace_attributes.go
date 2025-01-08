// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v67/github"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func createResourceAttributes(resource pcommon.Resource, event interface{}, config *Config, logger *zap.Logger) {
	attrs := resource.Attributes()

	switch e := event.(type) {
	case *github.WorkflowJobEvent:
		var sn string
		if e.GetRepo().CustomProperties["service_name"] != nil {
			sn = e.GetRepo().CustomProperties["service_name"].(string)
		} else {
			sn = generateServiceName(config, e.GetRepo().GetName())
		}

		attrs.PutStr("service.name", sn)

		attrs.PutStr("cicd.pipeline.name", e.GetWorkflowJob().GetWorkflowName())

		attrs.PutStr("cicd.pipeline.task.created_at", e.GetWorkflowJob().GetCreatedAt().Format(time.RFC3339))
		attrs.PutStr("cicd.pipeline.task.completed_at", e.GetWorkflowJob().GetCompletedAt().Format(time.RFC3339))
		attrs.PutStr("cicd.pipeline.task.conclusion", e.GetWorkflowJob().GetConclusion())
		attrs.PutStr("cicd.pipeline.task.head_branch", e.GetWorkflowJob().GetHeadBranch())
		attrs.PutStr("cicd.pipeline.task.head_sha", e.GetWorkflowJob().GetHeadSHA())
		attrs.PutStr("cicd.pipeline.task.html_url", e.GetWorkflowJob().GetHTMLURL())

		if len(e.WorkflowJob.Labels) > 0 {
			labels := e.GetWorkflowJob().Labels
			for i, label := range labels {
				labels[i] = strings.ToLower(label)
			}
			sort.Strings(labels)
			joinedLabels := strings.Join(labels, ",")
			attrs.PutStr("cicd.pipeline.task.labels", joinedLabels)
		} else {
			attrs.PutStr("cicd.pipeline.task.labels", "no labels")
		}

		attrs.PutStr("cicd.pipeline.task.name", e.GetWorkflowJob().GetName())
		attrs.PutInt("cicd.pipeline.task.run.id", e.GetWorkflowJob().GetRunID())
		attrs.PutStr("cicd.pipeline.task.runner.group.name", e.GetWorkflowJob().GetRunnerGroupName())
		attrs.PutStr("cicd.pipeline.task.runner.name", e.GetWorkflowJob().GetRunnerName())
		attrs.PutStr("cicd.pipeline.task.sender.login", e.GetSender().GetLogin())
		attrs.PutStr("cicd.pipeline.task.started_at", e.GetWorkflowJob().GetStartedAt().Format(time.RFC3339))
		attrs.PutStr("cicd.pipeline.task.status", e.GetWorkflowJob().GetStatus())

		attrs.PutStr("vcs.vendor.name", "github")

		attrs.PutStr("vcs.repository.owner.login", e.GetRepo().GetOwner().GetLogin())
		attrs.PutStr("vcs.repository.name", e.GetRepo().GetName())
		attrs.PutStr("vcs.repository.url.full", e.GetRepo().GetURL())

	case *github.WorkflowRunEvent:
		var sn string
		if e.GetRepo().CustomProperties["service_name"] != nil {
			sn = e.GetRepo().CustomProperties["service_name"].(string)
		} else {
			sn = generateServiceName(config, e.GetRepo().GetName())
		}

		attrs.PutStr("service.name", sn)

		attrs.PutStr("cicd.pipeline.run.actor.login", e.GetWorkflowRun().GetActor().GetLogin())

		attrs.PutStr("cicd.pipeline.run.conclusion", e.GetWorkflowRun().GetConclusion())
		attrs.PutStr("cicd.pipeline.run.created_at", e.GetWorkflowRun().GetCreatedAt().Format(time.RFC3339))
		attrs.PutStr("cicd.pipeline.run.display_title", e.GetWorkflowRun().GetDisplayTitle())
		attrs.PutStr("cicd.pipeline.run.event", e.GetWorkflowRun().GetEvent())
		attrs.PutStr("cicd.pipeline.run.head_branch", e.GetWorkflowRun().GetHeadBranch())
		attrs.PutStr("cicd.pipeline.run.head_sha", e.GetWorkflowRun().GetHeadSHA())
		attrs.PutStr("cicd.pipeline.run.html_url", e.GetWorkflowRun().GetHTMLURL())
		attrs.PutInt("cicd.pipeline.run.id", e.GetWorkflowRun().GetID())
		attrs.PutStr("cicd.pipeline.run.name", e.GetWorkflowRun().GetName())
		attrs.PutStr("cicd.pipeline.run.path", e.GetWorkflow().GetPath())

		if e.GetWorkflowRun().GetPreviousAttemptURL() != "" {
			htmlURL := transformGitHubAPIURL(e.GetWorkflowRun().GetPreviousAttemptURL())
			attrs.PutStr("cicd.pipeline.run.previous_attempt_url", htmlURL)
		}

		if len(e.GetWorkflowRun().ReferencedWorkflows) > 0 {
			var referencedWorkflows []string
			for _, workflow := range e.GetWorkflowRun().ReferencedWorkflows {
				referencedWorkflows = append(referencedWorkflows, workflow.GetPath())
			}
			attrs.PutStr("cicd.pipeline.run.referenced_workflows", strings.Join(referencedWorkflows, ";"))
		}

		attrs.PutInt("cicd.pipeline.run.run_attempt", int64(e.GetWorkflowRun().GetRunAttempt()))
		attrs.PutStr("cicd.pipeline.run.run_started_at", e.GetWorkflowRun().RunStartedAt.Format(time.RFC3339))
		attrs.PutStr("cicd.pipeline.run.status", e.GetWorkflowRun().GetStatus())
		attrs.PutStr("cicd.pipeline.run.sender.login", e.GetSender().GetLogin())
		attrs.PutStr("cicd.pipeline.run.triggering_actor.login", e.GetWorkflowRun().GetTriggeringActor().GetLogin())
		attrs.PutStr("cicd.pipeline.run.updated_at", e.GetWorkflowRun().GetUpdatedAt().Format(time.RFC3339))

		attrs.PutStr("vcs.vendor.name", "github")

		attrs.PutStr("vcs.ref.head_branch", e.GetWorkflowRun().GetHeadBranch())
		attrs.PutStr("vcs.ref.head_commit.author.email", e.GetWorkflowRun().GetHeadCommit().GetAuthor().GetEmail())
		attrs.PutStr("vcs.ref.head_commit.author.name", e.GetWorkflowRun().GetHeadCommit().GetAuthor().GetName())
		attrs.PutStr("vcs.ref.head_commit.committer.email", e.GetWorkflowRun().GetHeadCommit().GetCommitter().GetEmail())
		attrs.PutStr("vcs.ref.head_commit.committer.name", e.GetWorkflowRun().GetHeadCommit().GetCommitter().GetName())
		attrs.PutStr("vcs.ref.head_commit.message", e.GetWorkflowRun().GetHeadCommit().GetMessage())
		attrs.PutStr("vcs.ref.head_commit.timestamp", e.GetWorkflowRun().GetHeadCommit().GetTimestamp().Format(time.RFC3339))
		attrs.PutStr("vcs.ref.head_sha", e.GetWorkflowRun().GetHeadSHA())

		if len(e.GetWorkflowRun().PullRequests) > 0 {
			var prUrls []string
			for _, pr := range e.GetWorkflowRun().PullRequests {
				prUrls = append(prUrls, convertPRURL(pr.GetURL()))
			}
			attrs.PutStr("vcs.change.url", strings.Join(prUrls, ";"))
		}

		attrs.PutStr("vcs.repository.name", e.GetRepo().GetName())

	default:
		logger.Error("unknown event type")
	}
}
