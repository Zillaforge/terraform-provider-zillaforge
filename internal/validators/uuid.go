// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &uuidValidator{}

// uuidValidator validates UUID strings (RFC 4122 format).
type uuidValidator struct{}

// UUIDValidator returns a validator for UUID strings.
func UUIDValidator() validator.String {
	return &uuidValidator{}
}

func (v *uuidValidator) Description(ctx context.Context) string {
	return "value must be a valid UUID (RFC 4122 format)"
}

func (v *uuidValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid UUID (RFC 4122 format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)"
}

func (v *uuidValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()

	// UUID regex pattern (RFC 4122)
	// Format: 8-4-4-4-12 hexadecimal digits
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	if !uuidPattern.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid UUID",
			fmt.Sprintf("The value '%s' is not a valid UUID. Expected format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (lowercase hexadecimal).", value),
		)
		return
	}
}
