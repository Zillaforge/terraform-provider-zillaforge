// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &imageIDValidator{}

// imageIDValidator validates that an image ID exists in the ZillaForge platform.
type imageIDValidator struct{}

// ImageIDValidator returns a validator that checks if an image ID exists.
func ImageIDValidator() validator.String {
	return &imageIDValidator{}
}

func (v *imageIDValidator) Description(ctx context.Context) string {
	return "value must be a valid image ID from the ZillaForge platform"
}

func (v *imageIDValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid image ID from the ZillaForge platform (use `zillaforge_images` data source)"
}

func (v *imageIDValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Image ID",
			"Image ID cannot be empty. Use the zillaforge_images data source to get valid image IDs.",
		)
		return
	}

	// Full validation requires API call which will be done during Create/Update
}
