// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ResourceBuilder is a helper struct to build resources predefined in metadata.yaml.
// The ResourceBuilder is not thread-safe and must not to be used in multiple goroutines.
type ResourceBuilder struct {
	config ResourceAttributesConfig
	res    pcommon.Resource
}

// NewResourceBuilder creates a new ResourceBuilder. This method should be called on the start of the application.
func NewResourceBuilder(rac ResourceAttributesConfig) *ResourceBuilder {
	return &ResourceBuilder{
		config: rac,
		res:    pcommon.NewResource(),
	}
}

// SetOrganizationName sets provided value as "organization.name" attribute.
func (rb *ResourceBuilder) SetOrganizationName(val string) {
	if rb.config.OrganizationName.Enabled {
		rb.res.Attributes().PutStr("organization.name", val)
	}
}

// SetTeamName sets provided value as "team.name" attribute.
func (rb *ResourceBuilder) SetTeamName(val string) {
	if rb.config.TeamName.Enabled {
		rb.res.Attributes().PutStr("team.name", val)
	}
}

// SetVcsVendorName sets provided value as "vcs.vendor.name" attribute.
func (rb *ResourceBuilder) SetVcsVendorName(val string) {
	if rb.config.VcsVendorName.Enabled {
		rb.res.Attributes().PutStr("vcs.vendor.name", val)
	}
}

// Emit returns the built resource and resets the internal builder state.
func (rb *ResourceBuilder) Emit() pcommon.Resource {
	r := rb.res
	rb.res = pcommon.NewResource()
	return r
}
