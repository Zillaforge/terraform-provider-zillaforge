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

func TestPortRangeValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        string
		expectError  bool
		errorSummary string
	}{
		// Valid cases
		{
			name:        "literal all lowercase",
			value:       "all",
			expectError: false,
		},
		{
			name:        "literal ALL uppercase",
			value:       "ALL",
			expectError: false,
		},
		{
			name:        "literal All mixed case",
			value:       "All",
			expectError: false,
		},
		{
			name:        "single port minimum",
			value:       "1",
			expectError: false,
		},
		{
			name:        "single port maximum",
			value:       "65535",
			expectError: false,
		},
		{
			name:        "single port HTTP",
			value:       "80",
			expectError: false,
		},
		{
			name:        "single port HTTPS",
			value:       "443",
			expectError: false,
		},
		{
			name:        "port range valid",
			value:       "8000-8100",
			expectError: false,
		},
		{
			name:        "port range same start and end",
			value:       "22-22",
			expectError: false,
		},
		{
			name:        "port range full range",
			value:       "1-65535",
			expectError: false,
		},

		// Invalid cases - port out of range
		{
			name:         "single port zero",
			value:        "0",
			expectError:  true,
			errorSummary: "Port Out of Range",
		},
		{
			name:         "single port above maximum",
			value:        "65536",
			expectError:  true,
			errorSummary: "Port Out of Range",
		},
		{
			name:         "port range start zero",
			value:        "0-100",
			expectError:  true,
			errorSummary: "Port Out of Range",
		},
		{
			name:         "port range end too high",
			value:        "8000-70000",
			expectError:  true,
			errorSummary: "Port Out of Range",
		},

		// Invalid cases - range logic errors
		{
			name:         "port range start greater than end",
			value:        "8100-8000",
			expectError:  true,
			errorSummary: "Invalid Port Range",
		},

		// Invalid cases - format errors
		{
			name:         "invalid format letters",
			value:        "http",
			expectError:  true,
			errorSummary: "Invalid Port Format",
		},
		{
			name:         "invalid format empty",
			value:        "",
			expectError:  true,
			errorSummary: "Invalid Port Format",
		},
		{
			name:         "invalid format multiple hyphens",
			value:        "80-90-100",
			expectError:  true,
			errorSummary: "Invalid Port Format",
		},
		{
			name:         "invalid format spaces",
			value:        "80 - 90",
			expectError:  true,
			errorSummary: "Invalid Port Format",
		},
		{
			name:         "invalid format negative",
			value:        "-80",
			expectError:  true,
			errorSummary: "Port Out of Range", // strconv.Atoi parses as negative number
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			PortRange().ValidateString(context.Background(), req, resp)

			if tt.expectError {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected error for value '%s', but got none", tt.value)
				}
				if tt.errorSummary != "" {
					found := false
					for _, diag := range resp.Diagnostics.Errors() {
						if diag.Summary() == tt.errorSummary {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error summary '%s', but got: %v", tt.errorSummary, resp.Diagnostics.Errors())
					}
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error for value '%s', but got: %v", tt.value, resp.Diagnostics.Errors())
				}
			}
		})
	}
}

func TestPortRangeValidator_UnknownAndNull(t *testing.T) {
	t.Parallel()

	// Test unknown value
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	PortRange().ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal("expected no error for unknown value")
	}

	// Test null value
	req = validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringNull(),
	}
	resp = &validator.StringResponse{}

	PortRange().ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal("expected no error for null value")
	}
}
