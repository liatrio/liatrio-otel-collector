//go:build tools
// +build tools

package tools // import "github.com/liatrio/liatrio-otel-collector/internal/tools"

// This file exists to ensure consistent versioning and tooling installs based on
// https://go.dev/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module


import (
    _ "github.com/golangci/golangci-lint/cmd/golangci-lint"
    _ "github.com/Khan/genqlient"
    _ "honnef.co/go/tools/cmd/staticcheck"
    _ "github.com/securego/gosec/v2/cmd/gosec"
    _ "golang.org/x/tools/cmd/goimports"
    _ "github.com/open-telemetry/opentelemetry-collector-contrib/cmd/mdatagen"
    _ "golang.org/x/vuln/cmd/govulncheck"
    _ "go.opentelemetry.io/collector/cmd/builder"
    _ "github.com/goreleaser/goreleaser"
    _ "go.opentelemetry.io/build-tools/multimod"
    _ "go.opentelemetry.io/build-tools/crosslink"
)

