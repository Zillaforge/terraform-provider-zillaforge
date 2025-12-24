// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &networkIDValidator{}

// networkIDValidator validates that a network ID exists in the ZillaForge platform.
type networkIDValidator struct{}

// NetworkIDValidator returns a validator that checks if a network ID exists.
func NetworkIDValidator() validator.String {
	return &networkIDValidator{}
}

func (v *networkIDValidator) Description(ctx context.Context) string {
	return "value must be a valid network ID from the ZillaForge platform"
}

func (v *networkIDValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid network ID from the ZillaForge platform (use `zillaforge_networks` data source)"
}

func (v *networkIDValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Network ID",
			"Network ID cannot be empty. Use the zillaforge_networks data source to get valid network IDs.",
		)
		return
	}

	// Full validation requires API call which will be done during Create/Update
}
