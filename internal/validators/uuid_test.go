// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators_test

import (
	"context"
	"testing"

	"github.com/Zillaforge/terraform-provider-zillaforge/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUUIDValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       types.String
		expectError bool
	}{
		{
			name:        "valid UUID lowercase",
			value:       types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
			expectError: false,
		},
		{
			name:        "valid UUID all zeros",
			value:       types.StringValue("00000000-0000-0000-0000-000000000000"),
			expectError: false,
		},
		{
			name:        "valid UUID all f's",
			value:       types.StringValue("ffffffff-ffff-ffff-ffff-ffffffffffff"),
			expectError: false,
		},
		{
			name:        "invalid UUID uppercase (not RFC 4122 lowercase)",
			value:       types.StringValue("550E8400-E29B-41D4-A716-446655440000"),
			expectError: true,
		},
		{
			name:        "invalid UUID missing dashes",
			value:       types.StringValue("550e8400e29b41d4a716446655440000"),
			expectError: true,
		},
		{
			name:        "invalid UUID wrong format",
			value:       types.StringValue("550e8400-e29b-41d4-a716"),
			expectError: true,
		},
		{
			name:        "invalid UUID with non-hex characters",
			value:       types.StringValue("550e8400-e29b-41d4-a716-44665544000g"),
			expectError: true,
		},
		{
			name:        "empty string",
			value:       types.StringValue(""),
			expectError: true,
		},
		{
			name:        "null value (should not error)",
			value:       types.StringNull(),
			expectError: false,
		},
		{
			name:        "unknown value (should not error)",
			value:       types.StringUnknown(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			validators.UUIDValidator().ValidateString(context.Background(), req, resp)

			if tt.expectError && !resp.Diagnostics.HasError() {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}
