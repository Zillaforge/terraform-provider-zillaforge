// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &cidrValidator{}

// cidrValidator validates CIDR notation strings for both IPv4 and IPv6 addresses.
type cidrValidator struct{}

// CIDR returns a validator for CIDR notation strings.
func CIDR() validator.String {
	return &cidrValidator{}
}

func (v *cidrValidator) Description(ctx context.Context) string {
	return "value must be a valid CIDR notation (e.g., '0.0.0.0/0', '192.168.1.0/24', '::/0', '2001:db8::/32')"
}

func (v *cidrValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid CIDR notation (e.g., `0.0.0.0/0`, `192.168.1.0/24`, `::/0`, `2001:db8::/32`)"
}

func (v *cidrValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Parse CIDR using Go's net.ParseCIDR
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR Notation",
			fmt.Sprintf("Value '%s' is not a valid CIDR notation. Must be in format 'IP/prefix' (e.g., '10.0.0.0/8' for IPv4 or '2001:db8::/32' for IPv6). Error: %s", value, err.Error()),
		)
		return
	}
}
