// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package modifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IPAddressUnknownOnNetworkChangeModifier is a plan modifier that marks ip_address
// as unknown when the network_id in the same network_attachment block changes.
// This prevents Terraform from showing inconsistent state errors when changing networks.
type IPAddressUnknownOnNetworkChangeModifier struct{}

func (m IPAddressUnknownOnNetworkChangeModifier) Description(ctx context.Context) string {
	return "Marks ip_address as unknown when network_id changes in the same network_attachment block"
}

func (m IPAddressUnknownOnNetworkChangeModifier) MarkdownDescription(ctx context.Context) string {
	return "Marks `ip_address` as unknown when `network_id` changes in the same `network_attachment` block"
}

func (m IPAddressUnknownOnNetworkChangeModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we're creating the resource, do nothing (ip_address will be computed or use config)
	if req.State.Raw.IsNull() {
		return
	}

	// If we're destroying the resource, do nothing
	if req.Plan.Raw.IsNull() {
		return
	}

	// Extract plan and state objects for this network_attachment block
	// The path structure is: network_attachment[idx].ip_address
	// We need to go up one level to get the object, then access network_id

	planPath := req.Path.ParentPath()

	var planAttachment types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, planPath, &planAttachment)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateAttachment types.Object
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, planPath, &stateAttachment)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if both are valid objects
	if planAttachment.IsNull() || planAttachment.IsUnknown() || stateAttachment.IsNull() || stateAttachment.IsUnknown() {
		return
	}

	// Get network_id from both
	planAttrs := planAttachment.Attributes()
	stateAttrs := stateAttachment.Attributes()

	planNetworkID, ok := planAttrs["network_id"].(types.String)
	if !ok {
		return
	}

	stateNetworkID, ok := stateAttrs["network_id"].(types.String)
	if !ok {
		return
	}

	// If network_id changed, mark ip_address as unknown (it will be reassigned by the cloud)
	if !planNetworkID.Equal(stateNetworkID) {
		resp.PlanValue = types.StringUnknown()
		return
	}

	// If network_id hasn't changed AND ip_address is not configured (null in config),
	// preserve the state value to avoid unnecessary diffs
	if req.ConfigValue.IsNull() && !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

func IPAddressUnknownOnNetworkChange() planmodifier.String {
	return IPAddressUnknownOnNetworkChangeModifier{}
}

// FloatingIPPreserveStateModifier is a plan modifier for the computed floating_ip attribute
// that preserves the state value unless floating_ip_id or network_id changes.
// This prevents unnecessary diffs when only runtime-only attributes change.
type FloatingIPPreserveStateModifier struct{}

func (m FloatingIPPreserveStateModifier) Description(ctx context.Context) string {
	return "Preserves floating_ip state value unless floating_ip_id or network_id changes"
}

func (m FloatingIPPreserveStateModifier) MarkdownDescription(ctx context.Context) string {
	return "Preserves `floating_ip` state value unless `floating_ip_id` or `network_id` changes"
}

func (m FloatingIPPreserveStateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we're creating the resource, floating_ip will be computed
	if req.State.Raw.IsNull() {
		return
	}

	// If we're destroying the resource, do nothing
	if req.Plan.Raw.IsNull() {
		return
	}

	// Extract plan and state objects for this network_attachment block
	planPath := req.Path.ParentPath()

	var planAttachment types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, planPath, &planAttachment)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateAttachment types.Object
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, planPath, &stateAttachment)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if both are valid objects
	if planAttachment.IsNull() || planAttachment.IsUnknown() || stateAttachment.IsNull() || stateAttachment.IsUnknown() {
		return
	}

	planAttrs := planAttachment.Attributes()
	stateAttrs := stateAttachment.Attributes()

	// Get network_id and floating_ip_id from plan and state
	planNetworkID, planNetOk := planAttrs["network_id"].(types.String)
	stateNetworkID, stateNetOk := stateAttrs["network_id"].(types.String)
	planFloatingIPID, planFIPOk := planAttrs["floating_ip_id"].(types.String)
	stateFloatingIPID, stateFIPOk := stateAttrs["floating_ip_id"].(types.String)

	if !planNetOk || !stateNetOk || !planFIPOk || !stateFIPOk {
		return
	}

	// If network_id changed, floating_ip will be recomputed (mark as unknown)
	if !planNetworkID.Equal(stateNetworkID) {
		resp.PlanValue = types.StringUnknown()
		return
	}

	// If floating_ip_id changed, floating_ip will be recomputed (mark as unknown)
	if !planFloatingIPID.Equal(stateFloatingIPID) {
		resp.PlanValue = types.StringUnknown()
		return
	}

	// If neither network_id nor floating_ip_id changed, ALWAYS preserve the state value
	// This includes null values - we want to avoid showing a diff when nothing changed
	resp.PlanValue = req.StateValue
}

func FloatingIPPreserveState() planmodifier.String {
	return FloatingIPPreserveStateModifier{}
}
