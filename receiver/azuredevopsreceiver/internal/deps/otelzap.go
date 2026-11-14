//go:build tools
// +build tools

// Package deps pins transitive modules that the collector distribution compiles
// but this module does not import directly.
//
// go.opentelemetry.io/otel/log v1.47.0 removed log.KeyValue. Collector service
// v0.158.0 still requires go.opentelemetry.io/contrib/bridges/otelzap v0.19.0,
// which does not compile against that API. v0.21.0 is the matching release;
// NewCore, WithLoggerProvider, and WithAttributes keep the signatures the
// collector calls. The import is build-tagged so it is not linked into the
// receiver, and go mod tidy still records the requirement that the
// distribution build selects.
package deps

import _ "go.opentelemetry.io/contrib/bridges/otelzap"
