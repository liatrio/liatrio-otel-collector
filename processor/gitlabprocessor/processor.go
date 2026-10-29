package gitlabprocessor // import "github.com/liatrio/liatrio-otel-collector/processor/gitlabprocessor"

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type pipelineProcessor struct {
	logger             *zap.Logger
	cfg                *Config
	GetPipeCompAttrsFn func(ctx context.Context, fullPath string, revision string) (map[string]string, error)
}

// newLogAttributesProcessor returns a processor that modifies attributes of a
// log record. To construct the attributes processors, the use of the factory
// methods are required in order to validate the inputs.
func newLogProcessor(_ context.Context, logger *zap.Logger, cfg *Config) *pipelineProcessor {
	p := &pipelineProcessor{
		logger: logger,
		cfg:    cfg,
	}
	p.GetPipeCompAttrsFn = p.getPipeCompAttrs
	return p
}

func (a *pipelineProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rs := rls.At(i)
		ilss := rs.ScopeLogs()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			logs := ils.LogRecords()
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

				comps, err := a.GetPipeCompAttrsFn(ctx, fullPath.AsString(), revision.AsString())
				if err != nil {
					a.logger.Error("error", zap.String("error", err.Error()))
					continue
				}

				// Process each component and add as attributes
				for compPath, version := range comps {
					attrName := "component." + compPath + ".version"
					lr.Attributes().PutStr(attrName, version)
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

func (a *pipelineProcessor) getPipeCompAttrs(ctx context.Context, fullPath string, revision string) (comps map[string]string, err error) {
	// Enable the ability to override the endpoint for self-hosted gitlab instances
	graphCURL := "https://gitlab.com/api/graphql"

	if a.cfg.ClientConfig.Endpoint != "" {
		var err error

		graphCURL, err = url.JoinPath(a.cfg.ClientConfig.Endpoint, "api/graphql")
		if err != nil {
			a.logger.Sugar().Errorf("error: %s", err.Error())
		}
	}

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
		a.logger.Error("error getting blob content for repo", zap.String("repo", fullPath), zap.String("error", err.Error()))
		return nil, err
	}

	if blob.Project.Id == "" {
		a.logger.Debug("no project id found for repo", zap.String("repo", fullPath))
		return nil, nil
	}

	if len(blob.Project.Repository.Blobs.GetNodes()) == 0 {
		a.logger.Debug("no blob content found for repo", zap.String("repo", fullPath))
		return nil, nil
	}

	raw := blob.Project.Repository.Blobs.GetNodes()[0].RawBlob

	config, err := getCiConfigData(context.Background(), graphClient, fullPath, revision, raw)

	if err != nil {
		a.logger.Error("error getting ci config data for repo", zap.String("repo", fullPath), zap.String("error", err.Error()))
		return nil, err
	}
	if len(config.CiConfig.Errors) > 0 {
		a.logger.Debug("graphql call for ci config returned errors for repo", zap.String("repo", fullPath), zap.Any("errors", config.CiConfig.Errors))
		return nil, nil
	}
	if len(config.CiConfig.Includes) == 0 {
		a.logger.Debug("no includes found for repo", zap.String("repo", fullPath))
	}
	for _, include := range config.CiConfig.Includes {
		switch include.Type {
		case "component":
			componentParts := strings.Split(include.Location, "@")
			if len(componentParts) == 2 {
				componentName := strings.TrimPrefix(componentParts[0], "gitlab.com/")
				componentVersion := componentParts[1]
				components[componentName] = componentVersion
			}
		case "file":
			componentParts := strings.Split(include.Blob, "/-/")
			if len(componentParts) == 2 {
				//concat the location (which is the file name) with the componentName (which
				//is the path of the project) then trim off the "https://gitlab.com/" part at
				//the beginning
				componentName := strings.TrimPrefix(componentParts[0], "https://gitlab.com/")
				componentName += include.Location
				//the only version we get for file includes is the commit sha in the link to
				//the blob of the file. Parse out the commit sha from the blob link.
				componentVersion := strings.Split(componentParts[1], "/")[1]

				components[componentName] = componentVersion
			}
		case "local":
			//for local includes, we'll just concat the location (path in the repo) with the
			//full namespace of the project from the blob link.
			componentParts := strings.Split(include.Blob, "/-/")
			if len(componentParts) == 2 {
				componentName := strings.TrimPrefix(componentParts[0], "https://gitlab.com/")
				componentName += "/" + include.Location
				componentVersion := "local"

				components[componentName] = componentVersion
			}
		}
	}

	return components, nil
}
