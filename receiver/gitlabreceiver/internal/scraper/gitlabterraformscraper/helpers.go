package gitlabterraformscraper

import (
	"context"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff/v5"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type terraformModule struct {
	Name            string
	System          string
	SourceProjectID int
}

func (gts *gitlabTerraformScraper) getModules(ctx context.Context, restClient *gitlab.Client) ([]terraformModule, error) {
	var modules []terraformModule

	operation := func() (string, error) {
		for nextPage := 1; nextPage > 0; {
			packages, res, err := restClient.Packages.ListGroupPackages(gts.cfg.GitLabOrg, &gitlab.ListGroupPackagesOptions{
				PackageType: gitlab.Ptr("terraform_module"),
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

			for _, p := range packages {
				name, system := parseModuleName(p.Name)
				modules = append(modules, terraformModule{
					Name:            name,
					System:          system,
					SourceProjectID: p.ProjectID,
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
		return "success", nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return nil, err
	}

	// Deduplicate modules by name+system (packages API may return one entry per version)
	seen := make(map[string]bool)
	var unique []terraformModule
	for _, m := range modules {
		key := m.Name + "/" + m.System
		if !seen[key] {
			seen[key] = true
			unique = append(unique, m)
		}
	}

	gts.logger.Sugar().Infof("Found %d total package entries, %d unique Terraform modules for GitLab group %s", len(modules), len(unique), gts.cfg.GitLabOrg)
	return unique, nil
}

type moduleConsumer struct {
	ProjectID   int
	ProjectName string
	ProjectURL  string
}

func (gts *gitlabTerraformScraper) searchModuleConsumers(ctx context.Context, restClient *gitlab.Client, module terraformModule) ([]moduleConsumer, error) {
	// Search for the module name in .tf files across the group.
	// We search by module name only (not name/system) because consumers may
	// reference modules via git URLs (e.g., git::https://.../module-name.git)
	// rather than the registry format (name/system).
	query := module.Name

	var allBlobs []*gitlab.Blob

	operation := func() (string, error) {
		for nextPage := 1; nextPage > 0; {
			blobs, res, err := restClient.Search.BlobsByGroup(gts.cfg.GitLabOrg, query, &gitlab.SearchOptions{
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

			allBlobs = append(allBlobs, blobs...)

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
		return "success", nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return nil, err
	}

	// Filter to .tf files only and deduplicate by project ID.
	// Exclude the module's own source project.
	seen := make(map[int]bool)
	var consumers []moduleConsumer
	for _, blob := range allBlobs {
		if !strings.HasSuffix(blob.Filename, ".tf") {
			continue
		}
		if blob.ProjectID == module.SourceProjectID {
			continue
		}
		if !seen[blob.ProjectID] {
			seen[blob.ProjectID] = true
			consumers = append(consumers, moduleConsumer{
				ProjectID: blob.ProjectID,
			})
		}
	}

	// Resolve project names and URLs
	for i, consumer := range consumers {
		name, url, err := gts.getProjectInfo(restClient, consumer.ProjectID)
		if err != nil {
			gts.logger.Sugar().Warnf("could not resolve project info for ID %d: %v", consumer.ProjectID, err)
			consumers[i].ProjectName = strconv.Itoa(consumer.ProjectID)
			consumers[i].ProjectURL = ""
			continue
		}
		consumers[i].ProjectName = name
		consumers[i].ProjectURL = url
	}

	return consumers, nil
}

func (gts *gitlabTerraformScraper) getProjectInfo(restClient *gitlab.Client, projectID int) (string, string, error) {
	var project *gitlab.Project

	operation := func() (string, error) {
		var err error
		project, _, err = restClient.Projects.GetProject(projectID, nil)
		if err != nil {
			if apiErr, ok := err.(*gitlab.ErrorResponse); ok && apiErr.Response.StatusCode == 429 {
				return "", backoff.RetryAfter(60)
			}
			return "", backoff.Permanent(err)
		}
		return "success", nil
	}

	_, err := backoff.Retry(context.Background(), operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return "", "", err
	}

	return project.PathWithNamespace, project.WebURL, nil
}

// parseModuleName splits a GitLab Terraform module package name into name and system.
// GitLab stores terraform module names as "name/system" (e.g., "my-vpc/aws").
// If no slash is present, system defaults to "generic".
func parseModuleName(packageName string) (string, string) {
	parts := strings.SplitN(packageName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return packageName, "generic"
}
