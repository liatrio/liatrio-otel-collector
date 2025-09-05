// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"time"
)

// AzureDevOpsRepository represents a repository in Azure DevOps webhook events
type AzureDevOpsRepository struct {
	Alias  string `json:"alias"`
	ID     string `json:"id"`
	Type   string `json:"type"`
	Change struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
		Version string `json:"version"`
	} `json:"change"`
	URL string `json:"url"`
}

// Azure DevOps webhook event types and structures
// Based on Azure DevOps Pipeline Service Hook Events documentation

// PipelineRunStateChangedEvent represents a ms.vss-pipelines.run-state-changed-event from Azure DevOps
type PipelineRunStateChangedEvent struct {
	SubscriptionID string `json:"subscriptionId"`
	NotificationID int64  `json:"notificationId"`
	ID             string `json:"id"`
	EventType      string `json:"eventType"`
	PublisherID    string `json:"publisherId"`
	Message        struct {
		Text     string `json:"text"`
		HTML     string `json:"html"`
		Markdown string `json:"markdown"`
	} `json:"message"`
	DetailedMessage struct {
		Text     string `json:"text"`
		HTML     string `json:"html"`
		Markdown string `json:"markdown"`
	} `json:"detailedMessage"`
	Resource struct {
		ProjectID string `json:"projectId"`
		Run       struct {
			Links struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
				Pipeline struct {
					Href string `json:"href"`
				} `json:"pipeline"`
			} `json:"_links"`
			TemplateParameters map[string]interface{} `json:"templateParameters"`
			Pipeline           struct {
				URL      string `json:"url"`
				ID       int64  `json:"id"`
				Revision int64  `json:"revision"`
				Name     string `json:"name"`
				Folder   string `json:"folder"`
			} `json:"pipeline"`
			State        string     `json:"state"`
			Result       string     `json:"result"`
			CreatedDate  time.Time  `json:"createdDate"`
			FinishedDate *time.Time `json:"finishedDate"`
			URL          string     `json:"url"`
			Resources    struct {
				Repositories map[string]struct {
					Repository struct {
						ID         string `json:"id,omitempty"`
						FullName   string `json:"fullName,omitempty"`
						Connection struct {
							ID string `json:"id"`
						} `json:"connection,omitempty"`
						Type string `json:"type"`
					} `json:"repository"`
					RefName string `json:"refName"`
					Version string `json:"version"`
				} `json:"repositories"`
			} `json:"resources"`
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"run"`
		Pipeline struct {
			URL      string `json:"url"`
			ID       int64  `json:"id"`
			Revision int64  `json:"revision"`
			Name     string `json:"name"`
			Folder   string `json:"folder"`
		} `json:"pipeline"`
		Stages []struct {
			Links struct {
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
			} `json:"_links"`
			ID          string     `json:"id"`
			Name        string     `json:"name"`
			DisplayName string     `json:"displayName"`
			Attempt     int64      `json:"attempt"`
			State       string     `json:"state"`
			Result      string     `json:"result,omitempty"`
			StartTime   *time.Time `json:"startTime"`
			FinishTime  *time.Time `json:"finishTime,omitempty"`
		} `json:"stages"`
		RequestedBy struct {
			DisplayName string `json:"displayName"`
			URL         string `json:"url"`
			Links       struct {
				Avatar struct {
					Href string `json:"href"`
				} `json:"avatar"`
			} `json:"_links"`
			ID         string `json:"id"`
			UniqueName string `json:"uniqueName"`
			ImageURL   string `json:"imageUrl"`
			Descriptor string `json:"descriptor"`
		} `json:"requestedBy"`
		RequestedFor struct {
			DisplayName string `json:"displayName"`
			URL         string `json:"url"`
			Links       struct {
				Avatar struct {
					Href string `json:"href"`
				} `json:"avatar"`
			} `json:"_links"`
			ID         string `json:"id"`
			UniqueName string `json:"uniqueName"`
			ImageURL   string `json:"imageUrl"`
			Descriptor string `json:"descriptor"`
		} `json:"requestedFor"`
		Queue struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Pool struct {
				ID       int64  `json:"id"`
				Name     string `json:"name"`
				IsHosted bool   `json:"isHosted"`
			} `json:"pool"`
		} `json:"queue"`
		RunID        int64                   `json:"runId"`
		RunURL       string                  `json:"runUrl"`
		Repositories []AzureDevOpsRepository `json:"repositories"`
	} `json:"resource"`
	ResourceVersion    string `json:"resourceVersion"`
	ResourceContainers struct {
		Collection struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"collection"`
		Account struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"account"`
		Project struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"project"`
	} `json:"resourceContainers"`
	CreatedDate time.Time `json:"createdDate"`
}

