package gitlabprocessor

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Khan/genqlient/graphql"
	// "github.com/aerospike/aerospike-client-go/v7/logger"
	// "github.com/xanzy/go-gitlab"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/attraction"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
)

type logProcessor struct {
	logger   *zap.Logger
	cfg      *Config
	client   *http.Client
	attrProc *attraction.AttrProc
	// skipExpr expr.BoolExpr[ottllog.TransformContext]
}

// newLogAttributesProcessor returns a processor that modifies attributes of a
// log record. To construct the attributes processors, the use of the factory
// methods are required in order to validate the inputs.
func newLogProcessor(logger *zap.Logger, attrProc *attraction.AttrProc, cfg Config) *logProcessor {
	return &logProcessor{
		logger:   logger,
		cfg:      &cfg,
		attrProc: attrProc,
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

				attrs, err := a.getPipeCompAttrs(ctx, fullPath.AsString(), revision.AsString())
				if err != nil {
					a.logger.Sugar().Errorf("error: %v", err)
				}

                a.logger.Sugar().Infof("attrs: %v", attrs)

				a.attrProc.Process(ctx, lr.Attributes())

				// if a.skipExpr != nil {
				// 	skip, err := a.skipExpr.Eval(ctx, ottllog.NewTransformContext(lr, library, resource, ils, rs))
				// 	if err != nil {
				// 		return ld, err
				// 	}
				// 	if skip {
				// 		continue
				// 	}
				// }
				//
				// a.attrProc.Process(ctx, a.logger, lr.Attributes())
			}
		}

	}

	// for i := 0; i < rls.Len(); i++ {
	return ld, nil
}

// func (a *logProcessor) getPipeCompAttrs(ctx context.Context, fullPath string, revision string) (attrs []string, err error) {
func (a *logProcessor) getPipeCompAttrs(ctx context.Context, fullPath string, revision string) (err error) {
	// a.client, err = a.cfg.ToClient(ctx, host, a.settings)

	// Enable the ability to override the endpoint for self-hosted gitlab instances
	graphCURL := "https://gitlab.com/api/graphql"
	// restCURL := "https://gitlab.com/"

	if a.cfg.ClientConfig.Endpoint != "" {
		var err error

		graphCURL, err = url.JoinPath(a.cfg.ClientConfig.Endpoint, "api/graphql")
		if err != nil {
			a.logger.Sugar().Errorf("error: %v", err)
		}

		// restCURL, err = url.JoinPath(a.cfg.ClientConfig.Endpoint, "/")
		// if err != nil {
		// 	a.logger.Sugar().Errorf("error: %v", err)
		// }
	}

	graphClient := graphql.NewClient(graphCURL, a.client)
	// restClient, err := gitlab.NewClient("", gitlab.WithHTTPClient(a.client), gitlab.WithBaseURL(restCURL))
	// if err != nil {
	// 	a.logger.Sugar().Errorf("error: %v", err)
	// }

	blog, err := getBlobContent(ctx, graphClient, "projectPath", "path", "sha")
	if err != nil {
		a.logger.Sugar().Errorf("error: %v", err)
	}
	a.logger.Sugar().Infof("blob content: %v", blog)
	return nil
}
