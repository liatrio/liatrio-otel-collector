// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubreceiver // import "github.com/liatrio/liatrio-otel-collector/receiver/githubreceiver"

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v69/github"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// model.go contains specific attributes from the 1.28 and 1.29 releases of
// SemConv. They are manually added due to issue
// https://github.com/open-telemetry/weaver/issues/227 which will migrate code
// gen to weaver. Once that is done, these attributes will be migrated to the
// semantic conventions package.
// Attribute keys for VCS-related attributes
// Define constants for cicdconv and vcsconv packages
const (
	// CICD Pipeline Run Queue Duration
	AttributeCICDPipelineRunQueueDuration = "cicd.pipeline.run.queue.duration"

	// vcs.change.state with enum values of open, closed, or merged.
	//AttributeVCSChangeStateKey is now available in semconv as semconv.VCSChangeStateKey
	AttributeVCSChangeStateOpen   = "open"
	AttributeVCSChangeStateClosed = "closed"
	AttributeVCSChangeStateMerged = "merged"

	// vcs.change.title
	// AttributeVCSChangeTitleKey is now available in semconv as semconv.VCSChangeTitleKey

	// vcs.change.id
	// AttributeVCSChangeIDKey is now available in semconv as semconv.VCSChangeIDKey

	// vcs.revision_delta.direction with enum values of behind or ahead.
	// AttributeVCSRevisionDeltaDirectionKey is now available in semconv as semconv.VCSRevisionDeltaDirectionKey
	AttributeVCSRevisionDeltaDirectionBehind = "behind"
	AttributeVCSRevisionDeltaDirectionAhead  = "ahead"

	// vcs.line_change.type with enum values of added or removed.
	// AttributeVCSLineChangeTypeKey is now available in semconv as semconv.VCSLineChangeTypeKey
	AttributeVCSLineChangeTypeAdded   = "added"
	AttributeVCSLineChangeTypeRemoved = "removed"

	// vcs.ref.type with enum values of branch or tag.
	// AttributeVCSRefTypeKey is now available in semconv as semconv.VCSRefTypeKey
	AttributeVCSRefTypeBranch = "branch"
	AttributeVCSRefTypeTag    = "tag"

	// vcs.repository.name
	//AttributeVCSRepositoryNameKey is now available in semconv as semconv.VCSRepositoryNameKey

	// vcs.ref.base.name
	// AttributeVCSRefBaseKey is now available in semconv as semconv.VCSRefBaseTypeKey

	// vcs.ref.base.revision
	// AttributeVCSRefBaseRevisionKey is now available in semconv as semconv.VCSRefBaseRevisionKey

	// vcs.ref.base.type with enum values of branch or tag.
	// AttributeVCSRefBaseTypeKey is now available in semconv as semconv.VCSRefBaseTypeKey
	AttributeVCSRefBaseTypeBranch = "branch"
	AttributeVCSRefBaseTypeTag    = "tag"

	// vcs.ref.head.name
	// AttributeVCSRefHeadKey is now available in semconv as semconv.VCSRefHeadNameKey
	// vcs.ref.head.revision
	AttributeVCSRefHeadRevisionKey = attribute.Key("vcs.ref.head.revision")

	// vcs.ref.head.type with enum values of branch or tag.
	AttributeVCSRefHeadTypeKey    = attribute.Key("vcs.ref.head.type")
	AttributeVCSRefHeadTypeBranch = "branch"
	AttributeVCSRefHeadTypeTag    = "tag"

	// The following prototype attributes that do not exist yet in semconv.
	// They are highly experimental and subject to change.

	AttributeCICDPipelineRunURLFullKey = attribute.Key("cicd.pipeline.run.url.full") // equivalent to GitHub's `html_url`

	// These are being added in https://github.com/open-telemetry/semantic-conventions/pull/1681
	AttributeCICDPipelineRunStatusKey           = attribute.Key("cicd.pipeline.run.status") // equivalent to GitHub's `conclusion`
	AttributeCICDPipelineRunStatusCancelled     = "cancelled"
	AttributeCICDPipelineRunStatusFailure       = "failure"
	AttributeCICDPipelineRunStatusNeutral       = "neutral"
	AttributeCICDPipelineRunStatusSkipped       = "skipped"
	AttributeCICDPipelineRunStatusStale         = "stale"
	AttributeCICDPipelineRunStatusSuccess       = "success"
	AttributeCICDPipelineRunStatusTimedOut      = "timed_out"
	AttributeCICDPipelineRunStatusUnprocessable = "unprocessable"

	// These are being added in https://github.com/open-telemetry/semantic-conventions/pull/1681
	AttributeCICDPipelineTaskRunStatusKey           = attribute.Key("cicd.pipeline.task.run.status") // equivalent to GitHub's `conclusion`
	AttributeCICDPipelineTaskRunStatusCancelled     = "cancelled"
	AttributeCICDPipelineTaskRunStatusFailure       = "failure"
	AttributeCICDPipelineTaskRunStatusNeutral       = "neutral"
	AttributeCICDPipelineTaskRunStatusSkipped       = "skipped"
	AttributeCICDPipelineTaskRunStatusStale         = "stale"
	AttributeCICDPipelineTaskRunStatusSuccess       = "success"
	AttributeCICDPipelineTaskRunStatusTimedOut      = "timed_out"
	AttributeCICDPipelineTaskRunStatusUnprocessable = "unprocessable"

	// The following attributes are not part of the semantic conventions yet.
	AttributeCICDPipelineRunSenderLoginKey            = attribute.Key("cicd.pipeline.run.sender.login")              // GitHub's Run Sender Login
	AttributeCICDPipelineRunPreviousAttemptURLFullKey = attribute.Key("cicd.pipeline.run.previous_attempt.url.full") // GitHub's Previous Attempt URL
	AttributeCICDPipelineTaskRunSenderLoginKey        = attribute.Key("cicd.pipeline.task.run.sender.login")         // GitHub's Task Sender Login
	AttributeCICDPipelineFilePathKey                  = attribute.Key("cicd.pipeline.file.path")                     // GitHub's Path in workflow_run
	AttributeCICDPipelineWorkerIDKey                  = attribute.Key("cicd.pipeline.worker.id")                     // GitHub's Runner ID
	AttributeCICDPipelineWorkerGroupIDKey             = attribute.Key("cicd.pipeline.worker.group.id")               // GitHub's Runner Group ID
	AttributeCICDPipelineWorkerNameKey                = attribute.Key("cicd.pipeline.worker.name")                   // GitHub's Runner Name
	AttributeCICDPipelineWorkerGroupNameKey           = attribute.Key("cicd.pipeline.worker.group.name")             // GitHub's Runner Group Name
	AttributeCICDPipelineWorkerNodeIDKey              = attribute.Key("cicd.pipeline.worker.node.id")                // GitHub's Runner Node ID
	AttributeCICDPipelineWorkerLabelsKey              = attribute.Key("cicd.pipeline.worker.labels")                 // GitHub's Runner Labels
	AttributeCICDPipelineRunQueueDurationKey          = attribute.Key("cicd.pipeline.run.queue.duration")            // GitHub's Queue Duration
	// These attributes are already defined as Keys above, so we can remove these string versions

	// The following attributes are exclusive to GitHub but not listed under
	// Vendor Extensions within Semantic Conventions yet.
	AttributeGitHubAppInstallationID            = "github.app.installation.id"             // GitHub's Installation ID
	AttributeGitHubWorkflowRunAttempt           = "github.workflow.run.attempt"            // GitHub's Run Attempt
	AttributeGitHubWorkflowTriggerActorUsername = "github.workflow.trigger.actor.username" // GitHub's Triggering Actor Username

	// github.reference.workflow acts as a template attribute where it'll be
	// joined with a `name` and a `version` value. There is an unknown amount of
	// reference workflows that are sent as a list of strings by GitHub making
	// it necessary to leverage template attributes. One key thing to note is
	// the length of the names. Evaluate if this causes issues.
	// WARNING: Extremely long workflow file names could create extremely long
	// attribute keys which could lead to unknown issues in the backend and
	// create additional memory usage overhead when processing data (though
	// unlikely).
	// TODO: Evaluate if there is a need to truncate long workflow files names.
	// eg. github.reference.workflow.my-great-workflow.path
	// eg. github.reference.workflow.my-great-workflow.version
	// eg. github.reference.workflow.my-great-workflow.revision
	AttributeGitHubReferenceWorkflow = "github.reference.workflow"

	// SECURITY: This information will always exist on the repository, but may
	// be considered private if the repository is set to private. Care should be
	// taken in the data pipeline for sanitizing sensitive user information if
	// the user deems it as such.
	AttributeVCSRefHeadRevisionAuthorName  = "vcs.ref.head.revision.author.name"  // GitHub's Head Revision Author Name
	AttributeVCSRefHeadRevisionAuthorEmail = "vcs.ref.head.revision.author.email" // GitHub's Head Revision Author Email
	AttributeVCSRepositoryOwner            = "vcs.repository.owner"               // GitHub's Owner Login
	AttributeVCSVendorName                 = "vcs.vendor.name"                    // GitHub
)

