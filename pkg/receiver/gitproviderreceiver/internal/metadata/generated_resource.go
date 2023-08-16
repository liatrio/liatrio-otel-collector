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

// SetGitVendorName sets provided value as "git.vendor.name" attribute.
func (rb *ResourceBuilder) SetGitVendorName(val string) {
	if rb.config.GitVendorName.Enabled {
		rb.res.Attributes().PutStr("git.vendor.name", val)
	}
}

// SetOrganizationName sets provided value as "organization.name" attribute.
func (rb *ResourceBuilder) SetOrganizationName(val string) {
	if rb.config.OrganizationName.Enabled {
		rb.res.Attributes().PutStr("organization.name", val)
	}
}

// Emit returns the built resource and resets the internal builder state.
func (rb *ResourceBuilder) Emit() pcommon.Resource {
	r := rb.res
	rb.res = pcommon.NewResource()
	return r
}
