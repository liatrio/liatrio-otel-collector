package ldapreceiver

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		expectErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Interval:     "30s",
				SearchFilter: "cn=test",
				Endpoint:     "ldap://example.com:389",
				BaseDN:       "dc=example,dc=com",
			},
			expectErr: false,
		},
		{
			name: "invalid interval",
			cfg: &Config{
				Interval:     "5s",
				SearchFilter: "cn=test",
				Endpoint:     "ldap://example.com:389",
				BaseDN:       "dc=example,dc=com",
			},
			expectErr: true,
		},
		{
			name: "missing search_filter",
			cfg: &Config{
				Interval: "30s",
				Endpoint: "ldap://example.com:389",
				BaseDN:   "dc=example,dc=com",
			},
			expectErr: true,
		},
		{
			name: "missing base_dn",
			cfg: &Config{
				Interval:     "30s",
				SearchFilter: "cn=test",
				Endpoint:     "ldap://example.com:389",
			},
			expectErr: true,
		},
		{
			name: "missing endpoint",
			cfg: &Config{
				Interval:     "30s",
				SearchFilter: "cn=test",
				BaseDN:       "dc=example,dc=com",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			} else if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