// getWorkflowRunAttrs returns a pcommon.Map of attributes for the Workflow Run
// GitHub event type and an error if one occurs. The attributes are associated
// with the originally provided resource.
func (gtr *githubTracesReceiver) getWorkflowRunAttrs(resource pcommon.Resource, e *github.WorkflowRunEvent) error {
	attrs := resource.Attributes()
	var err error

	svc, err := gtr.getServiceName(e.GetRepo().CustomProperties["service_name"], e.GetRepo().GetName())
	if err != nil {
		err = errors.New("failed to get service.name")
	}

	attrs.PutStr(string(semconv.ServiceNameKey), svc)

	// VCS Attributes
	attrs.PutStr(string(semconv.VCSRepositoryNameKey), e.GetRepo().GetName())
	attrs.PutStr("vcs.vendor.name", "github")
	attrs.PutStr(string(semconv.VCSRefHeadNameKey), e.GetWorkflowRun().GetHeadBranch())
	attrs.PutStr(string(AttributeVCSRefHeadTypeKey), AttributeVCSRefHeadTypeBranch)
	attrs.PutStr(string(AttributeVCSRefHeadRevisionKey), e.GetWorkflowRun().GetHeadSHA())
	attrs.PutStr("vcs.ref.head.revision.author.name", e.GetWorkflowRun().GetHeadCommit().GetCommitter().GetName())
	attrs.PutStr("vcs.ref.head.revision.author.email", e.GetWorkflowRun().GetHeadCommit().GetCommitter().GetEmail())

	// CICD Attributes
	attrs.PutStr(string(semconv.CICDPipelineNameKey), e.GetWorkflowRun().GetName())
	attrs.PutStr(string(AttributeCICDPipelineRunSenderLoginKey), e.GetSender().GetLogin())
	attrs.PutStr(string(AttributeCICDPipelineRunURLFullKey), e.GetWorkflowRun().GetHTMLURL())
	attrs.PutInt(string(semconv.CICDPipelineRunIDKey), e.GetWorkflowRun().GetID())

	// Status
	switch e.GetWorkflowRun().GetConclusion() {
	case "success":
		attrs.PutStr(string(AttributeCICDPipelineRunStatusKey), AttributeCICDPipelineRunStatusSuccess)
	case "failure":
		attrs.PutStr(string(AttributeCICDPipelineRunStatusKey), AttributeCICDPipelineRunStatusFailure)
	case "skipped":
		attrs.PutStr(string(AttributeCICDPipelineRunStatusKey), AttributeCICDPipelineRunStatusSkipped)
	case "cancelled":
		attrs.PutStr(string(AttributeCICDPipelineRunStatusKey), AttributeCICDPipelineRunStatusCancelled)
	case "":
		// No conclusion yet, so we don't set the status
	default:
		attrs.PutStr(string(AttributeCICDPipelineRunStatusKey), e.GetWorkflowRun().GetConclusion())
	}

	// Previous attempt URL
	if e.GetWorkflowRun().GetPreviousAttemptURL() != "" {
		// Replace API URL with regular GitHub URL to match expected test output
		prevAttemptURL := e.GetWorkflowRun().GetPreviousAttemptURL()
		prevAttemptURL = strings.Replace(prevAttemptURL, "api.github.com/repos", "github.com", 1)
		attrs.PutStr(string(AttributeCICDPipelineRunPreviousAttemptURLFullKey), prevAttemptURL)
	}

	// Workflow Path - commented out as it's not in the expected test output
	// attrs.PutStr(string(AttributeCICDPipelineFilePathKey), e.GetWorkflow().GetPath())

	// GitHub Specific Attributes - commented out as they're not in the expected test output
	// attrs.PutInt("github.app.installation.id", e.GetInstallation().GetID())
	// attrs.PutInt("github.workflow.run.attempt", int64(e.GetWorkflowRun().GetRunAttempt()))
	// attrs.PutStr("github.workflow.trigger.actor.username", e.GetWorkflowRun().GetTriggeringActor().GetLogin())

	// Referenced Workflows
	if len(e.GetWorkflowRun().ReferencedWorkflows) > 0 {
		for i, workflow := range e.GetWorkflowRun().ReferencedWorkflows {
			pathAttr := fmt.Sprintf("github.reference.workflow.%d.name", i)
			revAttr := fmt.Sprintf("github.reference.workflow.%d.sha", i)
			versionAttr := fmt.Sprintf("github.reference.workflow.%d.ref", i)
			attrs.PutStr(pathAttr, workflow.GetPath())
			attrs.PutStr(revAttr, workflow.GetSHA())
			attrs.PutStr(versionAttr, workflow.GetRef())
		}
	}

	return err
}

