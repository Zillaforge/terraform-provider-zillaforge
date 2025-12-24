// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	servermodels "github.com/Zillaforge/cloud-sdk/models/vps/servers"
	vpscore "github.com/Zillaforge/cloud-sdk/modules/vps/core"
	serversdk "github.com/Zillaforge/cloud-sdk/modules/vps/servers"
	resourcemodels "github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// buildCreateRequest maps Terraform plan to cloud-SDK ServerCreateRequest.
func buildCreateRequest(ctx context.Context, plan resourcemodels.ServerResourceModel) (*servermodels.ServerCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Map network_attachment blocks
	var networkAttachments []resourcemodels.NetworkAttachmentModel
	diags.Append(plan.NetworkAttachment.ElementsAs(ctx, &networkAttachments, false)...)
	if diags.HasError() {
		return nil, diags
	}

	nics := make([]servermodels.ServerNICCreateRequest, len(networkAttachments))
	for i, att := range networkAttachments {
		// Extract security_group_ids from nested block
		var sgList []types.String
		diags.Append(att.SecurityGroupIDs.ElementsAs(ctx, &sgList, false)...)
		if diags.HasError() {
			return nil, diags
		}

		securityGroupIDs := make([]string, len(sgList))
		for j, sg := range sgList {
			securityGroupIDs[j] = sg.ValueString()
		}

		fixedIP := ""
		if !att.IPAddress.IsNull() {
			fixedIP = att.IPAddress.ValueString()
		}

		nics[i] = servermodels.ServerNICCreateRequest{
			NetworkID: att.NetworkID.ValueString(),
			SGIDs:     securityGroupIDs,
			FixedIP:   fixedIP,
		}
	}

	req := &servermodels.ServerCreateRequest{
		Name:     plan.Name.ValueString(),
		FlavorID: plan.FlavorID.ValueString(),
		ImageID:  plan.ImageID.ValueString(),
		NICs:     nics,
	}

	// Add optional fields
	if !plan.Description.IsNull() {
		req.Description = plan.Description.ValueString()
	}
	if !plan.Keypair.IsNull() {
		req.KeypairID = plan.Keypair.ValueString()
	}
	if !plan.Password.IsNull() {
		// Password should be base64 encoded
		req.Password = base64.StdEncoding.EncodeToString([]byte(plan.Password.ValueString()))
	}
	if !plan.UserData.IsNull() {
		// UserData (BootScript) should be base64 encoded
		req.BootScript = base64.StdEncoding.EncodeToString([]byte(plan.UserData.ValueString()))
	}

	return req, diags
}

