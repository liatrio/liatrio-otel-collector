// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package backstagereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/backstagereceiver"

import (
	"encoding/json"
	"time"

	"github.com/liatrio/liatrio-otel-collector/receiver/backstagereceiver/internal/semconv"
	"go.einride.tech/backstage/catalog"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type attrUpdaterFunc func(pcommon.Map)

func pullObjectsToLogData(event []*catalog.Entity, observedAt time.Time, config *BackstageConfig) plog.Logs {
	return entitiesToLogData(event, observedAt, config)
}

func entitiesToLogData(event []*catalog.Entity, observedAt time.Time, config *BackstageConfig, attrUpdaters ...attrUpdaterFunc) plog.Logs {
	out := plog.NewLogs()
	resourceLogs := out.ResourceLogs()
	namespaceResourceMap := make(map[string]plog.LogRecordSlice)

	for _, e := range event {
		ns := e.Metadata.Namespace

		logSlice, ok := namespaceResourceMap[ns]
		if !ok {
			rl := resourceLogs.AppendEmpty()
			resourceAttrs := rl.Resource().Attributes()
			if ns != "" {
				resourceAttrs.PutStr(semconv.AttributeEntityNamespace, ns)
			}
			sl := rl.ScopeLogs().AppendEmpty()
			logSlice = sl.LogRecords()
			namespaceResourceMap[ns] = logSlice
		}
		record := logSlice.AppendEmpty()
		record.SetObservedTimestamp(pcommon.NewTimestampFromTime(observedAt))

		attrs := record.Attributes()

		// small helper fun
		ifset := func(k, v string) {
			if v != "" {
				attrs.PutStr(k, v)
			}
		}

		ifset(semconv.AttributeEntityApiVersion, e.APIVersion)
		ifset(semconv.AttributeEntityKind, string(e.Kind))
		ifset(semconv.AttributeEntityName, e.Metadata.Name)
		ifset(semconv.AttributeEntityUID, e.Metadata.UID)

		ifset(semconv.AttributeEntityTitle, e.Metadata.Title)

		switch e.Kind {
		case catalog.EntityKindComponent:
			v, err := e.ComponentSpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityType, v.Type)
			ifset(semconv.AttributeEntityOwner, v.Owner)
			ifset(semconv.AttributeEntityLifecycle, v.Lifecycle)

		case catalog.EntityKindSystem:
			v, err := e.SystemSpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityOwner, v.Owner)

		case catalog.EntityKindDomain:
			v, err := e.DomainSpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityOwner, v.Owner)

		case catalog.EntityKindUser:
			break

		case catalog.EntityKindAPI:
			v, err := e.APISpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityType, v.Type)
			ifset(semconv.AttributeEntityOwner, v.Owner)
			ifset(semconv.AttributeEntityLifecycle, v.Lifecycle)

		case catalog.EntityKindResource:
			v, err := e.ResourceSpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityType, v.Type)
			ifset(semconv.AttributeEntityOwner, v.Owner)

		case catalog.EntityKindLocation:
			v, err := e.LocationSpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityType, v.Type)

		case catalog.EntityKindTemplate:
			v, err := e.TemplateSpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityType, v.Type)
			ifset(semconv.AttributeEntityOwner, v.Owner)

		case catalog.EntityKindGroup:
			v, err := e.GroupSpec()
			if err != nil {
				break
			}
			ifset(semconv.AttributeEntityType, v.Type)
		}

		for _, attrUpdate := range attrUpdaters {
			attrUpdate(attrs)
		}

		dest := record.Body()
		destMap := dest.SetEmptyMap()

		m := map[string]interface{}{}

		//nolint:errcheck
		json.Unmarshal(e.Raw, &m)

		//nolint:errcheck
		destMap.FromRaw(m)
	}
	return out
}