// PipelineStageStateChangedEvent represents a ms.vss-pipelines.stage-state-changed-event from Azure DevOps
type PipelineStageStateChangedEvent struct {
	SubscriptionID string `json:"subscriptionId"`
	NotificationID int64  `json:"notificationId"`
	ID             string `json:"id"`
	EventType      string `json:"eventType"`
	PublisherID    string `json:"publisherId"`
	Message        struct {
		Text     string `json:"text"`
		HTML     string `json:"html"`
		Markdown string `json:"markdown"`
	} `json:"message"`
	DetailedMessage struct {
		Text     string `json:"text"`
		HTML     string `json:"html"`
		Markdown string `json:"markdown"`
	} `json:"detailedMessage"`
	Resource struct {
		Stage struct {
			Links struct {
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
			} `json:"_links"`
			ID          string     `json:"id"`
			Name        string     `json:"name"`
			DisplayName string     `json:"displayName"`
			Attempt     int64      `json:"attempt"`
			State       string     `json:"state"`
			Result      string     `json:"result"`
			StartTime   *time.Time `json:"startTime"`
			FinishTime  *time.Time `json:"finishTime"`
		} `json:"stage"`
		Run struct {
			Links struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
				Pipeline struct {
					Href string `json:"href"`
				} `json:"pipeline"`
			} `json:"_links"`
			TemplateParameters map[string]interface{} `json:"templateParameters"`
			Pipeline           struct {
				URL      string `json:"url"`
				ID       int64  `json:"id"`
				Revision int64  `json:"revision"`
				Name     string `json:"name"`
				Folder   string `json:"folder"`
			} `json:"pipeline"`
			State        string     `json:"state"`
			Result       string     `json:"result,omitempty"`
			CreatedDate  time.Time  `json:"createdDate"`
			FinishedDate *time.Time `json:"finishedDate,omitempty"`
			URL          string     `json:"url"`
			Resources    struct {
				Repositories map[string]struct {
					Repository struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"repository"`
					RefName string `json:"refName"`
					Version string `json:"version"`
				} `json:"repositories"`
			} `json:"resources"`
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"run"`
		Pipeline struct {
			URL      string `json:"url"`
			ID       int64  `json:"id"`
			Revision int64  `json:"revision"`
			Name     string `json:"name"`
			Folder   string `json:"folder"`
		} `json:"pipeline"`
		RunID        int64                   `json:"runId"`
		StageName    string                  `json:"stageName"`
		RunURL       string                  `json:"runUrl"`
		ProjectID    string                  `json:"projectId"`
		Repositories []AzureDevOpsRepository `json:"repositories"`
	} `json:"resource"`
	ResourceVersion    string `json:"resourceVersion"`
	ResourceContainers struct {
		Collection struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"collection"`
		Account struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"account"`
		Project struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"project"`
	} `json:"resourceContainers"`
	CreatedDate time.Time `json:"createdDate"`
}

// PipelineJobStateChangedEvent represents a ms.vss-pipelines.job-state-changed-event from Azure DevOps
type PipelineJobStateChangedEvent struct {
	SubscriptionID string `json:"subscriptionId"`
	NotificationID int64  `json:"notificationId"`
	ID             string `json:"id"`
	EventType      string `json:"eventType"`
	PublisherID    string `json:"publisherId"`
	Message        struct {
		Text     string `json:"text"`
		HTML     string `json:"html"`
		Markdown string `json:"markdown"`
	} `json:"message"`
	DetailedMessage struct {
		Text     string `json:"text"`
		HTML     string `json:"html"`
		Markdown string `json:"markdown"`
	} `json:"detailedMessage"`
	Resource struct {
		ProjectID string `json:"projectId"`
		Job       struct {
			Links struct {
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
			} `json:"_links"`
			ID         string     `json:"id"`
			Name       string     `json:"name"`
			Attempt    int64      `json:"attempt"`
			State      string     `json:"state"`
			Result     string     `json:"result"`
			StartTime  *time.Time `json:"startTime"`
			FinishTime *time.Time `json:"finishTime"`
		} `json:"job"`
		Stage struct {
			Links struct {
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
			} `json:"_links"`
			ID          string     `json:"id"`
			Name        string     `json:"name"`
			DisplayName string     `json:"displayName"`
			Attempt     int64      `json:"attempt"`
			State       string     `json:"state"`
			Result      string     `json:"result,omitempty"`
			StartTime   *time.Time `json:"startTime"`
			FinishTime  *time.Time `json:"finishTime,omitempty"`
		} `json:"stage"`
		Run struct {
			Links struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
				Pipeline struct {
					Href string `json:"href"`
				} `json:"pipeline"`
			} `json:"_links"`
			TemplateParameters map[string]interface{} `json:"templateParameters"`
			Pipeline           struct {
				URL      string `json:"url"`
				ID       int64  `json:"id"`
				Revision int64  `json:"revision"`
				Name     string `json:"name"`
				Folder   string `json:"folder"`
			} `json:"pipeline"`
			State        string     `json:"state"`
			Result       string     `json:"result,omitempty"`
			CreatedDate  time.Time  `json:"createdDate"`
			FinishedDate *time.Time `json:"finishedDate,omitempty"`
			URL          string     `json:"url"`
			Resources    struct {
				Repositories map[string]struct {
					Repository struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"repository"`
					RefName string `json:"refName"`
					Version string `json:"version"`
				} `json:"repositories"`
			} `json:"resources"`
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"run"`
		Pipeline struct {
			URL      string `json:"url"`
			ID       int64  `json:"id"`
			Revision int64  `json:"revision"`
			Name     string `json:"name"`
			Folder   string `json:"folder"`
		} `json:"pipeline"`
		Repositories []AzureDevOpsRepository `json:"repositories"`
	} `json:"resource"`
	ResourceVersion    string `json:"resourceVersion"`
	ResourceContainers struct {
		Collection struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"collection"`
		Account struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"account"`
		Project struct {
			ID      string `json:"id"`
			BaseURL string `json:"baseUrl"`
		} `json:"project"`
	} `json:"resourceContainers"`
	CreatedDate time.Time `json:"createdDate"`
}
