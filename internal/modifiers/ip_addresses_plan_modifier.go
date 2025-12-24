// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package modifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// IPAddressesUnknownOnNetworkChangeModifier marks ip_addresses as unknown when
// network_attachment changes (e.g., network_id changes, NICs added/removed).
// This ensures Terraform doesn't expect specific IPs when the network topology changes.
type IPAddressesUnknownOnNetworkChangeModifier struct{}

func (m IPAddressesUnknownOnNetworkChangeModifier) Description(ctx context.Context) string {
	return "Marks ip_addresses as unknown when network_attachment changes"
}

func (m IPAddressesUnknownOnNetworkChangeModifier) MarkdownDescription(ctx context.Context) string {
	return "Marks `ip_addresses` as unknown when `network_attachment` changes"
}

func (m IPAddressesUnknownOnNetworkChangeModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	tflog.Debug(ctx, "IPAddressesUnknownOnNetworkChangeModifier called")

	// If we're creating the resource, leave as-is (will be computed)
	if req.State.Raw.IsNull() {
		tflog.Debug(ctx, "State is null (create), skipping")
		return
	}

	// If we're destroying the resource, do nothing
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "Plan is null (destroy), skipping")
		return
	}

	// Get network_attachment from both plan and state
	var planNetworkAttachment types.List
	var stateNetworkAttachment types.List

	// Get the network_attachment attribute path
	networkAttachmentPath := req.Path.ParentPath().AtName("network_attachment")

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, networkAttachmentPath, &planNetworkAttachment)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get plan network_attachment")
		return
	}

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, networkAttachmentPath, &stateNetworkAttachment)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get state network_attachment")
		return
	}

	// Compare only network topology (network_id and count), not computed values like ip_address
	// Extract network_ids from plan and state
	var planAttachments []types.Object
	var stateAttachments []types.Object

	resp.Diagnostics.Append(planNetworkAttachment.ElementsAs(ctx, &planAttachments, false)...)
	resp.Diagnostics.Append(stateNetworkAttachment.ElementsAs(ctx, &stateAttachments, false)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If number of attachments changed, network topology changed
	if len(planAttachments) != len(stateAttachments) {
		tflog.Info(ctx, "network_attachment count changed, marking ip_addresses as unknown", map[string]interface{}{
			"plan_count":  len(planAttachments),
			"state_count": len(stateAttachments),
		})
		resp.PlanValue = types.ListUnknown(types.StringType)
		return
	}

	// Check if any network_id changed
	for i := range planAttachments {
		planNetworkID := planAttachments[i].Attributes()["network_id"]
		stateNetworkID := stateAttachments[i].Attributes()["network_id"]

		if !planNetworkID.Equal(stateNetworkID) {
			tflog.Info(ctx, "network_id changed in network_attachment, marking ip_addresses as unknown", map[string]interface{}{
				"index": i,
			})
			resp.PlanValue = types.ListUnknown(types.StringType)
			return
		}
	}

	// If network topology hasn't changed, preserve the state value to avoid spurious diffs.
	// ip_addresses is computed-only and derives from network topology, so it should only
	// change when network_attachment changes.
	tflog.Debug(ctx, "network topology unchanged, preserving state value", map[string]interface{}{
		"state_is_null":    req.StateValue.IsNull(),
		"state_is_unknown": req.StateValue.IsUnknown(),
	})
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		tflog.Debug(ctx, "Set PlanValue to StateValue")
	}
}

func IPAddressesUnknownOnNetworkChange() planmodifier.List {
	return IPAddressesUnknownOnNetworkChangeModifier{}
}
