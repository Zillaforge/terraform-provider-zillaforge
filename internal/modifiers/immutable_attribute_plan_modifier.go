// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package modifiers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

var _ planmodifier.String = &immutableAttributePlanModifier{}

// immutableAttributePlanModifier rejects plan-time changes to an attribute when
// the resource already exists in state. This prevents users from changing
// attributes that are not supported for in-place updates.
type immutableAttributePlanModifier struct {
	AttributeName string
}

// ImmutableAttributePlanModifier returns a plan modifier that rejects changes
// to an attribute on existing resources at plan time.
func ImmutableAttributePlanModifier(attrName string) planmodifier.String {
	return &immutableAttributePlanModifier{AttributeName: attrName}
}

func (m *immutableAttributePlanModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("Rejects in-place changes to '%s' attribute", m.AttributeName)
}

func (m *immutableAttributePlanModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Rejects in-place changes to `%s` attribute", m.AttributeName)
}

func (m *immutableAttributePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the state is null, this is a create operation - allow
	if req.StateValue.IsNull() {
		return
	}

	// If the plan value is unknown, we cannot determine if it's changing - allow
	if req.PlanValue.IsUnknown() {
		return
	}

	// If both plan and state are null, allow (shouldn't happen but be defensive)
	if req.PlanValue.IsNull() && req.StateValue.IsNull() {
		return
	}

	// Compare values - if different, reject the change
	if !req.PlanValue.Equal(req.StateValue) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			fmt.Sprintf("Unsupported Change: %s", m.AttributeName),
			fmt.Sprintf("Changing '%s' is not supported in-place and is rejected by the provider. To change this attribute, you must recreate the resource manually or use the ZillaForge platform directly.", m.AttributeName),
		)
	}
}
