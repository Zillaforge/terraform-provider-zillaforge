// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &flavorIDValidator{}

// flavorIDValidator validates that a flavor ID exists in the ZillaForge platform.
type flavorIDValidator struct{}

// FlavorIDValidator returns a validator that checks if a flavor ID exists.
func FlavorIDValidator() validator.String {
	return &flavorIDValidator{}
}

func (v *flavorIDValidator) Description(ctx context.Context) string {
	return "value must be a valid flavor ID from the ZillaForge platform"
}

func (v *flavorIDValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid flavor ID from the ZillaForge platform (use `zillaforge_flavors` data source)"
}

func (v *flavorIDValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	// Note: Actual validation against the cloud-sdk would require the client instance
	// For now, we perform basic ID format validation
	// TODO: Implement cloud-sdk lookup when client is available in validator context
	value := req.ConfigValue.ValueString()
	if value == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Flavor ID",
			"Flavor ID cannot be empty. Use the zillaforge_flavors data source to get valid flavor IDs.",
		)
		return
	}

	// Basic validation: flavor IDs should not be empty
	// Full validation requires API call which will be done during Create/Update
}
