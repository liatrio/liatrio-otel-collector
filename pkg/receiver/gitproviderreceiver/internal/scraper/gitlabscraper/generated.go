// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package gitlabscraper

import (
	"context"
	"time"

	"github.com/Khan/genqlient/graphql"
)

// MergeRequestNode includes the requested fields of the GraphQL type MergeRequest.
type MergeRequestNode struct {
	// Internal ID of the merge request.
	Iid string `json:"iid"`
	// Title of the merge request.
	Title string `json:"title"`
	// Source branch of the merge request.
	SourceBranch string `json:"sourceBranch"`
	// Target branch of the merge request.
	TargetBranch string `json:"targetBranch"`
	// Timestamp of when the merge request was created.
	CreatedAt time.Time `json:"createdAt"`
	// Timestamp of when the merge request was merged, null if not merged.
	MergedAt time.Time `json:"mergedAt"`
	// Alias for target_project.
	Project MergeRequestNodeProject `json:"project"`
}

// GetIid returns MergeRequestNode.Iid, and is useful for accessing the field via an interface.
func (v *MergeRequestNode) GetIid() string { return v.Iid }

// GetTitle returns MergeRequestNode.Title, and is useful for accessing the field via an interface.
func (v *MergeRequestNode) GetTitle() string { return v.Title }

// GetSourceBranch returns MergeRequestNode.SourceBranch, and is useful for accessing the field via an interface.
func (v *MergeRequestNode) GetSourceBranch() string { return v.SourceBranch }

// GetTargetBranch returns MergeRequestNode.TargetBranch, and is useful for accessing the field via an interface.
func (v *MergeRequestNode) GetTargetBranch() string { return v.TargetBranch }

// GetCreatedAt returns MergeRequestNode.CreatedAt, and is useful for accessing the field via an interface.
func (v *MergeRequestNode) GetCreatedAt() time.Time { return v.CreatedAt }

// GetMergedAt returns MergeRequestNode.MergedAt, and is useful for accessing the field via an interface.
func (v *MergeRequestNode) GetMergedAt() time.Time { return v.MergedAt }

// GetProject returns MergeRequestNode.Project, and is useful for accessing the field via an interface.
func (v *MergeRequestNode) GetProject() MergeRequestNodeProject { return v.Project }

// MergeRequestNodeProject includes the requested fields of the GraphQL type Project.
type MergeRequestNodeProject struct {
	// Full path of the project.
	FullPath string `json:"fullPath"`
}

// GetFullPath returns MergeRequestNodeProject.FullPath, and is useful for accessing the field via an interface.
func (v *MergeRequestNodeProject) GetFullPath() string { return v.FullPath }

// State of a GitLab merge request
type MergeRequestState string

const (
	// Merge request has been merged.
	MergeRequestStateMerged MergeRequestState = "merged"
	// In open state.
	MergeRequestStateOpened MergeRequestState = "opened"
	// In closed state.
	MergeRequestStateClosed MergeRequestState = "closed"
	// Discussion has been locked.
	MergeRequestStateLocked MergeRequestState = "locked"
	// All available.
	MergeRequestStateAll MergeRequestState = "all"
)

// __getAllGroupProjectsInput is used internally by genqlient
type __getAllGroupProjectsInput struct {
	FullPath string  `json:"fullPath"`
	After    *string `json:"after"`
}

// GetFullPath returns __getAllGroupProjectsInput.FullPath, and is useful for accessing the field via an interface.
func (v *__getAllGroupProjectsInput) GetFullPath() string { return v.FullPath }

// GetAfter returns __getAllGroupProjectsInput.After, and is useful for accessing the field via an interface.
func (v *__getAllGroupProjectsInput) GetAfter() *string { return v.After }

// __getBranchNamesInput is used internally by genqlient
type __getBranchNamesInput struct {
	FullPath string `json:"fullPath"`
}