// buildUpdateRequestWithNetworkChanges maps changed attributes from Terraform plan to cloud-SDK ServerUpdateRequest.
// Returns the update context with server changes and network changes, and diagnostics.
func buildUpdateRequestWithNetworkChanges(ctx context.Context, plan, state resourcemodels.ServerResourceModel) (*resourcemodels.UpdateContext, diag.Diagnostics) {
	var diags diag.Diagnostics
	updateCtx := &resourcemodels.UpdateContext{
		ServerUpdate:   &servermodels.ServerUpdateRequest{},
		NetworkChanges: make(map[string]servermodels.ServerNICUpdateRequest),
		HasChanges:     false,
	}

	// Update name if changed
	if !plan.Name.Equal(state.Name) {
		updateCtx.ServerUpdate.Name = plan.Name.ValueString()
		updateCtx.HasChanges = true
		tflog.Debug(ctx, "Name changed", map[string]interface{}{
			"old": state.Name.ValueString(),
			"new": updateCtx.ServerUpdate.Name,
		})
	}

	// Update description if changed
	if !plan.Description.Equal(state.Description) {
		updateCtx.ServerUpdate.Description = plan.Description.ValueString()
		updateCtx.HasChanges = true
		tflog.Debug(ctx, "Description changed", map[string]interface{}{
			"old": state.Description.ValueString(),
			"new": updateCtx.ServerUpdate.Description,
		})
	}

	// Disallow changing flavor or image in-place: these represent platform-level
	// resize or reprovision operations and are out of scope for in-place updates.
	if !plan.FlavorID.Equal(state.FlavorID) {
		diags.AddError(
			"Unsupported Change: flavor_id",
			"Changing 'flavor_id' is a platform resize operation and is out of scope for in-place updates. Please recreate the instance or perform a manual resize in the ZillaForge platform.",
		)
		return updateCtx, diags
	}
	if !plan.ImageID.Equal(state.ImageID) {
		diags.AddError(
			"Unsupported Change: image_id",
			"Changing 'image_id' is not supported by in-place updates. This operation requires recreating the instance (replacement).",
		)
		return updateCtx, diags
	}

	// Handle network_attachment changes (including security_group_ids and network_id)
	if !plan.NetworkAttachment.Equal(state.NetworkAttachment) {
		// Parse network attachments
		var planAttachments, stateAttachments []resourcemodels.NetworkAttachmentModel
		diags.Append(plan.NetworkAttachment.ElementsAs(ctx, &planAttachments, false)...)
		diags.Append(state.NetworkAttachment.ElementsAs(ctx, &stateAttachments, false)...)
		if diags.HasError() {
			return updateCtx, diags
		}

		// Build a map of network IDs to plan and state attachments for comparison
		planByNetwork := make(map[string]resourcemodels.NetworkAttachmentModel)
		stateByNetwork := make(map[string]resourcemodels.NetworkAttachmentModel)

		for _, att := range planAttachments {
			planByNetwork[att.NetworkID.ValueString()] = att
		}
		for _, att := range stateAttachments {
			stateByNetwork[att.NetworkID.ValueString()] = att
		}

		// Track NICs to delete (networks in state but not in plan)
		for networkID := range stateByNetwork {
			if _, exists := planByNetwork[networkID]; !exists {
				updateCtx.NetworksToDelete = append(updateCtx.NetworksToDelete, networkID)
				updateCtx.HasChanges = true
				tflog.Debug(ctx, "Network to be removed", map[string]interface{}{
					"network_id": networkID,
				})
			}
		}

		// Track NICs to create (networks in plan but not in state)
		for networkID, planAtt := range planByNetwork {
			if _, exists := stateByNetwork[networkID]; !exists {
				// Extract security_group_ids for the new NIC
				var sgList []types.String
				diags.Append(planAtt.SecurityGroupIDs.ElementsAs(ctx, &sgList, false)...)
				if diags.HasError() {
					return updateCtx, diags
				}

				securityGroupIDs := make([]string, len(sgList))
				for j, sg := range sgList {
					securityGroupIDs[j] = sg.ValueString()
				}

				// When creating a NEW NIC (network doesn't exist in state), do NOT use
				// ip_address from plan because it might be from an old network that was replaced.
				// The cloud will auto-assign an IP, which will be reflected in the final state.
				fixedIP := ""

				updateCtx.NetworksToCreate = append(updateCtx.NetworksToCreate, servermodels.ServerNICCreateRequest{
					NetworkID: networkID,
					SGIDs:     securityGroupIDs,
					FixedIP:   fixedIP,
				})
				updateCtx.HasChanges = true
				tflog.Debug(ctx, "Network to be added", map[string]interface{}{
					"network_id": networkID,
				})
			}
		}

		// Check for security_group_ids changes in existing networks
		for networkID, stateAtt := range stateByNetwork {
			if planAtt, exists := planByNetwork[networkID]; exists {
				// Check if security_group_ids changed
				if !planAtt.SecurityGroupIDs.Equal(stateAtt.SecurityGroupIDs) {
					// Extract security_group_ids
					var sgList []types.String
					diags.Append(planAtt.SecurityGroupIDs.ElementsAs(ctx, &sgList, false)...)
					if diags.HasError() {
						return updateCtx, diags
					}

					securityGroupIDs := make([]string, len(sgList))
					for j, sg := range sgList {
						securityGroupIDs[j] = sg.ValueString()
					}

					updateCtx.NetworkChanges[networkID] = servermodels.ServerNICUpdateRequest{
						SGIDs: securityGroupIDs,
					}

					tflog.Debug(ctx, "Security group IDs changed for network", map[string]interface{}{
						"network_id": networkID,
					})

					updateCtx.HasChanges = true
				}
			}
		}
	}

	return updateCtx, diags
}

