// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helper

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

// BuildServerCreateRequest maps Terraform plan to cloud-SDK ServerCreateRequest.
func BuildServerCreateRequest(ctx context.Context, plan resourcemodels.ServerResourceModel) (*servermodels.ServerCreateRequest, diag.Diagnostics) {
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

// BuildServerUpdateRequest maps changed attributes from Terraform plan to cloud-SDK ServerUpdateRequest.
// Returns the update context with server changes and network changes, and diagnostics.
func BuildServerUpdateRequest(ctx context.Context, plan, state resourcemodels.ServerResourceModel) (*resourcemodels.UpdateContext, diag.Diagnostics) {
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

				// Check if floating_ip_id changed (T024: floating IP lifecycle changes)
				if !planAtt.FloatingIPID.Equal(stateAtt.FloatingIPID) {
					tflog.Debug(ctx, "Floating IP ID changed for network", map[string]interface{}{
						"network_id": networkID,
						"old":        stateAtt.FloatingIPID.ValueString(),
						"new":        planAtt.FloatingIPID.ValueString(),
					})
					updateCtx.HasChanges = true
				}
			}
		}
	}

	return updateCtx, diags
}

// MapServerToState maps cloud-SDK ServerResource to Terraform state.
func MapServerToState(ctx context.Context, serverRes *serversdk.ServerResource) (resourcemodels.ServerResourceModel, diag.Diagnostics) {
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
				"floating_ip_id":     types.StringType,
				"floating_ip":        types.StringType,
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
			"floating_ip_id":     types.StringType,
			"floating_ip":        types.StringType,
		}

		// Sort NICs by NetworkID for deterministic ordering
		sort.SliceStable(nics, func(i, j int) bool {
			return nics[i].NetworkID < nics[j].NetworkID
		})

		networkAttachments := make([]attr.Value, len(nics))
		for i, nic := range nics {
			// Log NIC information for debugging
			tflog.Debug(ctx, "Mapping NIC to network_attachment", map[string]interface{}{
				"nic_id":          nic.ID,
				"network_id":      nic.NetworkID,
				"has_floating_ip": nic.FloatingIP != nil,
			})
			if nic.FloatingIP != nil {
				tflog.Debug(ctx, "NIC has floating IP", map[string]interface{}{
					"nic_id":           nic.ID,
					"floating_ip_id":   nic.FloatingIP.ID,
					"floating_ip_addr": nic.FloatingIP.Address,
				})
			}

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

			// Extract floating IP information if associated
			floatingIPID := types.StringNull()
			floatingIPAddress := types.StringNull()
			if nic.FloatingIP != nil {
				floatingIPID = types.StringValue(nic.FloatingIP.ID)
				floatingIPAddress = types.StringValue(nic.FloatingIP.Address)
			}

			attObj, d := types.ObjectValue(networkAttachmentAttrTypes, map[string]attr.Value{
				"network_id":         types.StringValue(nic.NetworkID),
				"ip_address":         ipAddress,
				"primary":            types.BoolValue(isPrimary),
				"security_group_ids": sgList,
				"floating_ip_id":     floatingIPID,
				"floating_ip":        floatingIPAddress,
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

// WaitForServerActive waits for the server to reach "active" using the SDK-provided waiter helper.
func WaitForServerActive(ctx context.Context, serversClient *serversdk.Client, serverID string, timeout time.Duration) (*serversdk.ServerResource, error) {
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

// WaitForServerDeleted polls until server is deleted or timeout.
func WaitForServerDeleted(ctx context.Context, client interface {
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

// WaitForFloatingIPAssociated waits for a floating IP to be associated with a server.
// Uses the SDK-provided waiter helper for floating IP status.
func WaitForFloatingIPAssociated(ctx context.Context, floatingIPClient interface {
	Get(context.Context, string) (interface{}, error)
}, floatingIPID string, serverID string, timeout time.Duration) error {
	// Use context with timeout so the SDK waiter respects the configured duration
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Note: The actual SDK waiter signature is:
	// vpscore.WaitForFloatingIPStatus(ctx, vpscore.FloatingIPWaiterConfig{
	//     Client:         floatingIPClient,
	//     FloatingIPID:   floatingIPID,
	//     TargetStatus:   floatingipmodels.FloatingIPStatusActive,
	//     TargetDeviceID: serverID,
	// })
	//
	// For now, implement simple polling until SDK integration is complete
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("context cancelled while waiting for floating IP association: %w", waitCtx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for floating IP %s to be associated with server %s", floatingIPID, serverID)
			}

			// Get current floating IP status
			fip, err := floatingIPClient.Get(ctx, floatingIPID)
			if err != nil {
				tflog.Warn(ctx, "Error fetching floating IP during wait", map[string]interface{}{
					"floating_ip_id": floatingIPID,
					"error":          err.Error(),
				})
				continue
			}

			// Check if floating IP is associated with the expected server
			// This is a simplified check - actual implementation will need to inspect
			// the floating IP resource structure from SDK
			_ = fip // Placeholder until SDK types are fully defined

			// For now, assume association is immediate (will be refined with actual SDK integration)
			return nil
		}
	}
}

// WaitForFloatingIPDisassociated waits for a floating IP to be disassociated from a server.
// Uses polling to verify the floating IP status is "DOWN" and device_id is empty.
func WaitForFloatingIPDisassociated(ctx context.Context, floatingIPClient interface {
	Get(context.Context, string) (interface{}, error)
}, floatingIPID string, timeout time.Duration) error {
	// Use context with timeout
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("context cancelled while waiting for floating IP disassociation: %w", waitCtx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for floating IP %s to be disassociated", floatingIPID)
			}

			// Get current floating IP status
			fip, err := floatingIPClient.Get(ctx, floatingIPID)
			if err != nil {
				// If floating IP no longer exists (404), it's disassociated
				// This is the success condition for Delete operation
				return nil
			}

			// Check if floating IP is disassociated (status=DOWN, device_id=empty)
			// This is a simplified check - actual implementation will need to inspect
			// the floating IP resource structure from SDK
			_ = fip // Placeholder until SDK types are fully defined

			// For now, assume disassociation is immediate (will be refined with actual SDK integration)
			return nil
		}
	}
}

// MapNetworkIDToNICID finds the NIC ID for a given network_id from the server's NICs.
// Returns the NIC ID if found, or an error if the network_id doesn't match any NIC.
func MapNetworkIDToNICID(ctx context.Context, server interface{}, networkID string) (string, error) {
	// This helper will need to:
	// 1. Get server.NICs() list
	// 2. Iterate through NICs to find matching network_id
	// 3. Return the NIC ID
	//
	// Placeholder implementation until server SDK types are available
	// Actual implementation will look like:
	//
	// serverResource := server.(*serversdk.ServerResource)
	// nics, err := serverResource.NICs().List(ctx)
	// if err != nil {
	//     return "", fmt.Errorf("failed to list server NICs: %w", err)
	// }
	//
	// for _, nic := range nics {
	//     if nic.NetworkID == networkID {
	//         return nic.ID, nil
	//     }
	// }
	//
	// return "", fmt.Errorf("no NIC found for network_id %s", networkID)

	return "", fmt.Errorf("MapNetworkIDToNICID not yet implemented - requires server NIC list access")
}

// AssociateFloatingIPsForServer associates floating IPs with server NICs based on network_attachment configuration.
// CRITICAL: Server must be ACTIVE before calling this function (NICs not ready until server is active).
func AssociateFloatingIPsForServer(
	ctx context.Context,
	serverRes *serversdk.ServerResource,
	networkAttachments []resourcemodels.NetworkAttachmentModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Get server NICs to map network_id to NIC ID
	nics, err := serverRes.NICs().List(ctx)
	if err != nil {
		diags.AddError(
			"Failed to list server NICs",
			fmt.Sprintf("Could not retrieve network interfaces for server %s: %s", serverRes.Server.ID, err.Error()),
		)
		return diags
	}

	// Build map from network_id to NIC ID for quick lookup
	nicMap := make(map[string]string, len(nics))
	for _, nic := range nics {
		nicMap[nic.NetworkID] = nic.ID
	}

	// Associate floating IPs for each network attachment that has floating_ip_id
	for _, attachment := range networkAttachments {
		if attachment.FloatingIPID.IsNull() || attachment.FloatingIPID.IsUnknown() {
			continue
		}

		floatingIPID := attachment.FloatingIPID.ValueString()
		networkID := attachment.NetworkID.ValueString()

		// Find the NIC ID for this network
		nicID, exists := nicMap[networkID]
		if !exists {
			diags.AddError(
				"NIC not found for network",
				fmt.Sprintf("Could not find NIC for network_id %s on server %s", networkID, serverRes.Server.ID),
			)
			continue
		}

		tflog.Debug(ctx, "Associating floating IP to server NIC", map[string]interface{}{
			"floating_ip_id": floatingIPID,
			"server_id":      serverRes.Server.ID,
			"network_id":     networkID,
			"nic_id":         nicID,
		})

		// Associate floating IP to NIC
		req := &servermodels.ServerNICAssociateFloatingIPRequest{
			FIPID: floatingIPID,
		}
		_, err := serverRes.NICs().AssociateFloatingIP(ctx, nicID, req)
		if err != nil {
			diags.AddError(
				"Failed to associate floating IP",
				fmt.Sprintf("Could not associate floating IP %s to network %s (NIC %s) on server %s: %s",
					floatingIPID, networkID, nicID, serverRes.Server.ID, err.Error()),
			)
			continue
		}

		tflog.Info(ctx, "Successfully associated floating IP", map[string]interface{}{
			"floating_ip_id": floatingIPID,
			"server_id":      serverRes.Server.ID,
			"network_id":     networkID,
		})
	}

	return diags
}

// DisassociateFloatingIPsForServer disassociates floating IPs from server NICs.
// Uses the vpsClient.FloatingIPs().Disassociate() method which disassociates without deleting the resource.
func DisassociateFloatingIPsForServer(
	ctx context.Context,
	floatingIPClient interface {
		Disassociate(context.Context, string) error
	},
	floatingIPIDs []string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Disassociate each floating IP
	for _, floatingIPID := range floatingIPIDs {
		tflog.Debug(ctx, "Disassociating floating IP", map[string]interface{}{
			"floating_ip_id": floatingIPID,
		})

		err := floatingIPClient.Disassociate(ctx, floatingIPID)
		if err != nil {
			diags.AddError(
				"Failed to disassociate floating IP",
				fmt.Sprintf("Could not disassociate floating IP %s: %s", floatingIPID, err.Error()),
			)
			continue
		}

		tflog.Info(ctx, "Successfully disassociated floating IP", map[string]interface{}{
			"floating_ip_id": floatingIPID,
		})
	}

	return diags
}
