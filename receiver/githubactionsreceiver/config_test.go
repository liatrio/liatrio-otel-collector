// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
)

// only one validate check so far
func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		expect error
		conf   Config
	}{
		{
			desc:   "Missing valid endpoint",
			expect: errMissingEndpointFromConfig,
			conf: Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "",
				},
			},
		},
		{
			desc:   "Valid Secret",
			expect: nil,
			conf: Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "localhost:8080",
				},
				Secret: "mysecret",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.conf.Validate()
			if test.expect == nil {
				require.NoError(t, err)
				require.Equal(t, "mysecret", test.conf.Secret)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expect.Error())
			}
		})
	}
}
