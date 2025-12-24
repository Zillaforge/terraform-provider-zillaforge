// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.List = &securityGroupIDsValidator{}

// securityGroupIDsValidator validates that security group IDs exist in the ZillaForge platform.
type securityGroupIDsValidator struct{}

// SecurityGroupIDsValidator returns a validator that checks if security group IDs exist.
func SecurityGroupIDsValidator() validator.List {
	return &securityGroupIDsValidator{}
}

func (v *securityGroupIDsValidator) Description(ctx context.Context) string {
	return "list must contain at least one valid security group ID from the ZillaForge platform"
}

func (v *securityGroupIDsValidator) MarkdownDescription(ctx context.Context) string {
	return "list must contain at least one valid security group ID from the ZillaForge platform (use `zillaforge_security_groups` data source)"
}

func (v *securityGroupIDsValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	// Skip validation if value is unknown or null
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	// Check that at least one security group ID is provided
	if len(req.ConfigValue.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Security Group IDs",
			"At least one security group ID is required per network attachment. Use the zillaforge_security_groups data source to get valid security group IDs.",
		)
		return
	}

	// Full validation of individual IDs requires API call which will be done during Create/Update
}
