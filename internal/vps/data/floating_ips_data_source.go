// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"context"
	"fmt"

	cloudsdk "github.com/Zillaforge/cloud-sdk"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/helper"
	"github.com/Zillaforge/terraform-provider-zillaforge/internal/vps/model"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FloatingIPsDataSource{}

// NewFloatingIPsDataSource creates a new instance of the floating IPs data source.
func NewFloatingIPsDataSource() datasource.DataSource {
	return &FloatingIPsDataSource{}
}

// FloatingIPsDataSource defines the data source implementation.
type FloatingIPsDataSource struct {
	client *cloudsdk.ProjectClient
}

func (d *FloatingIPsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_floating_ips"
}

func (d *FloatingIPsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries existing floating IPs (public IP addresses) from ZillaForge VPS. Use this data source to retrieve floating IP details by ID, name, IP address, or status. Returns a list of matching floating IPs, which can then be referenced in other resources. Supports client-side filtering with AND logic when multiple filters are specified.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Optional filter to query a specific floating IP by ID (UUID format). Returns a list containing a single floating IP if found, or an empty list if not found.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Optional filter to query floating IPs by exact name (case-sensitive match). Returns a list of matching floating IPs (typically 0 or 1).",
				Optional:            true,
			},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "Optional filter to query floating IPs by exact IP address (e.g., `203.0.113.42`). Returns a list containing a single floating IP if found.",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Optional filter to query floating IPs by status. Valid values: `ACTIVE`, `DOWN`, `PENDING`, `REJECTED`. Returns a list of all floating IPs with the specified status.",
				Optional:            true,
			},
			"floating_ips": schema.ListNestedAttribute{
				MarkdownDescription: "List of floating IP objects matching the filter criteria. Empty list if no matches found. Results are sorted by ID for deterministic ordering.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the floating IP (UUID format). Use this value to reference the floating IP in other resources.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Human-readable name for the floating IP. May be empty if not set during allocation.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Optional description providing context about the floating IP's purpose. May be empty if not set.",
							Computed:            true,
						},
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "The public IP address (e.g., `203.0.113.42`). This is the actual IP that can be used for network connectivity.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Current status of the floating IP. Possible values: `ACTIVE` (ready for use), `DOWN` (unavailable), `PENDING` (being provisioned), `REJECTED` (allocation failed).",
							Computed:            true,
						},
						"device_id": schema.StringAttribute{
							MarkdownDescription: "ID of the device (VPS instance) this floating IP is associated with. Null or empty if not associated with any device. Association is managed outside Terraform.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *FloatingIPsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	projectClient, ok := req.ProviderData.(*cloudsdk.ProjectClient)
	if ok {
		d.client = projectClient
	}
}

func (d *FloatingIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model.FloatingIPDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpsClient := d.client.VPS()

	// List all floating IPs from API
	tflog.Debug(ctx, "Listing all floating IPs from API")

	allFloatingIPs, err := vpsClient.FloatingIPs().List(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to List Floating IPs",
			fmt.Sprintf("Unable to list floating IPs: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Retrieved floating IPs from API", map[string]interface{}{
		"total_count": len(allFloatingIPs),
	})

	// Apply client-side filtering (handles all filter combinations with AND logic)
	filteredFloatingIPs := helper.FilterFloatingIPs(allFloatingIPs, &config)

	tflog.Debug(ctx, "Applied client-side filters", map[string]interface{}{
		"filtered_count":    len(filteredFloatingIPs),
		"has_id_filter":     !config.ID.IsNull(),
		"has_name_filter":   !config.Name.IsNull(),
		"has_ip_filter":     !config.IPAddress.IsNull(),
		"has_status_filter": !config.Status.IsNull(),
	})

	// Convert to model list (already sorted by ID in FilterFloatingIPs)
	floatingIPModels := make([]model.FloatingIPModel, 0, len(filteredFloatingIPs))
	for _, fip := range filteredFloatingIPs {
		floatingIPModels = append(floatingIPModels, helper.MapFloatingIPToModel(fip))
	}

	// Set state
	config.FloatingIPs = floatingIPModels

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)

	tflog.Info(ctx, "Successfully queried floating IPs", map[string]interface{}{
		"result_count": len(floatingIPModels),
	})
}
