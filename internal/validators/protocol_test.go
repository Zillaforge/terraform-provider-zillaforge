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

func TestProtocolValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		// Valid protocols - lowercase
		{
			name:        "tcp lowercase",
			value:       "tcp",
			expectError: false,
		},
		{
			name:        "udp lowercase",
			value:       "udp",
			expectError: false,
		},
		{
			name:        "icmp lowercase",
			value:       "icmp",
			expectError: false,
		},
		{
			name:        "any lowercase",
			value:       "any",
			expectError: false,
		},

		// Valid protocols - uppercase (case-insensitive)
		{
			name:        "TCP uppercase",
			value:       "TCP",
			expectError: false,
		},
		{
			name:        "UDP uppercase",
			value:       "UDP",
			expectError: false,
		},
		{
			name:        "ICMP uppercase",
			value:       "ICMP",
			expectError: false,
		},
		{
			name:        "ANY uppercase",
			value:       "ANY",
			expectError: false,
		},

		// Valid protocols - mixed case
		{
			name:        "Tcp mixed case",
			value:       "Tcp",
			expectError: false,
		},
		{
			name:        "Udp mixed case",
			value:       "Udp",
			expectError: false,
		},
		{
			name:        "Icmp mixed case",
			value:       "Icmp",
			expectError: false,
		},
		{
			name:        "Any mixed case",
			value:       "Any",
			expectError: false,
		},

		// Invalid protocols
		{
			name:        "all (not a valid protocol)",
			value:       "all",
			expectError: true,
		},
		{
			name:        "http (application protocol)",
			value:       "http",
			expectError: true,
		},
		{
			name:        "ssh (application protocol)",
			value:       "ssh",
			expectError: true,
		},
		{
			name:        "numeric protocol 6",
			value:       "6",
			expectError: true,
		},
		{
			name:        "numeric protocol 17",
			value:       "17",
			expectError: true,
		},
		{
			name:        "empty string",
			value:       "",
			expectError: true,
		},
		{
			name:        "invalid text",
			value:       "not-a-protocol",
			expectError: true,
		},
		{
			name:        "icmpv6",
			value:       "icmpv6",
			expectError: true,
		},
		{
			name:        "esp",
			value:       "esp",
			expectError: true,
		},
		{
			name:        "gre",
			value:       "gre",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			Protocol().ValidateString(context.Background(), req, resp)

			if tt.expectError {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected error for value '%s', but got none", tt.value)
				}
				// Verify error message contains "Invalid Protocol"
				found := false
				for _, diag := range resp.Diagnostics.Errors() {
					if diag.Summary() == "Invalid Protocol" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected 'Invalid Protocol' error, but got: %v", resp.Diagnostics.Errors())
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error for value '%s', but got: %v", tt.value, resp.Diagnostics.Errors())
				}
			}
		})
	}
}

func TestProtocolValidator_UnknownAndNull(t *testing.T) {
	t.Parallel()

	// Test unknown value
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	Protocol().ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal("expected no error for unknown value")
	}

	// Test null value
	req = validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringNull(),
	}
	resp = &validator.StringResponse{}

	Protocol().ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal("expected no error for null value")
	}
}
