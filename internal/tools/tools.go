//go:build tools
// +build tools

package tools // import "github.com/liatrio/liatrio-otel-collector/internal/tools"

// This file exists to ensure consistent versioning and tooling installs based on
// https://go.dev/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

import (
	_ "github.com/Khan/genqlient"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/osv-scanner/cmd/osv-scanner"
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "go.opentelemetry.io/build-tools/crosslink"
	_ "go.opentelemetry.io/build-tools/multimod"
	_ "go.opentelemetry.io/collector/cmd/builder"
	_ "go.opentelemetry.io/collector/cmd/mdatagen"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
