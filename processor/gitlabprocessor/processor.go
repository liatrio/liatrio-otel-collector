package gitlabprocessor

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Khan/genqlient/graphql"
	// "github.com/aerospike/aerospike-client-go/v7/logger"
	// "github.com/xanzy/go-gitlab"
	// "go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
)

type logProcessor struct {
	logger *zap.Logger
	cfg    *Config
	// client *http.Client
	// skipExpr expr.BoolExpr[ottllog.TransformContext]
}

// newLogAttributesProcessor returns a processor that modifies attributes of a
// log record. To construct the attributes processors, the use of the factory
// methods are required in order to validate the inputs.
func newLogProcessor(_ context.Context, logger *zap.Logger, cfg *Config) *logProcessor {
	return &logProcessor{
		logger: logger,
		cfg:    cfg,
		// client: &http.Client{},
	}
}

func (a *logProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rs := rls.At(i)
		ilss := rs.ScopeLogs()
		// resource := rs.Resource()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			logs := ils.LogRecords()
			// library := ils.Scope()
			for k := 0; k < logs.Len(); k++ {
				lr := logs.At(k)
				fullPath, exists := lr.Attributes().Get("vcs.repository.name")
				if !exists {
					continue
				}

				revision, exists := lr.Attributes().Get("vcs.ref.head.revision")
				if !exists {
					continue
				}

				comps, err := a.getPipeCompAttrs(ctx, fullPath.AsString(), revision.AsString())
				if err != nil {
					a.logger.Sugar().Errorf("error: %v", err)
					continue
				}

				// Process each component and add as attributes
				for compPath, version := range comps {
					// Split the path and get the last component
					parts := strings.Split(compPath, "/")
					if len(parts) > 0 {
						// Get the last part and transform it
						componentName := parts[len(parts)-1]
						// Convert hyphens to underscores and ensure lowercase
						componentName = strings.ToLower(strings.ReplaceAll(componentName, "-", "_"))
						// Create the attribute name
						attrName := "component." + componentName + ".version"

						// Add the attribute to the log record
						lr.Attributes().PutStr(attrName, version)
					}
				}
			}
		}

	}
	return ld, nil
}

type authedTransport struct {
	key     string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "bearer "+t.key)
	return t.wrapped.RoundTrip(req)
}

// func (a *logProcessor) getPipeCompAttrs(ctx context.Context, fullPath string, revision string) (attrs []string, err error) {
func (a *logProcessor) getPipeCompAttrs(ctx context.Context, fullPath string, revision string) (comps map[string]string, err error) {
	// a.client, err = a.cfg.ToClient(ctx, component.Host, component.TelemetrySettings)

	// Enable the ability to override the endpoint for self-hosted gitlab instances
	graphCURL := "https://gitlab.com/api/graphql"
	// restCURL := "https://gitlab.com/"

	if a.cfg.ClientConfig.Endpoint != "" {
		var err error

		graphCURL, err = url.JoinPath(a.cfg.ClientConfig.Endpoint, "api/graphql")
		if err != nil {
			a.logger.Sugar().Errorf("error: %v", err)
		}
	}

	// key := os.Getenv("GITHUB_TOKEN")
	// if key == "" {
	// 	err = fmt.Errorf("must set GITHUB_TOKEN=<github token>")
	// 	return
	// }

	// key := a.cfg.Token
	// ac := a.cfg.Auth.GetClientAuthenticator()

	httpClient := http.Client{
		Transport: &authedTransport{
			key:     a.cfg.Token,
			wrapped: http.DefaultTransport,
		},
	}

	graphClient := graphql.NewClient(graphCURL, &httpClient)
	components := make(map[string]string)

	blob, err := getBlobContent(context.Background(), graphClient, fullPath, ".gitlab-ci.yml", revision)
	if err != nil {
		a.logger.Sugar().Errorf("error getting blob content: %v", err)
		return nil, err
	}

	// Check if response was 200
	// if blob.GetProject() == "" {
	if blob.Project.Id == "" {
		return nil, fmt.Errorf("no project found")
	}

	if len(blob.Project.Repository.Blobs.GetNodes()) == 0 {
		return nil, fmt.Errorf("no blob content found")
	}

	raw := blob.Project.Repository.Blobs.GetNodes()[0].RawBlob
	lines := strings.Split(raw, "\n")

	inIncludes := false
	currentIndent := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " "))

		// Start of includes section
		if strings.HasPrefix(trimmedLine, "include:") {
			inIncludes = true
			currentIndent = indent
			continue
		}

		// Exit includes section if we're back to the original indent level or less
		if inIncludes && indent <= currentIndent && trimmedLine != "" {
			inIncludes = false
			continue
		}

		// Process component lines
		if inIncludes && strings.Contains(trimmedLine, "component:") {
			componentParts := strings.Split(trimmedLine, "component:")
			if len(componentParts) == 2 {
				componentStr := strings.TrimSpace(componentParts[1])
				parts := strings.Split(componentStr, "@")
				if len(parts) == 2 {
					componentName := strings.TrimSpace(parts[0])
					componentVersion := strings.TrimSpace(parts[1])
					components[componentName] = componentVersion
				}
			}
		}
	}

	a.logger.Sugar().Infof("blob content: %v", blob)
	return components, nil
}