// GetFullPath returns __getBranchNamesInput.FullPath, and is useful for accessing the field via an interface.
func (v *__getBranchNamesInput) GetFullPath() string { return v.FullPath }

// __getMergeRequestsInput is used internally by genqlient
type __getMergeRequestsInput struct {
	FullPath string            `json:"fullPath"`
	After    *string           `json:"after"`
	State    MergeRequestState `json:"state"`
}

// GetFullPath returns __getMergeRequestsInput.FullPath, and is useful for accessing the field via an interface.
func (v *__getMergeRequestsInput) GetFullPath() string { return v.FullPath }

// GetAfter returns __getMergeRequestsInput.After, and is useful for accessing the field via an interface.
func (v *__getMergeRequestsInput) GetAfter() *string { return v.After }

// GetState returns __getMergeRequestsInput.State, and is useful for accessing the field via an interface.
func (v *__getMergeRequestsInput) GetState() MergeRequestState { return v.State }

// __getProjectsByTopicInput is used internally by genqlient
type __getProjectsByTopicInput struct {
	Org    string   `json:"org"`
	Topics []string `json:"topics"`
}

// GetOrg returns __getProjectsByTopicInput.Org, and is useful for accessing the field via an interface.
func (v *__getProjectsByTopicInput) GetOrg() string { return v.Org }

// GetTopics returns __getProjectsByTopicInput.Topics, and is useful for accessing the field via an interface.
func (v *__getProjectsByTopicInput) GetTopics() []string { return v.Topics }

// getAllGroupProjectsGroup includes the requested fields of the GraphQL type Group.
type getAllGroupProjectsGroup struct {
	// Projects within this namespace.
	Projects getAllGroupProjectsGroupProjectsProjectConnection `json:"projects"`
}

// GetProjects returns getAllGroupProjectsGroup.Projects, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroup) GetProjects() getAllGroupProjectsGroupProjectsProjectConnection {
	return v.Projects
}

// getAllGroupProjectsGroupProjectsProjectConnection includes the requested fields of the GraphQL type ProjectConnection.
// The GraphQL type's documentation follows.
//
// The connection type for Project.
type getAllGroupProjectsGroupProjectsProjectConnection struct {
	// Total count of collection.
	Count int `json:"count"`
	// Information to aid in pagination.
	PageInfo getAllGroupProjectsGroupProjectsProjectConnectionPageInfo `json:"pageInfo"`
	// A list of nodes.
	Nodes []getAllGroupProjectsGroupProjectsProjectConnectionNodesProject `json:"nodes"`
}

// GetCount returns getAllGroupProjectsGroupProjectsProjectConnection.Count, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnection) GetCount() int { return v.Count }

// GetPageInfo returns getAllGroupProjectsGroupProjectsProjectConnection.PageInfo, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnection) GetPageInfo() getAllGroupProjectsGroupProjectsProjectConnectionPageInfo {
	return v.PageInfo
}

// GetNodes returns getAllGroupProjectsGroupProjectsProjectConnection.Nodes, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnection) GetNodes() []getAllGroupProjectsGroupProjectsProjectConnectionNodesProject {
	return v.Nodes
}

// getAllGroupProjectsGroupProjectsProjectConnectionNodesProject includes the requested fields of the GraphQL type Project.
type getAllGroupProjectsGroupProjectsProjectConnectionNodesProject struct {
	// Name of the project (without namespace).
	Name string `json:"name"`
	// Full path of the project.
	FullPath string `json:"fullPath"`
	// Timestamp of the project creation.
	CreatedAt time.Time `json:"createdAt"`
	// Timestamp of the project last activity.
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// GetName returns getAllGroupProjectsGroupProjectsProjectConnectionNodesProject.Name, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnectionNodesProject) GetName() string {
	return v.Name
}

// GetFullPath returns getAllGroupProjectsGroupProjectsProjectConnectionNodesProject.FullPath, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnectionNodesProject) GetFullPath() string {
	return v.FullPath
}