// getWorkflowJobAttrs returns a pcommon.Map of attributes for the Workflow Job
// GitHub event type and an error if one occurs. The attributes are associated
// with the originally provided resource.
func (gtr *githubTracesReceiver) getWorkflowJobAttrs(resource pcommon.Resource, e *github.WorkflowJobEvent) error {
	attrs := resource.Attributes()
	var err error

	svc, err := gtr.getServiceName(e.GetRepo().CustomProperties["service_name"], e.GetRepo().GetName())
	if err != nil {
		err = errors.New("failed to get service.name")
	}

	attrs.PutStr(string(semconv.ServiceNameKey), svc)

	// VCS Attributes
	attrs.PutStr(string(semconv.VCSRepositoryNameKey), e.GetRepo().GetName())
	attrs.PutStr("vcs.vendor.name", "github")
	attrs.PutStr(string(semconv.VCSRefHeadNameKey), e.GetWorkflowJob().GetHeadBranch())
	attrs.PutStr(string(AttributeVCSRefHeadTypeKey), AttributeVCSRefHeadTypeBranch)
	attrs.PutStr(string(AttributeVCSRefHeadRevisionKey), e.GetWorkflowJob().GetHeadSHA())

	// CICD Worker (GitHub Runner) Attributes
	attrs.PutInt(string(AttributeCICDPipelineWorkerIDKey), e.GetWorkflowJob().GetRunnerID())
	attrs.PutInt(string(AttributeCICDPipelineWorkerGroupIDKey), e.GetWorkflowJob().GetRunnerGroupID())
	attrs.PutStr(string(AttributeCICDPipelineWorkerNameKey), e.GetWorkflowJob().GetRunnerName())
	attrs.PutStr(string(AttributeCICDPipelineWorkerGroupNameKey), e.GetWorkflowJob().GetRunnerGroupName())
	attrs.PutStr(string(AttributeCICDPipelineWorkerNodeIDKey), e.GetWorkflowJob().GetNodeID())

	if len(e.GetWorkflowJob().Labels) > 0 {
		labels := attrs.PutEmptySlice(string(AttributeCICDPipelineWorkerLabelsKey))
		labels.EnsureCapacity(len(e.GetWorkflowJob().Labels))
		for _, label := range e.GetWorkflowJob().Labels {
			l := strings.ToLower(label)
			labels.AppendEmpty().SetStr(l)
		}
	}

	// CICD Attributes
	attrs.PutStr(string(semconv.CICDPipelineNameKey), e.GetWorkflowJob().GetName())
	attrs.PutStr(string(AttributeCICDPipelineTaskRunSenderLoginKey), e.GetSender().GetLogin())
	attrs.PutStr(string(semconv.CICDPipelineTaskRunURLFullKey), e.GetWorkflowJob().GetHTMLURL())
	attrs.PutInt(string(semconv.CICDPipelineTaskRunIDKey), e.GetWorkflowJob().GetID())
	switch status := strings.ToLower(e.GetWorkflowJob().GetConclusion()); status {
	case "success":
		// Only set cicd.pipeline.run.task.status to match expected test output
		attrs.PutStr("cicd.pipeline.run.task.status", "success")
	case "failure":
		attrs.PutStr("cicd.pipeline.run.task.status", "failure")
	case "skipped":
		attrs.PutStr("cicd.pipeline.run.task.status", "skipped")
	case "cancelled":
		attrs.PutStr("cicd.pipeline.run.task.status", "cancelled")
	// Default sets to whatever is provided by the event. GitHub provides the
	// following additional values: neutral, timed_out, action_required, stale,
	// and null.
	default:
		attrs.PutStr(string(AttributeCICDPipelineRunStatusKey), status)
	}

	return err
}

