// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCIDRValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		// Valid IPv4 CIDR
		{
			name:        "IPv4 all addresses",
			value:       "0.0.0.0/0",
			expectError: false,
		},
		{
			name:        "IPv4 private network class A",
			value:       "10.0.0.0/8",
			expectError: false,
		},
		{
			name:        "IPv4 private network class B",
			value:       "172.16.0.0/12",
			expectError: false,
		},
		{
			name:        "IPv4 private network class C",
			value:       "192.168.1.0/24",
			expectError: false,
		},
		{
			name:        "IPv4 single host /32",
			value:       "203.0.113.5/32",
			expectError: false,
		},
		{
			name:        "IPv4 subnet /28",
			value:       "192.168.100.0/28",
			expectError: false,
		},

		// Valid IPv6 CIDR
		{
			name:        "IPv6 all addresses",
			value:       "::/0",
			expectError: false,
		},
		{
			name:        "IPv6 documentation prefix",
			value:       "2001:db8::/32",
			expectError: false,
		},
		{
			name:        "IPv6 link-local",
			value:       "fe80::/10",
			expectError: false,
		},
		{
			name:        "IPv6 single host /128",
			value:       "2001:db8::1/128",
			expectError: false,
		},
		{
			name:        "IPv6 subnet /64",
			value:       "2001:db8:abcd::/64",
			expectError: false,
		},
		{
			name:        "IPv6 compressed notation",
			value:       "2001:db8::8a2e:370:7334/64",
			expectError: false,
		},

		// Invalid CIDR - format errors
		{
			name:        "missing prefix",
			value:       "192.168.1.0",
			expectError: true,
		},
		{
			name:        "invalid IP address",
			value:       "256.1.1.1/24",
			expectError: true,
		},
		{
			name:        "invalid prefix too large IPv4",
			value:       "192.168.1.0/33",
			expectError: true,
		},
		{
			name:        "invalid prefix too large IPv6",
			value:       "2001:db8::/129",
			expectError: true,
		},
		{
			name:        "invalid prefix negative",
			value:       "192.168.1.0/-1",
			expectError: true,
		},
		{
			name:        "empty string",
			value:       "",
			expectError: true,
		},
		{
			name:        "plain text",
			value:       "not-a-cidr",
			expectError: true,
		},
		{
			name:        "IP without CIDR notation",
			value:       "192.168.1.1",
			expectError: true,
		},
		{
			name:        "malformed IPv6",
			value:       "gggg::/32",
			expectError: true,
		},
		{
			name:        "host bits set (non-canonical)",
			value:       "192.168.1.5/24",
			expectError: false, // Go's ParseCIDR allows this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			CIDR().ValidateString(context.Background(), req, resp)

			if tt.expectError {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected error for value '%s', but got none", tt.value)
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error for value '%s', but got: %v", tt.value, resp.Diagnostics.Errors())
				}
			}
		})
	}
}

func TestCIDRValidator_UnknownAndNull(t *testing.T) {
	t.Parallel()

	// Test unknown value
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	CIDR().ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal("expected no error for unknown value")
	}

	// Test null value
	req = validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringNull(),
	}
	resp = &validator.StringResponse{}

	CIDR().ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal("expected no error for null value")
	}
}