// GetCreatedAt returns getAllGroupProjectsGroupProjectsProjectConnectionNodesProject.CreatedAt, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnectionNodesProject) GetCreatedAt() time.Time {
	return v.CreatedAt
}

// GetLastActivityAt returns getAllGroupProjectsGroupProjectsProjectConnectionNodesProject.LastActivityAt, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnectionNodesProject) GetLastActivityAt() time.Time {
	return v.LastActivityAt
}

// getAllGroupProjectsGroupProjectsProjectConnectionPageInfo includes the requested fields of the GraphQL type PageInfo.
// The GraphQL type's documentation follows.
//
// Information about pagination in a connection.
type getAllGroupProjectsGroupProjectsProjectConnectionPageInfo struct {
	// When paginating forwards, are there more items?
	HasNextPage bool `json:"hasNextPage"`
	// When paginating forwards, the cursor to continue.
	EndCursor string `json:"endCursor"`
}

// GetHasNextPage returns getAllGroupProjectsGroupProjectsProjectConnectionPageInfo.HasNextPage, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnectionPageInfo) GetHasNextPage() bool {
	return v.HasNextPage
}

// GetEndCursor returns getAllGroupProjectsGroupProjectsProjectConnectionPageInfo.EndCursor, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsGroupProjectsProjectConnectionPageInfo) GetEndCursor() string {
	return v.EndCursor
}

// getAllGroupProjectsResponse is returned by getAllGroupProjects on success.
type getAllGroupProjectsResponse struct {
	// Find a group.
	Group getAllGroupProjectsGroup `json:"group"`
}

// GetGroup returns getAllGroupProjectsResponse.Group, and is useful for accessing the field via an interface.
func (v *getAllGroupProjectsResponse) GetGroup() getAllGroupProjectsGroup { return v.Group }

// getBranchNamesProject includes the requested fields of the GraphQL type Project.
type getBranchNamesProject struct {
	// Git repository of the project.
	Repository getBranchNamesProjectRepository `json:"repository"`
}

// GetRepository returns getBranchNamesProject.Repository, and is useful for accessing the field via an interface.
func (v *getBranchNamesProject) GetRepository() getBranchNamesProjectRepository { return v.Repository }

// getBranchNamesProjectRepository includes the requested fields of the GraphQL type Repository.
type getBranchNamesProjectRepository struct {
	// Names of branches available in this repository that match the search pattern.
	BranchNames []string `json:"branchNames"`
}

// GetBranchNames returns getBranchNamesProjectRepository.BranchNames, and is useful for accessing the field via an interface.
func (v *getBranchNamesProjectRepository) GetBranchNames() []string { return v.BranchNames }

// getBranchNamesResponse is returned by getBranchNames on success.
type getBranchNamesResponse struct {
	// Find a project.
	Project getBranchNamesProject `json:"project"`
}

// GetProject returns getBranchNamesResponse.Project, and is useful for accessing the field via an interface.
func (v *getBranchNamesResponse) GetProject() getBranchNamesProject { return v.Project }

// getMergeRequestsProject includes the requested fields of the GraphQL type Project.
type getMergeRequestsProject struct {
	// Merge requests of the project.
	MergeRequests getMergeRequestsProjectMergeRequestsMergeRequestConnection `json:"mergeRequests"`
}

// GetMergeRequests returns getMergeRequestsProject.MergeRequests, and is useful for accessing the field via an interface.
func (v *getMergeRequestsProject) GetMergeRequests() getMergeRequestsProjectMergeRequestsMergeRequestConnection {
	return v.MergeRequests
}

// getMergeRequestsProjectMergeRequestsMergeRequestConnection includes the requested fields of the GraphQL type MergeRequestConnection.
// The GraphQL type's documentation follows.
//
// The connection type for MergeRequest.
type getMergeRequestsProjectMergeRequestsMergeRequestConnection struct {
	// Information to aid in pagination.
	PageInfo getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo `json:"pageInfo"`
	// A list of nodes.
	Nodes []MergeRequestNode `json:"nodes"`
}

