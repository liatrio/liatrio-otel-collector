package gitlabterraformscraper

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff/v5"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var sourceLineRegex = regexp.MustCompile(`(?m)^\s*source\s*=\s*"([^"]*)"`)

type terraformModule struct {
	Name            string
	System          string
	SourceProjectID int
}

func (gts *gitlabTerraformScraper) getModules(ctx context.Context, restClient *gitlab.Client) ([]terraformModule, error) {
	var modules []terraformModule

	operation := func() (string, error) {
		// Accumulate into a fresh slice on each attempt; if the operation
		// fails partway through and is retried, we don't want stale entries
		// from the previous attempt to leak into the final result.
		var attemptModules []terraformModule
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
				attemptModules = append(attemptModules, terraformModule{
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
		modules = attemptModules
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

// searchModuleConsumers finds projects in the configured group whose .tf
// files reference the given module via a `source = "..."` line. When
// resolveProjectInfo is true, each returned consumer also has its display
// name and URL populated via a per-project API lookup; pass false to skip
// that work when only the consumer count is needed.
func (gts *gitlabTerraformScraper) searchModuleConsumers(ctx context.Context, restClient *gitlab.Client, module terraformModule, resolveProjectInfo bool) ([]moduleConsumer, error) {
	// Build the search query with server-side filters:
	//   - Quote the module name to force exact-token match (avoids the
	//     Elasticsearch tokenizer splitting on hyphens, e.g. "my-vpc" → "my" "vpc").
	//   - Require the term `source` to be present, since legitimate consumers
	//     declare modules via `source = "..."`.
	//   - Restrict to .tf files via the extension filter, eliminating hits in
	//     READMEs, docs, and unrelated file types before they cross the wire.
	// We match on module name only (not name/system) because consumers may
	// reference modules via git URLs (e.g., git::https://.../module-name.git)
	// rather than the registry format (name/system).
	query := fmt.Sprintf(`"%s" source extension:tf`, module.Name)

	var allBlobs []*gitlab.Blob

	operation := func() (string, error) {
		// Accumulate into a fresh slice on each attempt; on retry we don't want
		// blobs from a partial previous attempt leaking through.
		var attemptBlobs []*gitlab.Blob
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

			attemptBlobs = append(attemptBlobs, blobs...)

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
		allBlobs = attemptBlobs
		return "success", nil
	}

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return nil, err
	}

	// Filter results: exclude the module's own source project, verify the match
	// is on a `source = "..."` line referencing this module as a path segment
	// (eliminates comments, descriptions, variable names, and substring matches
	// against longer module names), then deduplicate by project ID. The .tf
	// extension filter is applied server-side via the search query.
	seen := make(map[int]bool)
	var consumers []moduleConsumer
	for _, blob := range allBlobs {
		if blob.ProjectID == module.SourceProjectID {
			continue
		}
		if !matchesModuleSource(blob.Data, module.Name) {
			continue
		}
		if !seen[blob.ProjectID] {
			seen[blob.ProjectID] = true
			consumers = append(consumers, moduleConsumer{
				ProjectID: blob.ProjectID,
			})
		}
	}

	// Resolve project names and URLs only when the caller needs them; otherwise
	// skip a `GET /projects/:id` call per consumer.
	if resolveProjectInfo {
		for i, consumer := range consumers {
			name, url, err := gts.getProjectInfo(ctx, restClient, consumer.ProjectID)
			if err != nil {
				gts.logger.Sugar().Warnf("could not resolve project info for ID %d: %v", consumer.ProjectID, err)
				consumers[i].ProjectName = strconv.Itoa(consumer.ProjectID)
				consumers[i].ProjectURL = ""
				continue
			}
			consumers[i].ProjectName = name
			consumers[i].ProjectURL = url
		}
	}

	return consumers, nil
}

func (gts *gitlabTerraformScraper) getProjectInfo(ctx context.Context, restClient *gitlab.Client, projectID int) (string, string, error) {
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

	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
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

// matchesModuleSource reports whether data contains a `source = "..."` line
// whose path references moduleName as a distinct segment. The module name must
// be preceded by a path separator (`/` or `:`) or the start of the source
// string, and followed by `/`, the end of the source string, or `.git`.
func matchesModuleSource(data, moduleName string) bool {
	if moduleName == "" {
		return false
	}
	nameSegment := regexp.MustCompile(`(?:^|[/:])` + regexp.QuoteMeta(moduleName) + `(?:/|$|\.git)`)
	for _, m := range sourceLineRegex.FindAllStringSubmatch(data, -1) {
		if nameSegment.MatchString(m[1]) {
			return true
		}
	}
	return false
}
