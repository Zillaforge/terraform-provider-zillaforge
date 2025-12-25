// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package model

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FloatingIPResourceModel represents the Terraform state for a floating IP resource.
type FloatingIPResourceModel struct {
	// Optional user-provided attributes
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	// Computed attributes (read-only)
	ID        types.String `tfsdk:"id"`
	IPAddress types.String `tfsdk:"ip_address"`
	Status    types.String `tfsdk:"status"`
	DeviceID  types.String `tfsdk:"device_id"`
}

// FloatingIPDataSourceModel describes the data source config and results.
type FloatingIPDataSourceModel struct {
	// Optional filters (all are optional, AND logic when multiple specified)
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	IPAddress types.String `tfsdk:"ip_address"`
	Status    types.String `tfsdk:"status"`

	// Computed results (list of matching floating IPs)
	FloatingIPs []FloatingIPModel `tfsdk:"floating_ips"`
}

// FloatingIPModel represents a single floating IP in data source results.
type FloatingIPModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IPAddress   types.String `tfsdk:"ip_address"`
	Status      types.String `tfsdk:"status"`
	DeviceID    types.String `tfsdk:"device_id"`
}