// GetPageInfo returns getMergeRequestsProjectMergeRequestsMergeRequestConnection.PageInfo, and is useful for accessing the field via an interface.
func (v *getMergeRequestsProjectMergeRequestsMergeRequestConnection) GetPageInfo() getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo {
	return v.PageInfo
}

// GetNodes returns getMergeRequestsProjectMergeRequestsMergeRequestConnection.Nodes, and is useful for accessing the field via an interface.
func (v *getMergeRequestsProjectMergeRequestsMergeRequestConnection) GetNodes() []MergeRequestNode {
	return v.Nodes
}

// getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo includes the requested fields of the GraphQL type PageInfo.
// The GraphQL type's documentation follows.
//
// Information about pagination in a connection.
type getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo struct {
	// When paginating forwards, are there more items?
	HasNextPage bool `json:"hasNextPage"`
	// When paginating forwards, the cursor to continue.
	EndCursor string `json:"endCursor"`
}

// GetHasNextPage returns getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo.HasNextPage, and is useful for accessing the field via an interface.
func (v *getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo) GetHasNextPage() bool {
	return v.HasNextPage
}

// GetEndCursor returns getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo.EndCursor, and is useful for accessing the field via an interface.
func (v *getMergeRequestsProjectMergeRequestsMergeRequestConnectionPageInfo) GetEndCursor() string {
	return v.EndCursor
}

// getMergeRequestsResponse is returned by getMergeRequests on success.
type getMergeRequestsResponse struct {
	// Find a project.
	Project getMergeRequestsProject `json:"project"`
}

// GetProject returns getMergeRequestsResponse.Project, and is useful for accessing the field via an interface.
func (v *getMergeRequestsResponse) GetProject() getMergeRequestsProject { return v.Project }

// getProjectsByTopicProjectsProjectConnection includes the requested fields of the GraphQL type ProjectConnection.
// The GraphQL type's documentation follows.
//
// The connection type for Project.
type getProjectsByTopicProjectsProjectConnection struct {
	// A list of nodes.
	Nodes []getProjectsByTopicProjectsProjectConnectionNodesProject `json:"nodes"`
}

// GetNodes returns getProjectsByTopicProjectsProjectConnection.Nodes, and is useful for accessing the field via an interface.
func (v *getProjectsByTopicProjectsProjectConnection) GetNodes() []getProjectsByTopicProjectsProjectConnectionNodesProject {
	return v.Nodes
}