// mapServerToState maps cloud-SDK ServerResource to Terraform state.
func mapServerToState(ctx context.Context, serverRes *serversdk.ServerResource) (resourcemodels.ServerResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var state resourcemodels.ServerResourceModel

	// Get embedded server
	server := serverRes.Server
	if server == nil {
		diags.AddError("Invalid Server Resource", "ServerResource.Server is nil")
		return state, diags
	}

	state.ID = types.StringValue(server.ID)
	state.Name = types.StringValue(server.Name)
	state.FlavorID = types.StringValue(server.FlavorID)
	state.ImageID = types.StringValue(server.ImageID)
	state.Status = types.StringValue(string(server.Status))
	state.CreatedAt = types.StringValue(server.CreatedAt)

	if server.Description != "" {
		state.Description = types.StringValue(server.Description)
	} else {
		state.Description = types.StringNull()
	}

	// Set optional fields
	if server.KeypairID != "" {
		state.Keypair = types.StringValue(server.KeypairID)
	} else {
		state.Keypair = types.StringNull()
	}

	// Fetch NICs to populate network_attachment
	nics, err := serverRes.NICs().List(ctx)
	if err != nil {
		diags.AddWarning("Failed to fetch server NICs", fmt.Sprintf("Could not retrieve network interfaces: %s", err.Error()))
		// Set empty list
		state.NetworkAttachment, _ = types.ListValue(
			types.ObjectType{AttrTypes: map[string]attr.Type{
				"network_id":         types.StringType,
				"ip_address":         types.StringType,
				"primary":            types.BoolType,
				"security_group_ids": types.ListType{ElemType: types.StringType},
			}},
			[]attr.Value{},
		)
	} else {
		// Map NICs to network_attachment blocks. Ensure deterministic ordering:
		// - Primary NIC appears first (if API exposes IsPrimary)
		// - Remaining NICs are sorted by NetworkID to make ordering stable
		networkAttachmentAttrTypes := map[string]attr.Type{
			"network_id":         types.StringType,
			"ip_address":         types.StringType,
			"primary":            types.BoolType,
			"security_group_ids": types.ListType{ElemType: types.StringType},
		}

		// Sort NICs by NetworkID for deterministic ordering
		sort.SliceStable(nics, func(i, j int) bool {
			return nics[i].NetworkID < nics[j].NetworkID
		})

		networkAttachments := make([]attr.Value, len(nics))
		for i, nic := range nics {
			// Map SecurityGroupIDs to types.List (sorted for deterministic ordering)
			sgIDs := make([]string, len(nic.SGIDs))
			copy(sgIDs, nic.SGIDs)
			sort.Strings(sgIDs)
			sgVals := make([]attr.Value, 0, len(sgIDs))
			for _, sg := range sgIDs {
				sgVals = append(sgVals, types.StringValue(sg))
			}
			sgList, d := types.ListValue(types.StringType, sgVals)
			diags.Append(d...)

			// Get IP address (use first address if available)
			ipAddress := types.StringNull()
			if len(nic.Addresses) > 0 {
				ipAddress = types.StringValue(nic.Addresses[0])
			}

			// No API primary flag available - use first NIC as primary (fallback)
			isPrimary := (i == 0)

			attObj, d := types.ObjectValue(networkAttachmentAttrTypes, map[string]attr.Value{
				"network_id":         types.StringValue(nic.NetworkID),
				"ip_address":         ipAddress,
				"primary":            types.BoolValue(isPrimary),
				"security_group_ids": sgList,
			})
			diags.Append(d...)
			networkAttachments[i] = attObj
		}

		networkAttachmentList, d := types.ListValue(
			types.ObjectType{AttrTypes: networkAttachmentAttrTypes},
			networkAttachments,
		)
		diags.Append(d...)
		state.NetworkAttachment = networkAttachmentList
	}

	// Map IP addresses to list (combine private and public IPs)
	allIPs := append(server.PrivateIPs, server.PublicIPs...)
	// Sort IP addresses to ensure consistent order and avoid spurious diffs
	sort.Strings(allIPs)
	ipVals := make([]attr.Value, len(allIPs))
	for i, ip := range allIPs {
		ipVals[i] = types.StringValue(ip)
	}
	ipList, d := types.ListValue(types.StringType, ipVals)
	diags.Append(d...)
	state.IPAddresses = ipList

	// User data and password are not returned by API for security
	state.UserData = types.StringNull()
	state.Password = types.StringNull()

	return state, diags
}

// waitForServerActive waits for the server to reach "active" using the SDK-provided waiter helper.
func waitForServerActive(ctx context.Context, serversClient *serversdk.Client, serverID string, timeout time.Duration) (*serversdk.ServerResource, error) {
	// Use context with timeout so the SDK waiter respects the configured duration.
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := vpscore.WaitForServerStatus(waitCtx, vpscore.ServerWaiterConfig{
		Client:       serversClient,
		ServerID:     serverID,
		TargetStatus: servermodels.ServerStatusActive,
	}); err != nil {
		return nil, fmt.Errorf("waiting for server to become active: %w", err)
	}

	// After waiter completes, fetch the latest server resource
	return serversClient.Get(ctx, serverID)
}

// waitForServerDeleted polls until server is deleted or timeout.
func waitForServerDeleted(ctx context.Context, client interface {
	Get(context.Context, string) (*serversdk.ServerResource, error)
}, serverID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for server to be deleted")
			}

			_, err := client.Get(ctx, serverID)
			if err != nil {
				// 404 error means server is deleted
				// This is the success condition
				return nil
			}

			// Server still exists, continue waiting
			continue
		}
	}
}
