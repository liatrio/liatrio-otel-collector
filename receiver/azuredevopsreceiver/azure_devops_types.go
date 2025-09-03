// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsreceiver

import (
	"time"
)

// Azure DevOps webhook event types and structures
// Based on Azure DevOps Pipeline Service Hook Events documentation

// PipelineRunStateChangedEvent represents a ms.vss-pipelines.run-state-changed-event from Azure DevOps
type PipelineRunStateChangedEvent struct {
	ID          string `json:"id"`
	EventType   string `json:"eventType"`
	PublisherID string `json:"publisherId"`
	Message     struct {
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
		Run struct {
			Links struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
			} `json:"_links"`
			Pipeline struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"pipeline"`
			State        string    `json:"state"`
			Result       string    `json:"result"`
			CreatedDate  time.Time `json:"createdDate"`
			FinishedDate time.Time `json:"finishedDate"`
			URL          string    `json:"url"`
		} `json:"run"`
	} `json:"resource"`
	CreatedDate time.Time `json:"createdDate"`
}

// PipelineStageStateChangedEvent represents a ms.vss-pipelines.stage-state-changed-event from Azure DevOps
type PipelineStageStateChangedEvent struct {
	ID          string `json:"id"`
	EventType   string `json:"eventType"`
	PublisherID string `json:"publisherId"`
	Message     struct {
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
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			State       string `json:"state"`
			Result      string `json:"result"`
		} `json:"stage"`
		Run struct {
			Pipeline struct {
				URL      string `json:"url"`
				ID       int    `json:"id"`
				Revision int    `json:"revision"`
				Name     string `json:"name"`
				Folder   string `json:"folder"`
			} `json:"pipeline"`
			State        string    `json:"state"`
			Result       string    `json:"result"`
			CreatedDate  time.Time `json:"createdDate"`
			FinishedDate time.Time `json:"finishedDate"`
			ID           int       `json:"id"`
			Name         string    `json:"name"`
		} `json:"run"`
		Pipeline struct {
			URL      string `json:"url"`
			ID       int    `json:"id"`
			Revision int    `json:"revision"`
			Name     string `json:"name"`
			Folder   string `json:"folder"`
		} `json:"pipeline"`
		Repositories []struct {
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
			} `json:"change"`
			URL string `json:"url"`
		} `json:"repositories"`
	} `json:"resource"`
	ResourceVersion    string `json:"resourceVersion"`
	ResourceContainers struct {
		Collection struct {
			ID string `json:"id"`
		} `json:"collection"`
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	} `json:"resourceContainers"`
	CreatedDate time.Time `json:"createdDate"`
}

// PipelineJobStateChangedEvent represents a ms.vss-pipelines.job-state-changed-event from Azure DevOps
type PipelineJobStateChangedEvent struct {
	SubscriptionID string `json:"subscriptionId"`
	NotificationID int    `json:"notificationId"`
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
		Job struct {
			Links struct {
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
				PipelineWeb struct {
					Href string `json:"href"`
				} `json:"pipeline.web"`
			} `json:"_links"`
			ID         string    `json:"id"`
			Name       string    `json:"name"`
			State      string    `json:"state"`
			Result     string    `json:"result"`
			StartTime  time.Time `json:"startTime"`
			FinishTime time.Time `json:"finishTime"`
			Attempt    int       `json:"attempt"`
		} `json:"job"`
		Stage struct {
			ID          string    `json:"id"`
			Name        string    `json:"name"`
			DisplayName string    `json:"displayName"`
			Attempt     int       `json:"attempt"`
			State       string    `json:"state"`
			Result      string    `json:"result"`
			StartTime   time.Time `json:"startTime"`
			FinishTime  time.Time `json:"finishTime"`
		} `json:"stage"`
		Run struct {
			Pipeline struct {
				URL      string `json:"url"`
				ID       int    `json:"id"`
				Revision int    `json:"revision"`
				Name     string `json:"name"`
				Folder   string `json:"folder"`
			} `json:"pipeline"`
			State        string    `json:"state"`
			Result       string    `json:"result"`
			CreatedDate  time.Time `json:"createdDate"`
			FinishedDate time.Time `json:"finishedDate"`
			ID           int       `json:"id"`
			Name         string    `json:"name"`
		} `json:"run"`
		Pipeline struct {
			URL      string `json:"url"`
			ID       int    `json:"id"`
			Revision int    `json:"revision"`
			Name     string `json:"name"`
			Folder   string `json:"folder"`
		} `json:"pipeline"`
		Repositories []struct {
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
		} `json:"repositories"`
	} `json:"resource"`
	ResourceVersion    string `json:"resourceVersion"`
	ResourceContainers struct {
		Collection struct {
			ID string `json:"id"`
		} `json:"collection"`
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	} `json:"resourceContainers"`
	CreatedDate time.Time `json:"createdDate"`
}
