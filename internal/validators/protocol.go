// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &protocolValidator{}

// protocolValidator validates protocol strings for security group rules.
// Allowed values: tcp, udp, icmp, any (case-insensitive, normalized to lowercase in state).
type protocolValidator struct{}

// Protocol returns a validator for protocol strings.
func Protocol() validator.String {
	return &protocolValidator{}
}

func (v *protocolValidator) Description(ctx context.Context) string {
	return "value must be one of: tcp, udp, icmp, any (case-insensitive)"
}

func (v *protocolValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be one of: `tcp`, `udp`, `icmp`, `any` (case-insensitive)"
}

func (v *protocolValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := strings.ToLower(req.ConfigValue.ValueString())

	// Valid protocols
	validProtocols := map[string]bool{
		"tcp":  true,
		"udp":  true,
		"icmp": true,
		"any":  true,
	}

	if !validProtocols[value] {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Protocol",
			fmt.Sprintf("Protocol '%s' is not valid. Must be one of: tcp, udp, icmp, any (case-insensitive).", req.ConfigValue.ValueString()),
		)
		return
	}
}
