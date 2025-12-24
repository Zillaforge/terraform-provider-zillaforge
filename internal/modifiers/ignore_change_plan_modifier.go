// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package modifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// ignoreChangePlanModifier sets the planned value to the state value when the
// resource already exists. This effectively ignores changes to the attribute
// after creation (no update will be planned). If the state value is null
// (resource doesn't yet exist), the plan value is left unchanged so the
// attribute can be used during create.

type ignoreChangePlanModifierString struct {
	AttributeName string
}

func IgnoreChangeAttributePlanModifierString(attrName string) planmodifier.String {
	return &ignoreChangePlanModifierString{AttributeName: attrName}
}

func (m *ignoreChangePlanModifierString) Description(ctx context.Context) string {
	return "Ignores in-place changes to '" + m.AttributeName + "' by using the state value when resource exists"
}

func (m *ignoreChangePlanModifierString) MarkdownDescription(ctx context.Context) string {
	return "Ignores in-place changes to `" + m.AttributeName + "` by using the state value when resource exists"
}

func (m *ignoreChangePlanModifierString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the state is null, this is a create - allow the plan value to be used
	if req.StateValue.IsNull() {
		return
	}

	// If the plan value is unknown, do nothing
	if req.PlanValue.IsUnknown() {
		return
	}

	// Replace the plan value with the state value to ignore changes
	resp.PlanValue = req.StateValue
}

// Bool variant.
type ignoreChangePlanModifierBool struct {
	AttributeName string
}

func IgnoreChangeAttributePlanModifierBool(attrName string) planmodifier.Bool {
	return &ignoreChangePlanModifierBool{AttributeName: attrName}
}

func (m *ignoreChangePlanModifierBool) Description(ctx context.Context) string {
	return "Ignores in-place changes to '" + m.AttributeName + "' by using the state value when resource exists"
}

func (m *ignoreChangePlanModifierBool) MarkdownDescription(ctx context.Context) string {
	return "Ignores in-place changes to `" + m.AttributeName + "` by using the state value when resource exists"
}

func (m *ignoreChangePlanModifierBool) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	// Allow on create
	if req.StateValue.IsNull() {
		return
	}

	// If the plan value is unknown, do nothing
	if req.PlanValue.IsUnknown() {
		return
	}

	resp.PlanValue = req.StateValue
}
