package gitlabcatalogscraper

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/cenkalti/backoff/v5"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// componentIncludeRegex matches component include lines in .gitlab-ci.yml
// e.g., "- component: gitlab.com/components/opentofu/fmt@4.5.0"
// Captures everything after the FQDN: "components/opentofu/fmt@4.5.0"
// Only matches lines that are not commented out (no leading #)
var componentIncludeRegex = regexp.MustCompile(`^\s*-\s*component:\s*[^/]+/(.+)`)

type gitlabProject struct {
	Name string
	ID   string
	Path string
	URL  string
}

func (gcs *gitlabCatalogScraper) getProjects(ctx context.Context, restClient *gitlab.Client) ([]gitlabProject, error) {
	var projectList []gitlabProject

	operation := func() (string, error) {
		var localProjects []gitlabProject

		for nextPage := 1; nextPage > 0; {
			projects, res, err := restClient.Groups.ListGroupProjects(gcs.cfg.GitLabOrg, &gitlab.ListGroupProjectsOptions{
				IncludeSubGroups: gitlab.Ptr(false),
				Archived:         gitlab.Ptr(false),
				ListOptions: gitlab.ListOptions{
					Page:    nextPage,
					PerPage: 100,
				},
			})
			if err != nil {
				if apiErr, ok := err.(*gitlab.ErrorResponse); ok && apiErr.Response.StatusCode == 429 {
					return "", backoff.RetryAfter(60)
				}
				return "", backoff.Permanent(err)
			}

			if len(projects) == 0 && nextPage == 1 {
				errMsg := fmt.Sprintf("no GitLab projects found for the given group/org: %s", gcs.cfg.GitLabOrg)
				gcs.logger.Sugar().Warn(errMsg)
				return "success", nil
			}

			for _, p := range projects {
				localProjects = append(localProjects, gitlabProject{
					Name: p.Name,
					ID:   strconv.Itoa(p.ID),
					Path: p.PathWithNamespace,
					URL:  p.WebURL,
				})
			}

			nextPageHeader := res.Header.Get("x-next-page")
			if len(nextPageHeader) > 0 {
				nextPage, err = strconv.Atoi(nextPageHeader)
				if err != nil {
					return "", backoff.Permanent(err)
				}
			} else {
				nextPage = 0
			}
		}

		projectList = localProjects
		return "success", nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return nil, err
	}

	gcs.logger.Sugar().Infof("Found %d projects for GitLab org %s", len(projectList), gcs.cfg.GitLabOrg)
	return projectList, nil
}

type catalogResourceInfo struct {
	Name                string
	FullPath            string
	StarCount           int
	Last30DayUsageCount int
}

// getComponentResourcePaths fetches a project's .gitlab-ci.yml and extracts
// a map of full component path → catalog resource path from the component: include lines.
// e.g., "components/opentofu/fmt" → "components/opentofu"
func (gcs *gitlabCatalogScraper) getComponentResourcePaths(restClient *gitlab.Client, projectPath string) (map[string]string, error) {
	result := make(map[string]string)

	fileContent, _, err := restClient.RepositoryFiles.GetRawFile(projectPath, ".gitlab-ci.yml", &gitlab.GetRawFileOptions{
		Ref: gitlab.Ptr("HEAD"),
	})
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(fileContent), "\n")
	for _, line := range lines {
		matches := componentIncludeRegex.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}
		// matches[1] is everything after the FQDN, e.g., "components/opentofu/fmt@4.5.0"
		pathWithVersion := matches[1]

		// Split off the @version from the last segment
		// "components/opentofu/fmt@4.5.0" → "components/opentofu/fmt"
		atIdx := strings.LastIndex(pathWithVersion, "@")
		if atIdx < 0 {
			continue
		}
		fullComponentPath := pathWithVersion[:atIdx]

		// Split into segments: ["components", "opentofu", "fmt"]
		// Resource path is everything except the last segment
		// Component name is the last segment
		lastSlash := strings.LastIndex(fullComponentPath, "/")
		if lastSlash < 0 {
			continue
		}
		resourcePath := fullComponentPath[:lastSlash]

		result[fullComponentPath] = resourcePath
	}

	return result, nil
}

// getCatalogResourceByPath fetches a catalog resource's details by its full path.
func (gcs *gitlabCatalogScraper) getCatalogResourceByPath(ctx context.Context, client graphql.Client, fullPath string) (*catalogResourceInfo, error) {
	var result *catalogResourceInfo

	operation := func() (string, error) {
		resp, err := getCatalogResource(ctx, client, fullPath)
		if err != nil {
			return "", backoff.Permanent(err)
		}

		result = &catalogResourceInfo{
			Name:                resp.CiCatalogResource.Name,
			FullPath:            resp.CiCatalogResource.FullPath,
			StarCount:           resp.CiCatalogResource.StarCount,
			Last30DayUsageCount: resp.CiCatalogResource.Last30DayUsageCount,
		}
		return "success", nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (gcs *gitlabCatalogScraper) getProjectComponentUsages(ctx context.Context, client graphql.Client, projectPath string) ([]ComponentUsageNode, error) {
	var usages []ComponentUsageNode
	var cursor *string

	for hasNextPage := true; hasNextPage; {
		operation := func() (string, error) {
			resp, err := getProjectComponentUsages(ctx, client, projectPath, cursor)
			if err != nil {
				return "", backoff.Permanent(err)
			}

			if len(resp.Project.ComponentUsages.Nodes) == 0 && !resp.Project.ComponentUsages.PageInfo.HasNextPage {
				hasNextPage = false
				return "success", nil
			}

			usages = append(usages, resp.Project.ComponentUsages.Nodes...)
			cursor = &resp.Project.ComponentUsages.PageInfo.EndCursor
			hasNextPage = resp.Project.ComponentUsages.PageInfo.HasNextPage

			return "success", nil
		}

		_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
		if err != nil {
			return nil, err
		}
	}

	return usages, nil
}
