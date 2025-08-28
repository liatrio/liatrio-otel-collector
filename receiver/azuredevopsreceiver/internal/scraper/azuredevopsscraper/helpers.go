// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"time"
)

// AzureDevOpsProject represents a project in Azure DevOps
type AzureDevOpsProject struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	Revision       int       `json:"revision"`
	Visibility     string    `json:"visibility"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

// AzureDevOpsRepository represents a Git repository in Azure DevOps
type AzureDevOpsRepository struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	WebURL        string `json:"webUrl"`
	DefaultBranch string `json:"defaultBranch"`
	Size          int64  `json:"size"`
	RemoteURL     string `json:"remoteUrl"`
	SSHURL        string `json:"sshUrl"`
	Project       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
}

// AzureDevOpsBranch represents a Git branch in Azure DevOps
type AzureDevOpsBranch struct {
	Name        string    `json:"name"`
	ObjectID    string    `json:"objectId"`
	CreatedDate time.Time `json:"createdDate"`
}

// AzureDevOpsPullRequest represents a pull request in Azure DevOps
type AzureDevOpsPullRequest struct {
	PullRequestID int    `json:"pullRequestId"`
	CodeReviewID  int    `json:"codeReviewId"`
	Status        string `json:"status"`
	CreatedBy     struct {
		DisplayName string `json:"displayName"`
		ID          string `json:"id"`
	} `json:"createdBy"`
	CreationDate    time.Time `json:"creationDate"`
	ClosedDate      time.Time `json:"closedDate"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	SourceRefName   string    `json:"sourceRefName"`
	TargetRefName   string    `json:"targetRefName"`
	MergeStatus     string    `json:"mergeStatus"`
	IsDraft         bool      `json:"isDraft"`
	MergeID         string    `json:"mergeId"`
	LastMergeCommit struct {
		CommitID string `json:"commitId"`
		URL      string `json:"url"`
	} `json:"lastMergeCommit"`
	Repository AzureDevOpsRepository `json:"repository"`
}

// AzureDevOpsCommit represents a Git commit in Azure DevOps
type AzureDevOpsCommit struct {
	CommitID string `json:"commitId"`
	Author   struct {
		Name  string    `json:"name"`
		Email string    `json:"email"`
		Date  time.Time `json:"date"`
	} `json:"author"`
	Committer struct {
		Name  string    `json:"name"`
		Email string    `json:"email"`
		Date  time.Time `json:"date"`
	} `json:"committer"`
	Comment   string `json:"comment"`
	URL       string `json:"url"`
	RemoteURL string `json:"remoteUrl"`
}