// getProjectsByTopicProjectsProjectConnectionNodesProject includes the requested fields of the GraphQL type Project.
type getProjectsByTopicProjectsProjectConnectionNodesProject struct {
	// Name of the project (without namespace).
	Name string `json:"name"`
	// Full path of the project.
	FullPath string `json:"fullPath"`
	// Timestamp of the project creation.
	CreatedAt time.Time `json:"createdAt"`
	// Timestamp of the project last activity.
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// GetName returns getProjectsByTopicProjectsProjectConnectionNodesProject.Name, and is useful for accessing the field via an interface.
func (v *getProjectsByTopicProjectsProjectConnectionNodesProject) GetName() string { return v.Name }

// GetFullPath returns getProjectsByTopicProjectsProjectConnectionNodesProject.FullPath, and is useful for accessing the field via an interface.
func (v *getProjectsByTopicProjectsProjectConnectionNodesProject) GetFullPath() string {
	return v.FullPath
}

// GetCreatedAt returns getProjectsByTopicProjectsProjectConnectionNodesProject.CreatedAt, and is useful for accessing the field via an interface.
func (v *getProjectsByTopicProjectsProjectConnectionNodesProject) GetCreatedAt() time.Time {
	return v.CreatedAt
}

// GetLastActivityAt returns getProjectsByTopicProjectsProjectConnectionNodesProject.LastActivityAt, and is useful for accessing the field via an interface.
func (v *getProjectsByTopicProjectsProjectConnectionNodesProject) GetLastActivityAt() time.Time {
	return v.LastActivityAt
}

// getProjectsByTopicResponse is returned by getProjectsByTopic on success.
type getProjectsByTopicResponse struct {
	// Find projects visible to the current user.
	Projects getProjectsByTopicProjectsProjectConnection `json:"projects"`
}

// GetProjects returns getProjectsByTopicResponse.Projects, and is useful for accessing the field via an interface.
func (v *getProjectsByTopicResponse) GetProjects() getProjectsByTopicProjectsProjectConnection {
	return v.Projects
}

// The query or mutation executed by getAllGroupProjects.
const getAllGroupProjects_Operation = `
query getAllGroupProjects ($fullPath: ID!, $after: String) {
	group(fullPath: $fullPath) {
		projects(includeSubgroups: true, after: $after) {
			count
			pageInfo {
				hasNextPage
				endCursor
			}
			nodes {
				name
				fullPath
				createdAt
				lastActivityAt
			}
		}
	}
}
`

func getAllGroupProjects(
	ctx context.Context,
	client graphql.Client,
	fullPath string,
	after *string,
) (*getAllGroupProjectsResponse, error) {
	req := &graphql.Request{
		OpName: "getAllGroupProjects",
		Query:  getAllGroupProjects_Operation,
		Variables: &__getAllGroupProjectsInput{
			FullPath: fullPath,
			After:    after,
		},
	}
	var err error

	var data getAllGroupProjectsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

// The query or mutation executed by getBranchNames.
const getBranchNames_Operation = `
query getBranchNames ($fullPath: ID!) {
	project(fullPath: $fullPath) {
		repository {
			branchNames(searchPattern: "*", offset: 0, limit: 100000)
		}
	}
}
`

func getBranchNames(
	ctx context.Context,
	client graphql.Client,
	fullPath string,
) (*getBranchNamesResponse, error) {
	req := &graphql.Request{
		OpName: "getBranchNames",
		Query:  getBranchNames_Operation,
		Variables: &__getBranchNamesInput{
			FullPath: fullPath,
		},
	}
	var err error

	var data getBranchNamesResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

// The query or mutation executed by getMergeRequests.
const getMergeRequests_Operation = `
query getMergeRequests ($fullPath: ID!, $after: String, $state: MergeRequestState) {
	project(fullPath: $fullPath) {
		mergeRequests(state: $state, after: $after) {
			pageInfo {
				hasNextPage
				endCursor
			}
			nodes {
				iid
				title
				sourceBranch
				targetBranch
				createdAt
				mergedAt
				project {
					fullPath
				}
			}
		}
	}
}
`

func getMergeRequests(
	ctx context.Context,
	client graphql.Client,
	fullPath string,
	after *string,
	state MergeRequestState,
) (*getMergeRequestsResponse, error) {
	req := &graphql.Request{
		OpName: "getMergeRequests",
		Query:  getMergeRequests_Operation,
		Variables: &__getMergeRequestsInput{
			FullPath: fullPath,
			After:    after,
			State:    state,
		},
	}
	var err error

	var data getMergeRequestsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

// The query or mutation executed by getProjectsByTopic.
const getProjectsByTopic_Operation = `
query getProjectsByTopic ($org: String!, $topics: [String!]) {
	projects(searchNamespaces: true, search: $org, topics: $topics) {
		nodes {
			name
			fullPath
			createdAt
			lastActivityAt
		}
	}
}
`

func getProjectsByTopic(
	ctx context.Context,
	client graphql.Client,
	org string,
	topics []string,
) (*getProjectsByTopicResponse, error) {
	req := &graphql.Request{
		OpName: "getProjectsByTopic",
		Query:  getProjectsByTopic_Operation,
		Variables: &__getProjectsByTopicInput{
			Org:    org,
			Topics: topics,
		},
	}
	var err error

	var data getProjectsByTopicResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