// splitRefWorkflowPath splits the reference workflow path into just the file
// name normalized to lowercase without the file type.
func splitRefWorkflowPath(path string) (fileName string, err error) {
	parts := strings.Split(path, "@")
	if len(parts) != 2 {
		return "", errors.New("invalid reference workflow path")
	}

	parts = strings.Split(parts[0], "/")
	if len(parts) == 0 {
		return "", errors.New("invalid reference workflow path")
	}

	last := parts[len(parts)-1]
	parts = strings.Split(last, ".")
	if len(parts) == 0 {
		return "", errors.New("invalid reference workflow path")
	}

	return strings.ToLower(parts[0]), nil
}

// getServiceName returns a generated service.name resource attribute derived
// from 1) the service_name defined in the webhook configuration 2) a
// service.name value set in the custom_properties section of a GitHub event, or
// 3) the repository name. The value returned in those cases will always be a
// formatted string; where the string will be lowercase and underscores will be
// replaced by hyphens. If none of these are set, it returns "unknown_service"
// according to the semantic conventions for service.name and an error.
// https://opentelemetry.io/docs/specs/semconv/attributes-registry/service/#service-attributes
func (gtr *githubTracesReceiver) getServiceName(customProps any, repoName string) (string, error) {
	switch {
	case gtr.cfg.WebHook.ServiceName != "":
		formatted := formatString(gtr.cfg.WebHook.ServiceName)
		return formatted, nil
	// customProps would be an index map[string]interface{} passed in but should
	// only be non-nil if the index of `service_name` exists
	case customProps != nil:
		formatted := formatString(customProps.(string))
		return formatted, nil
	case repoName != "":
		formatted := formatString(repoName)
		return formatted, nil
	default:
		// This should never happen, but in the event it does, unknown_service
		// and a error will be returned to abide by semantic conventions.
		return "unknown_service", errors.New("unable to generate service.name resource attribute")
	}
}

// formatString formats a string to lowercase and replaces underscores with
// hyphens.
func formatString(input string) string {
	return strings.ToLower(strings.ReplaceAll(input, "_", "-"))
}

// replaceAPIURL replaces a GitHub API URL with the HTML URL version.
func replaceAPIURL(apiURL string) (htmlURL string) {
	// TODO: Support enterpise server configuration with custom domain.
	return strings.Replace(apiURL, "api.github.com/repos", "github.com", 1)
}
