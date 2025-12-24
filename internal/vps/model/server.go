// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package model

import (
	servermodels "github.com/Zillaforge/cloud-sdk/models/vps/servers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ServerResourceModel represents the Terraform state for a server resource.
type ServerResourceModel struct {
	// Required user-provided attributes
	Name              types.String `tfsdk:"name"`
	FlavorID          types.String `tfsdk:"flavor_id"`
	ImageID           types.String `tfsdk:"image_id"`
	NetworkAttachment types.List   `tfsdk:"network_attachment"` // List of NetworkAttachmentModel

	// Optional user-provided attributes
	Description    types.String `tfsdk:"description"`
	Keypair        types.String `tfsdk:"keypair"`
	Password       types.String `tfsdk:"password"`
	UserData       types.String `tfsdk:"user_data"`
	WaitForActive  types.Bool   `tfsdk:"wait_for_active"`
	WaitForDeleted types.Bool   `tfsdk:"wait_for_deleted"`

	// Computed attributes (read-only)
	ID          types.String `tfsdk:"id"`
	Status      types.String `tfsdk:"status"`
	IPAddresses types.List   `tfsdk:"ip_addresses"` // List of types.String
	CreatedAt   types.String `tfsdk:"created_at"`

	// Timeouts configuration
	Timeouts types.Object `tfsdk:"timeouts"` // TimeoutsModel
}

// NetworkAttachmentModel represents a network interface attachment.
type NetworkAttachmentModel struct {
	NetworkID        types.String `tfsdk:"network_id"`
	IPAddress        types.String `tfsdk:"ip_address"`
	Primary          types.Bool   `tfsdk:"primary"`
	SecurityGroupIDs types.List   `tfsdk:"security_group_ids"` // List of types.String
}

// TimeoutsModel for configurable operation timeouts.
type TimeoutsModel struct {
	Create types.String `tfsdk:"create"`
	Update types.String `tfsdk:"update"`
	Delete types.String `tfsdk:"delete"`
}

// UpdateContext contains server update request and network changes.
type UpdateContext struct {
	ServerUpdate     *servermodels.ServerUpdateRequest
	NetworkChanges   map[string]servermodels.ServerNICUpdateRequest
	NetworksToDelete []string
	NetworksToCreate []servermodels.ServerNICCreateRequest
	HasChanges       bool
}
